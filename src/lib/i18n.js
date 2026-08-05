/**
 * Copy table.
 *
 * Every user-facing string lives here rather than inline in JSX. That is the
 * whole requirement from spec §6: adding a Swahili table later means adding a
 * `sw` object below and a language switch, not touching a single component.
 *
 * Keys are grouped by screen or concern. Values may contain {placeholders},
 * which `t` interpolates.
 */

const en = {
  brand: {
    name: 'AfyaLink',
    namePrefix: 'Afya',
    nameSuffix: 'Link',
    tagline: 'Healing continues beyond the ward.',
  },

  common: {
    loading: 'Loading…',
    retry: 'Try again',
    cancel: 'Cancel',
    save: 'Save',
    saving: 'Saving…',
    close: 'Close',
    back: 'Back',
    signOut: 'Sign out',
    none: 'None recorded',
    notProvided: 'Not provided',
    somethingWentWrong: 'Something went wrong. Please try again.',
    sessionExpired: 'Your session has ended. Please sign in again.',
    today: 'Today',
    yes: 'Yes',
    no: 'No',
  },

  roles: {
    patient: 'Patient',
    caregiver: 'Caregiver',
    clinician: 'Clinician',
  },

  login: {
    heading: 'Welcome to {brand}',
    subtitle: 'Sign in to your account',
    emailLabel: 'Email address',
    emailPlaceholder: 'you@example.com',
    passwordLabel: 'Password',
    passwordPlaceholder: '••••••••',
    submit: 'Sign in',
    submitting: 'Signing in…',
    invalidCredentials: 'Invalid email or password.',
    rateLimited: 'Too many attempts. Please wait a moment and try again.',
    noSelfSignup:
      'Accounts are created by your clinician at discharge. Contact your care team if you cannot sign in.',
    panelHeading: 'Healing continues beyond the ward.',
    panelBody:
      'AfyaLink connects patients, clinicians, and caregivers across Kenya — so recovery never happens alone, no matter where you are.',
    statReadmissions: 'readmissions preventable',
    statWindow: 'critical post-discharge window',
    statCounties: 'counties covered',
  },

  nav: {
    dashboard: 'Dashboard',
    patients: 'Patients',
    alerts: 'Alerts',
    checkins: 'Check-ins',
    medications: 'Medications',
    home: 'Home',
  },

  clinician: {
    dashboardTitle: 'Clinician dashboard',
    dashboardSubtitle: 'Your patients and open alerts',
    patientListTitle: 'Your patients',
    patientListEmpty: 'You have no patients yet.',
    alertInboxTitle: 'Alert inbox',
    alertInboxEmpty: 'No open alerts. Everyone is on track.',
    showResolved: 'Show resolved',
    hideResolved: 'Hide resolved',
    openAlerts: 'Open alerts',
    // Phrased to avoid a plural: "{count} open" reads correctly at 1 and at 5,
    // which keeps the badge free of pluralisation rules that would have to be
    // reimplemented per language.
    openAlertCount: '{count} open',
    totalPatients: 'Patients',
    criticalAlerts: 'Critical',
    resolveAlert: 'Resolve',
    resolving: 'Resolving…',
    resolvedBadge: 'Resolved',
    patientDetailTitle: 'Patient record',
    diagnosis: 'Diagnosis',
    allergies: 'Allergies',
    emergencyContact: 'Emergency contact',
    dateOfBirth: 'Date of birth',
    dischargeRecords: 'Discharge records',
    dischargeEmpty: 'No discharge record yet.',
    dischargedOn: 'Discharged {date}',
    medications: 'Medications',
    medicationsEmpty: 'No medications prescribed.',
    checkinHistory: 'Check-in history',
    checkinHistoryEmpty: 'No check-ins scheduled yet.',
    clinicalNotes: 'Clinical notes',
    clinicalNotesEmpty: 'No notes yet.',
    addNote: 'Add a clinical note',
    notePlaceholder: 'Observation, plan, or follow-up…',
    saveNote: 'Save note',
    noteSaved: 'Note saved.',
  },

  caregiver: {
    dashboardTitle: 'Caregiver view',
    dashboardSubtitle: "Following {name}'s recovery",
    noPatient: 'You are not linked to a patient yet. Your care team will connect you.',
    // The list carries completed and missed check-ins alongside upcoming ones,
    // so the heading must not claim otherwise.
    checkins: 'Check-ins',
    medicationSchedule: 'Medication schedule',
    alertHistory: 'Alert history',
    alertHistoryEmpty: 'No alerts. Everything looks on track.',
    readOnlyNotice: 'This is a read-only view of your loved one’s recovery.',
  },

  patient: {
    dashboardTitle: 'Good day, {name}',
    dashboardSubtitle: "Here's your recovery overview",
    todaysCheckin: "Today's check-in",
    noCheckinDue: 'No check-in due right now. We will let you know when the next one opens.',
    startCheckin: 'Start check-in',
    todaysMedication: "Today's medication",
    medicationsEmpty: 'No medication scheduled.',
    markTaken: 'Mark as taken',
    marking: 'Recording…',
    recoveryDay: 'Recovery day',
    dosesTaken: 'Doses taken today',
    checkinsCompleted: 'Check-ins completed',
  },

  checkin: {
    title: 'Check-in',
    intro: 'Answer honestly — your clinician sees these answers.',
    progress: 'Question {current} of {total}',
    scaleHint: 'Slide or tap to rate from {min} to {max}',
    submit: 'Submit check-in',
    submitting: 'Submitting…',
    completeTitle: 'Check-in complete',
    completeBody: 'Thank you. Your answers have been recorded.',
    alertRaisedTitle: 'Your care team has been notified',
    alertRaisedBody:
      'Based on your answers, an alert was sent to your clinician. If you feel worse, seek care immediately.',
    alreadyCompleted: 'You have already completed this check-in.',
    unanswered: 'Please answer every question before submitting.',
  },

  medication: {
    everyHours: 'Every {hours} hours',
    scheduledFor: 'Scheduled {time}',
    takenAt: 'Taken {time}',
    statusTaken: 'Taken',
    statusPending: 'Pending',
    statusMissed: 'Missed',
  },

  checkinStatus: {
    pending: 'Pending',
    completed: 'Completed',
    missed: 'Missed',
  },

  severity: {
    low: 'Low',
    medium: 'Medium',
    high: 'High',
    critical: 'Critical',
  },

  landing: {
    heroTitle: 'Recovery does not end at discharge.',
    heroBody:
      'AfyaLink keeps patients, clinicians, and caregivers connected through the critical weeks after leaving hospital.',
    signIn: 'Sign in',
  },

  errors: {
    forbidden: 'You do not have access to this record.',
    notFound: 'We could not find what you were looking for.',
    network: 'Could not reach the server. Check your connection and try again.',
  },
}

const dictionaries = { en }

// Active language. A future switcher sets this (and re-renders); nothing else
// in the app needs to change.
let activeLanguage = 'en'

export function setLanguage(language) {
  if (dictionaries[language]) activeLanguage = language
}

export function availableLanguages() {
  return Object.keys(dictionaries)
}

/**
 * Look up a dot-separated key and interpolate {placeholders}.
 *
 * A missing key returns the key itself rather than throwing or rendering
 * blank — a visible `clinician.missingKey` in the UI is far easier to spot in
 * review than an empty element.
 */
export function t(key, vars) {
  const dictionary = dictionaries[activeLanguage] ?? en

  const value = key.split('.').reduce((node, part) => (node == null ? undefined : node[part]), dictionary)

  if (typeof value !== 'string') return key
  if (!vars) return value

  return value.replace(/\{(\w+)\}/g, (match, name) => (name in vars ? String(vars[name]) : match))
}
