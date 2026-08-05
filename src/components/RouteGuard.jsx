import { Navigate, useLocation } from 'react-router-dom'

import { useAuth } from '../lib/authContext'
import { homePathForRole } from '../lib/roles'
import { Spinner } from './ui'

/**
 * Gate a route on authentication and, optionally, on role.
 *
 * These guards are a usability measure, not a security boundary. Nothing here
 * protects data — the API authorises every request independently, and a user
 * who edits their way past this component still gets a 403 from the server.
 * The purpose is to send people somewhere sensible instead of showing them a
 * screen that will only ever render errors.
 */
export function RequireAuth({ roles, children }) {
  const { isAuthenticated, role, restoring } = useAuth()
  const location = useLocation()

  // The startup refresh has not settled yet. Redirecting now would sign out
  // every user on every page reload, because the access token is memory-only.
  if (restoring) {
    return (
      <div className="min-h-screen grid place-items-center">
        <Spinner />
      </div>
    )
  }

  if (!isAuthenticated) {
    // Remember where they were headed so login can return them there.
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  if (roles && !roles.includes(role)) {
    // Signed in, wrong section: send them to their own home rather than to
    // login, which would look like the session had failed.
    return <Navigate to={homePathForRole(role)} replace />
  }

  return children
}

/** Keep an already-signed-in user off the login screen. */
export function RedirectIfAuthenticated({ children }) {
  const { isAuthenticated, role, restoring } = useAuth()

  if (restoring) {
    return (
      <div className="min-h-screen grid place-items-center">
        <Spinner />
      </div>
    )
  }

  if (isAuthenticated) {
    return <Navigate to={homePathForRole(role)} replace />
  }

  return children
}
