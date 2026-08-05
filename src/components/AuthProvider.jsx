import { useCallback, useEffect, useMemo, useState } from 'react'

import { ApiError, api, setAccessToken, setSessionEndedHandler } from '../lib/api'
import { AuthContext } from '../lib/authContext'

export default function AuthProvider({ children }) {
  const [user, setUser] = useState(null)

  // `restoring` covers the startup refresh. Route guards must wait for it:
  // rendering before it settles would bounce an already-signed-in user to
  // /login on every page reload, because the access token is memory-only and
  // starts empty.
  const [restoring, setRestoring] = useState(true)

  const clearSession = useCallback(() => {
    setAccessToken(null)
    setUser(null)
  }, [])

  useEffect(() => {
    setSessionEndedHandler(clearSession)
  }, [clearSession])

  useEffect(() => {
    let cancelled = false

    // Attempt a silent refresh. The HttpOnly cookie survived the reload even
    // though the access token did not, so this restores the session without
    // asking for credentials again.
    api
      .refresh()
      .then((payload) => {
        if (!cancelled) setUser(payload.user)
      })
      .catch(() => {
        // No valid cookie: the user is simply signed out. Not an error.
        if (!cancelled) clearSession()
      })
      .finally(() => {
        if (!cancelled) setRestoring(false)
      })

    return () => {
      cancelled = true
    }
  }, [clearSession])

  const login = useCallback(async (email, password) => {
    const payload = await api.login(email, password)
    setAccessToken(payload.access_token)
    setUser(payload.user)
    return payload.user
  }, [])

  const logout = useCallback(async () => {
    try {
      await api.logout()
    } catch (error) {
      // A failed logout call still clears local state — leaving the user
      // apparently signed in because the network blipped would be worse than
      // a refresh token that outlives its cookie.
      if (!(error instanceof ApiError)) throw error
    } finally {
      clearSession()
    }
  }, [clearSession])

  const value = useMemo(
    () => ({
      user,
      role: user?.role ?? null,
      isAuthenticated: Boolean(user),
      restoring,
      login,
      logout,
    }),
    [user, restoring, login, logout],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
