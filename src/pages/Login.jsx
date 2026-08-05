import { useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'

import { ApiError } from '../lib/api'
import { useAuth } from '../lib/authContext'
import { homePathForRole } from '../lib/roles'
import { t } from '../lib/i18n'
import { Button, Field, Input, Notice } from '../components/ui'

/**
 * Email + password sign-in.
 *
 * This replaces the prototype's phone/OTP flow, hospital-administrator tab, and
 * profession picker. The MVP has exactly three roles and no public self-signup:
 * a clinician creates patient and caregiver accounts at discharge, so there is
 * nothing to register here and no role for the user to choose — the API returns
 * it, and the redirect follows from that.
 */
export default function Login() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState(null)
  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = async (event) => {
    event.preventDefault()
    if (submitting) return

    setSubmitting(true)
    setError(null)

    try {
      const user = await login(email, password)

      // Return the user to whatever they were trying to reach, falling back to
      // their role's home.
      const intended = location.state?.from?.pathname
      navigate(intended ?? homePathForRole(user.role), { replace: true })
    } catch (cause) {
      if (cause instanceof ApiError && cause.isRateLimited) {
        setError(t('login.rateLimited'))
      } else if (cause instanceof ApiError && cause.status === 401) {
        setError(t('login.invalidCredentials'))
      } else if (cause instanceof ApiError && cause.status === 0) {
        setError(t('errors.network'))
      } else {
        setError(t('common.somethingWentWrong'))
      }
      setSubmitting(false)
    }
  }

  return (
    <div className="min-h-screen grid md:grid-cols-2">
      {/* Brand panel — hidden on small screens, where the form is the priority */}
      <aside className="hidden md:flex flex-col justify-between bg-navy p-12 relative overflow-hidden">
        <div
          className="absolute -top-24 -left-24 size-96 rounded-full bg-teal/12"
          aria-hidden="true"
        />
        <div
          className="absolute -bottom-20 -right-20 size-72 rounded-full bg-teal/8"
          aria-hidden="true"
        />

        <div className="relative font-display text-[1.8rem] text-white">
          {t('brand.namePrefix')}
          <span className="text-teal">{t('brand.nameSuffix')}</span>
        </div>

        <div className="relative">
          <h2 className="font-display text-[2rem] text-white leading-tight mb-4">
            {t('login.panelHeading')}
          </h2>
          <p className="text-white/55 text-[0.92rem] leading-relaxed font-light">{t('login.panelBody')}</p>
        </div>

        <div className="relative flex flex-wrap gap-6">
          {[
            { value: '60%', label: t('login.statReadmissions') },
            { value: '72hrs', label: t('login.statWindow') },
            { value: '47', label: t('login.statCounties') },
          ].map((stat) => (
            <div key={stat.label}>
              <span className="block font-display text-[1.6rem] text-teal">{stat.value}</span>
              <p className="text-[0.72rem] text-white/40 mt-0.5">{stat.label}</p>
            </div>
          ))}
        </div>
      </aside>

      {/* Form */}
      <main className="flex items-center justify-center px-6 py-10">
        <div className="w-full max-w-[430px]">
          <h1 className="font-display text-[1.8rem] text-navy mb-1">
            {t('login.heading', { brand: t('brand.name') })}
          </h1>
          <p className="text-[0.88rem] text-ink-muted font-light mb-7">{t('login.subtitle')}</p>

          <form onSubmit={handleSubmit} noValidate>
            <Field label={t('login.emailLabel')} htmlFor="email">
              <Input
                id="email"
                type="email"
                autoComplete="username"
                required
                placeholder={t('login.emailPlaceholder')}
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                invalid={Boolean(error)}
              />
            </Field>

            <Field label={t('login.passwordLabel')} htmlFor="password">
              <Input
                id="password"
                type="password"
                autoComplete="current-password"
                required
                placeholder={t('login.passwordPlaceholder')}
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                invalid={Boolean(error)}
              />
            </Field>

            {error && (
              <div className="mb-4">
                <Notice tone="danger">{error}</Notice>
              </div>
            )}

            <Button type="submit" size="lg" disabled={submitting || !email || !password}>
              {submitting ? t('login.submitting') : t('login.submit')}
            </Button>
          </form>

          <p className="mt-6 text-[0.8rem] text-ink-muted leading-relaxed text-center">
            {t('login.noSelfSignup')}
          </p>
        </div>
      </main>
    </div>
  )
}
