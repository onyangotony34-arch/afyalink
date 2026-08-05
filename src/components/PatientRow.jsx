import { Link } from 'react-router-dom'

import { formatDate, initials } from '../lib/format'
import { t } from '../lib/i18n'
import { Avatar, Badge } from './ui'

/** One patient in the clinician's list. */
export default function PatientRow({ patient, openAlertCount = 0 }) {
  return (
    <Link
      to={`/clinician/patients/${patient.id}`}
      className="flex items-center gap-3 py-3.5 border-b border-hairline last:border-b-0 hover:bg-canvas -mx-2 px-2 rounded-lg transition-colors"
    >
      <Avatar name={patient.full_name} initials={initials(patient.full_name)} />

      <div className="min-w-0 flex-1">
        <div className="text-[0.9rem] font-medium text-ink truncate">{patient.full_name}</div>
        <div className="text-xs text-ink-muted truncate">
          {/* The diagnosis is only present for roles authorised to see it; the
              date of birth is the sensible fallback line. */}
          {patient.diagnosis ?? `${t('clinician.dateOfBirth')}: ${formatDate(patient.dob)}`}
        </div>
      </div>

      {openAlertCount > 0 && (
        <Badge tone="danger">{t('clinician.openAlertCount', { count: openAlertCount })}</Badge>
      )}
    </Link>
  )
}
