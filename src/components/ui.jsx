/**
 * Shared primitives.
 *
 * These exist so screens compose rather than restate: the original codebase
 * repeated the same card, pill, and button rules across three ~700-line files.
 * Every visual decision here matches the palette in index.css.
 */

import { cx } from '../lib/cx'
import { t } from '../lib/i18n'

const BUTTON_VARIANTS = {
  primary: 'bg-teal text-white hover:bg-teal-dark disabled:bg-ink-faint',
  ghost: 'border-2 border-teal text-teal bg-transparent hover:bg-teal-light disabled:border-ink-faint disabled:text-ink-faint',
  subtle: 'bg-white border border-hairline text-ink hover:border-teal disabled:text-ink-faint',
  danger: 'bg-danger text-white hover:brightness-90 disabled:bg-ink-faint',
}

const BUTTON_SIZES = {
  sm: 'px-3 py-1.5 text-sm',
  md: 'px-4 py-2.5 text-[0.92rem]',
  lg: 'px-5 py-3 text-base w-full',
}

export function Button({ variant = 'primary', size = 'md', className, children, ...props }) {
  return (
    <button
      className={cx(
        'rounded-[10px] font-medium transition-colors disabled:cursor-not-allowed',
        BUTTON_VARIANTS[variant],
        BUTTON_SIZES[size],
        className,
      )}
      {...props}
    >
      {children}
    </button>
  )
}

export function Card({ className, children, ...props }) {
  return (
    <div className={cx('bg-white border border-hairline rounded-2xl p-6', className)} {...props}>
      {children}
    </div>
  )
}

export function SectionTitle({ children, action }) {
  return (
    <div className="flex items-center justify-between mb-4">
      <h2 className="text-[0.95rem] font-medium text-navy">{children}</h2>
      {action}
    </div>
  )
}

export function PageHeader({ title, subtitle, action }) {
  return (
    <header className="flex flex-wrap items-start justify-between gap-4 mb-8">
      <div>
        <h1 className="font-display text-[1.6rem] text-navy">{title}</h1>
        {subtitle && <p className="text-sm text-ink-muted mt-0.5">{subtitle}</p>}
      </div>
      {action}
    </header>
  )
}

export function Field({ label, hint, error, children, htmlFor }) {
  return (
    <div className="mb-4">
      <label htmlFor={htmlFor} className="block text-sm font-medium text-ink mb-1.5">
        {label}
        {hint && <span className="ml-1 text-xs font-light text-ink-muted">{hint}</span>}
      </label>
      {children}
      {error && (
        <p className="mt-1 text-xs text-danger" role="alert">
          {error}
        </p>
      )}
    </div>
  )
}

export function Input({ className, invalid, ...props }) {
  return (
    <input
      className={cx(
        'w-full px-4 py-2.5 rounded-[10px] bg-white text-[0.9rem] text-ink',
        'border-2 outline-none transition-colors placeholder:text-ink-faint',
        invalid ? 'border-danger' : 'border-hairline focus:border-teal',
        className,
      )}
      aria-invalid={invalid || undefined}
      {...props}
    />
  )
}

export function Textarea({ className, ...props }) {
  return (
    <textarea
      className={cx(
        'w-full px-4 py-2.5 rounded-[10px] bg-white text-[0.9rem] text-ink',
        'border-2 border-hairline outline-none transition-colors focus:border-teal',
        'placeholder:text-ink-faint resize-y min-h-24',
        className,
      )}
      {...props}
    />
  )
}

const BADGE_TONES = {
  teal: 'bg-teal-light text-teal-dark',
  amber: 'bg-amber-light text-amber',
  danger: 'bg-danger-light text-danger',
  info: 'bg-info-light text-info',
  neutral: 'bg-canvas text-ink-muted border border-hairline',
}

export function Badge({ tone = 'neutral', className, children }) {
  return (
    <span
      className={cx(
        'inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium',
        BADGE_TONES[tone],
        className,
      )}
    >
      {children}
    </span>
  )
}

/**
 * Severity is the one thing a clinician scans for, so it gets a deliberate
 * ramp: teal for low through red for critical.
 */
const SEVERITY_TONES = {
  low: 'info',
  medium: 'amber',
  high: 'amber',
  critical: 'danger',
}

export function SeverityBadge({ severity }) {
  return (
    <Badge tone={SEVERITY_TONES[severity] ?? 'neutral'}>
      {t(`severity.${severity}`)}
    </Badge>
  )
}

const STATUS_TONES = {
  taken: 'teal',
  completed: 'teal',
  pending: 'amber',
  missed: 'danger',
}

export function StatusPill({ status, label }) {
  return <Badge tone={STATUS_TONES[status] ?? 'neutral'}>{label}</Badge>
}

export function StatCard({ label, value, sub, badge }) {
  return (
    <div className="bg-white border border-hairline rounded-2xl p-5">
      <div className="text-[0.78rem] text-ink-muted mb-2">{label}</div>
      <div className="font-display text-[2rem] leading-none text-navy">{value}</div>
      {sub && <div className="text-xs text-ink-muted mt-1.5">{sub}</div>}
      {badge && <div className="mt-2">{badge}</div>}
    </div>
  )
}

export function ProgressBar({ value, label }) {
  const percent = Math.max(0, Math.min(100, Math.round(value)))
  return (
    <div className="mb-4">
      {label && (
        <div className="flex justify-between text-[0.8rem] mb-1.5">
          <span className="text-ink-muted">{label}</span>
          <span className="text-teal-dark font-medium">{percent}%</span>
        </div>
      )}
      <div
        className="h-2 bg-teal-light rounded-full overflow-hidden"
        role="progressbar"
        aria-valuenow={percent}
        aria-valuemin={0}
        aria-valuemax={100}
      >
        <div className="h-full bg-teal rounded-full transition-[width] duration-500" style={{ width: `${percent}%` }} />
      </div>
    </div>
  )
}

export function Avatar({ name, initials: text, className }) {
  return (
    <div
      className={cx(
        'shrink-0 rounded-full bg-teal text-white grid place-items-center text-sm font-medium',
        'size-9',
        className,
      )}
      aria-hidden="true"
      title={name}
    >
      {text}
    </div>
  )
}

export function Spinner({ label = t('common.loading') }) {
  return (
    <div className="flex items-center gap-3 text-sm text-ink-muted py-8" role="status">
      <span className="size-4 rounded-full border-2 border-teal border-t-transparent animate-spin" />
      {label}
    </div>
  )
}

export function EmptyState({ children }) {
  return <p className="text-sm text-ink-muted py-6">{children}</p>
}

export function ErrorBanner({ message, onRetry }) {
  if (!message) return null
  return (
    <div
      className="flex flex-wrap items-center justify-between gap-3 bg-danger-light border border-danger/20 text-danger rounded-xl px-4 py-3 text-sm mb-4"
      role="alert"
    >
      <span>{message}</span>
      {onRetry && (
        <button onClick={onRetry} className="underline font-medium">
          {t('common.retry')}
        </button>
      )}
    </div>
  )
}

export function Notice({ tone = 'teal', children }) {
  const tones = {
    teal: 'bg-teal-light text-teal-dark border-teal/20',
    amber: 'bg-amber-light text-amber border-amber/20',
    danger: 'bg-danger-light text-danger border-danger/20',
  }
  return (
    <div className={cx('rounded-xl border px-4 py-3 text-sm leading-relaxed', tones[tone])}>{children}</div>
  )
}
