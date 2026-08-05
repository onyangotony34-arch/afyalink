import { createContext, useContext } from 'react'

/**
 * Auth context and its hook.
 *
 * Kept apart from the provider component so this module exports no components,
 * which is what lets Fast Refresh work on the provider file.
 */
export const AuthContext = createContext(null)

export function useAuth() {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used inside an AuthProvider')
  }
  return context
}
