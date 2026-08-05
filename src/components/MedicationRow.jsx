import { formatTime } from '../lib/format'
import { t } from '../lib/i18n'
import { cx } from '../lib/cx'
import { Button, StatusPill } from './ui'

const ICON_BY_STATUS = {
  taken: '✅',
  missed: '⚠️',
  pending: '💊',
}

const ICON_TONE = {
  taken: 'bg-teal-light',
  missed: 'bg-danger-light',
  pending: 'bg-info-light',
}

/**
 * One scheduled dose.
 *
 * `onMarkTaken` is only passed on the patient's own screens — clinicians and
 * caregivers see the same row without the action, because the API refuses to
 * let them record a dose on the patient's behalf.
 */
export default function DoseRow({ medication, log, onMarkTaken, marking }) {
  const status = log?.status ?? 'pending'

  return (
    <div className="flex items-center gap-3 py-3.5 border-b border-hairline last:border-b-0">
      <div className={cx('size-10 rounded-[10px] grid place-items-center text-lg shrink-0', ICON_TONE[status])}>
        <span aria-hidden="true">{ICON_BY_STATUS[status]}</span>
      </div>

      <div className="min-w-0 flex-1">
        <div className="text-[0.9rem] font-medium text-ink truncate">
          {medication.name} {medication.dosage}
        </div>
        <div className="text-xs text-ink-muted">
          {log
            ? status === 'taken' && log.taken_at
              ? t('medication.takenAt', { time: formatTime(log.taken_at) })
              : t('medication.scheduledFor', { time: formatTime(log.scheduled_at) })
            : t('medication.everyHours', { hours: medication.frequency_hours })}
        </div>
      </div>

      {onMarkTaken && status !== 'taken' ? (
        <Button size="sm" onClick={() => onMarkTaken(medication, log)} disabled={marking}>
          {marking ? t('patient.marking') : t('patient.markTaken')}
        </Button>
      ) : (
        <StatusPill status={status} label={t(`medication.status${capitalise(status)}`)} />
      )}
    </div>
  )
}

function capitalise(value) {
  return value.charAt(0).toUpperCase() + value.slice(1)
}
