/** The three roles the API recognises. There is no admin, nurse, or CHW role. */
export const ROLES = {
  patient: 'patient',
  caregiver: 'caregiver',
  clinician: 'clinician',
}

/** Where each role lands after signing in. */
export function homePathForRole(role) {
  switch (role) {
    case ROLES.clinician:
      return '/clinician'
    case ROLES.caregiver:
      return '/caregiver'
    case ROLES.patient:
      return '/patient'
    default:
      return '/login'
  }
}
