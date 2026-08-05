import { NavLink, useNavigate } from 'react-router-dom'

import { useAuth } from '../lib/authContext'
import { ROLES } from '../lib/roles'
import { t } from '../lib/i18n'
import { initials } from '../lib/format'
import { cx } from '../lib/cx'
import { Avatar } from './ui'

/**
 * Navigation per role.
 *
 * Each role only ever sees links to what it can actually reach, so the shell
 * never offers a caregiver a route that would answer 403.
 */
const NAV_BY_ROLE = {
  [ROLES.clinician]: [
    { to: '/clinician', end: true, icon: '🏠', labelKey: 'nav.dashboard', shortKey: 'nav.home' },
    { to: '/clinician/alerts', icon: '🚨', labelKey: 'nav.alerts', shortKey: 'nav.alerts' },
  ],
  [ROLES.caregiver]: [
    { to: '/caregiver', end: true, icon: '🏠', labelKey: 'nav.dashboard', shortKey: 'nav.home' },
  ],
  [ROLES.patient]: [
    { to: '/patient', end: true, icon: '🏠', labelKey: 'nav.dashboard', shortKey: 'nav.home' },
    { to: '/patient/checkin', icon: '📋', labelKey: 'nav.checkins', shortKey: 'nav.checkins' },
  ],
}

function Wordmark({ className }) {
  return (
    <span className={cx('font-display text-white', className)}>
      {t('brand.namePrefix')}
      <span className="text-teal">{t('brand.nameSuffix')}</span>
    </span>
  )
}

export default function AppShell({ children }) {
  const { user, role, logout } = useAuth()
  const navigate = useNavigate()

  const items = NAV_BY_ROLE[role] ?? []

  const handleSignOut = async () => {
    await logout()
    navigate('/login', { replace: true })
  }

  return (
    <div className="min-h-screen md:grid md:grid-cols-[240px_1fr]">
      {/* Desktop sidebar */}
      <aside className="hidden md:flex flex-col sticky top-0 h-screen bg-navy py-8">
        <Wordmark className="text-2xl px-6 mb-8" />

        <nav className="flex-1" aria-label={t('nav.dashboard')}>
          {items.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={({ isActive }) =>
                cx(
                  'flex items-center gap-2.5 px-6 py-3 text-[0.88rem] transition-colors border-l-[3px]',
                  isActive
                    ? 'text-white border-teal bg-teal/10'
                    : 'text-white/50 border-transparent hover:text-white hover:bg-white/5',
                )
              }
            >
              <span className="w-5 text-center" aria-hidden="true">
                {item.icon}
              </span>
              {t(item.labelKey)}
            </NavLink>
          ))}
        </nav>

        <div className="px-6 pt-6 border-t border-white/10">
          <div className="flex items-center gap-2.5 mb-4">
            <Avatar name={user?.email} initials={initials(user?.email)} />
            <div className="min-w-0">
              <div className="text-[0.85rem] text-white font-medium truncate">{user?.email}</div>
              <div className="text-xs text-white/40">{t(`roles.${role}`)}</div>
            </div>
          </div>
          <button
            onClick={handleSignOut}
            className="text-[0.82rem] text-white/50 hover:text-white transition-colors"
          >
            {t('common.signOut')}
          </button>
        </div>
      </aside>

      {/* Mobile top bar — the sidebar's identity and sign-out still need a home
          on small screens, where the bottom nav only carries navigation. */}
      <div className="md:hidden flex items-center justify-between bg-navy px-4 py-3">
        <Wordmark className="text-xl" />
        <button onClick={handleSignOut} className="text-[0.82rem] text-white/60">
          {t('common.signOut')}
        </button>
      </div>

      {/* pb-24 on mobile keeps the last card clear of the fixed bottom nav. */}
      <main className="px-4 py-6 pb-24 md:px-10 md:py-8 md:pb-8 overflow-y-auto">{children}</main>

      {/* Mobile bottom nav */}
      <nav
        className="md:hidden fixed bottom-0 inset-x-0 z-50 bg-navy border-t border-white/10 pt-2 pb-3"
        aria-label={t('nav.dashboard')}
      >
        <div className="flex justify-around">
          {items.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={({ isActive }) =>
                cx(
                  'flex flex-col items-center gap-0.5 px-4 py-1 text-[0.7rem] transition-colors',
                  isActive ? 'text-teal' : 'text-white/50',
                )
              }
            >
              <span className="text-xl" aria-hidden="true">
                {item.icon}
              </span>
              {t(item.shortKey)}
            </NavLink>
          ))}
        </div>
      </nav>
    </div>
  )
}
