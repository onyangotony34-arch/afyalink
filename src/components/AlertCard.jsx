import { Link } from 'react-router-dom'

import { formatDateTime } from '../lib/format'
import { t } from '../lib/i18n'
import { cx } from '../lib/cx'
import { Badge, Button, SeverityBadge } from './ui'

/**
 * One alert in the inbox.
 *
 * The left edge is colour-coded by severity so a clinician can triage the list
 * peripherally, without reading each message.
 */
const EDGE_BY_SEVERITY = {
  low: 'border-l-info',
  medium: 'border-l-amber',
  high: 'border-l-amber',
  critical: 'border-l-danger',
}

export default function AlertCard({ alert, onResolve, resolving, patientHref }) {
  const isResolved = alert.resolved

  return (
    <article
      className={cx(
        'bg-white border border-hairline border-l-4 rounded-xl p-4',
        EDGE_BY_SEVERITY[alert.severity] ?? 'border-l-hairline',
        isResolved && 'opacity-60',
      )}
    >
      <div className="flex flex-wrap items-center gap-2 mb-2">
        <SeverityBadge severity={alert.severity} />
        {isResolved && <Badge tone="teal">{t('clinician.resolvedBadge')}</Badge>}
        <span className="text-xs text-ink-muted ml-auto">{formatDateTime(alert.created_at)}</span>
      </div>

      <p className="text-[0.9rem] text-ink leading-relaxed mb-1">{alert.message}</p>

      <div className="flex flex-wrap items-center justify-between gap-2 mt-3">
        {patientHref ? (
          <Link to={patientHref} className="text-sm font-medium text-teal hover:text-teal-dark">
            {alert.patient_full_name}
          </Link>
        ) : (
          <span className="text-sm font-medium text-ink">{alert.patient_full_name}</span>
        )}

        {onResolve && !isResolved && (
          <Button size="sm" variant="subtle" onClick={() => onResolve(alert)} disabled={resolving}>
            {resolving ? t('clinician.resolving') : t('clinician.resolveAlert')}
          </Button>
        )}
      </div>
    </article>
  )
}
