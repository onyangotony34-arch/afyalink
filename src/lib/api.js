/**
 * API client.
 *
 * Two rules shape this file:
 *
 *  1. The access token lives in a module-scoped variable and nowhere else.
 *     Not localStorage, not sessionStorage, not a cookie the page can read —
 *     so a successful XSS cannot lift a long-lived credential out of storage.
 *     The cost is that a page reload loses it, which is what the silent
 *     refresh on startup is for.
 *
 *  2. The refresh token is never touched by JavaScript at all. It rides in an
 *     HttpOnly cookie the browser attaches to /api/auth on its own.
 */

const BASE_URL = import.meta.env.VITE_API_URL ?? '/api'

let accessToken = null

// Callback invoked when the session ends unrecoverably, so the auth context
// can clear its state and the guards can redirect to /login.
let onSessionEnded = () => {}

export function setAccessToken(token) {
  accessToken = token
}

export function getAccessToken() {
  return accessToken
}

export function setSessionEndedHandler(handler) {
  onSessionEnded = handler
}

/** Error carrying the HTTP status and the backend's stable error code. */
export class ApiError extends Error {
  constructor(message, { status, code, fields } = {}) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.fields = fields ?? []
  }

  get isForbidden() {
    return this.status === 403
  }

  get isConflict() {
    return this.status === 409
  }

  get isRateLimited() {
    return this.status === 429
  }
}

async function parseBody(response) {
  const text = await response.text()
  if (!text) return null
  try {
    return JSON.parse(text)
  } catch {
    return null
  }
}

async function rawRequest(path, { method = 'GET', body, signal } = {}) {
  const headers = {}
  if (accessToken) headers.Authorization = `Bearer ${accessToken}`
  if (body !== undefined) headers['Content-Type'] = 'application/json'

  let response
  try {
    response = await fetch(`${BASE_URL}${path}`, {
      method,
      headers,
      // Required so the browser sends the HttpOnly refresh cookie.
      credentials: 'include',
      body: body === undefined ? undefined : JSON.stringify(body),
      signal,
    })
  } catch (cause) {
    if (cause?.name === 'AbortError') throw cause
    throw new ApiError('network', { status: 0, code: 'network_error' })
  }

  const payload = await parseBody(response)

  if (!response.ok) {
    throw new ApiError(payload?.error ?? response.statusText, {
      status: response.status,
      code: payload?.code,
      fields: payload?.fields,
    })
  }

  return payload
}

// A single in-flight refresh shared by every caller.
//
// Without this, a dashboard firing four parallel requests on a stale token
// would attempt four refreshes. Because the backend rotates on every use and
// treats a reused token as theft, those extra attempts would be flagged as a
// replay and revoke the user's whole session — a self-inflicted logout.
let refreshInFlight = null

function refreshSession() {
  if (!refreshInFlight) {
    refreshInFlight = rawRequest('/auth/refresh', { method: 'POST' })
      .then((payload) => {
        setAccessToken(payload.access_token)
        return payload
      })
      .finally(() => {
        refreshInFlight = null
      })
  }
  return refreshInFlight
}

/**
 * Perform an API request, refreshing the access token once on a 401.
 *
 * `retryOnUnauthorized` is false for the auth endpoints themselves, so a
 * failed login cannot recurse into a refresh attempt.
 */
export async function request(path, options = {}) {
  const { retryOnUnauthorized = true, ...rest } = options

  try {
    return await rawRequest(path, rest)
  } catch (error) {
    if (!(error instanceof ApiError) || error.status !== 401 || !retryOnUnauthorized) {
      throw error
    }

    try {
      await refreshSession()
    } catch {
      setAccessToken(null)
      onSessionEnded()
      throw error
    }

    return rawRequest(path, rest)
  }
}

/* ---- Endpoint wrappers ----------------------------------------------- */

export const api = {
  // Auth
  login: (email, password) =>
    request('/auth/login', {
      method: 'POST',
      body: { email, password },
      retryOnUnauthorized: false,
    }),

  refresh: () => refreshSession(),

  logout: () => request('/auth/logout', { method: 'POST', retryOnUnauthorized: false }),

  me: () => request('/auth/me'),

  register: (payload) => request('/auth/register', { method: 'POST', body: payload }),

  // Patients
  listPatients: (signal) => request('/patients', { signal }),

  getPatient: (patientId, signal) => request(`/patients/${patientId}`, { signal }),

  createPatient: (payload) => request('/patients', { method: 'POST', body: payload }),

  listDischarges: (patientId, signal) => request(`/patients/${patientId}/discharge`, { signal }),

  createDischarge: (patientId, payload) =>
    request(`/patients/${patientId}/discharge`, { method: 'POST', body: payload }),

  // Medications
  listMedications: (patientId, signal) => request(`/patients/${patientId}/medications`, { signal }),

  createMedication: (patientId, payload) =>
    request(`/patients/${patientId}/medications`, { method: 'POST', body: payload }),

  markDoseTaken: (medicationId, logId) =>
    request(`/medications/${medicationId}/logs/${logId}/taken`, { method: 'POST' }),

  // Check-ins
  listCheckins: (patientId, signal) => request(`/patients/${patientId}/checkins`, { signal }),

  respondToCheckin: (checkinId, answers) =>
    request(`/checkins/${checkinId}/respond`, { method: 'POST', body: { answers } }),

  // Alerts
  listAlerts: ({ includeResolved = false } = {}, signal) =>
    request(`/alerts${includeResolved ? '?include_resolved=true' : ''}`, { signal }),

  resolveAlert: (alertId) => request(`/alerts/${alertId}/resolve`, { method: 'PATCH' }),

  // Clinical notes
  listNotes: (patientId, signal) => request(`/patients/${patientId}/notes`, { signal }),

  createNote: (patientId, body) => request(`/patients/${patientId}/notes`, { method: 'POST', body: { body } }),
}
