import AlertCard from '../../components/AlertCard'
import DoseRow from '../../components/MedicationRow'
import {
  Card,
  EmptyState,
  ErrorBanner,
  Notice,
  PageHeader,
  SectionTitle,
  Spinner,
  StatusPill,
} from '../../components/ui'
import { api } from '../../lib/api'
import { formatDate, formatDateTime } from '../../lib/format'
import { t } from '../../lib/i18n'
import { describeError, useResource } from '../../lib/useResource'

/**
 * The caregiver's read-only view of their linked patient.
 *
 * Nothing on this screen mutates anything: no resolve button on alerts, no
 * mark-as-taken on doses, no note entry. That mirrors the API, which refuses
 * those actions for a caregiver regardless of what the UI offers, so the two
 * cannot drift into a screen full of buttons that only produce 403s.
 *
 * The patient comes from GET /api/patients, which the API already scopes to
 * the caregiver's linked patient — the client never filters a wider list.
 */
export default function CaregiverDashboard() {
  const patients = useResource((signal) => api.listPatients(signal), [])

  if (patients.loading) return <Spinner />
  if (patients.error) {
    return <ErrorBanner message={describeError(patients.error)} onRetry={patients.reload} />
  }

  const patient = patients.data?.patients?.[0]
  if (!patient) {
    return (
      <>
        <PageHeader title={t('caregiver.dashboardTitle')} />
        <Notice tone="amber">{t('caregiver.noPatient')}</Notice>
      </>
    )
  }

  return <CaregiverPatientView patient={patient} />
}

function CaregiverPatientView({ patient }) {
  const checkins = useResource((signal) => api.listCheckins(patient.id, signal), [patient.id])
  const medications = useResource((signal) => api.listMedications(patient.id, signal), [patient.id])
  const alerts = useResource((signal) => api.listAlerts({ includeResolved: true }, signal), [])

  const checkinList = checkins.data?.checkins ?? []
  const medicationList = medications.data?.medications ?? []
  const alertList = alerts.data?.alerts ?? []

  return (
    <>
      <PageHeader
        title={t('caregiver.dashboardTitle')}
        subtitle={t('caregiver.dashboardSubtitle', { name: patient.full_name })}
      />

      <div className="mb-6">
        <Notice>{t('caregiver.readOnlyNotice')}</Notice>
      </div>

      <div className="grid lg:grid-cols-2 gap-6 items-start">
        <Card>
          <SectionTitle>{t('caregiver.checkins')}</SectionTitle>

          {checkins.loading ? (
            <Spinner />
          ) : checkins.error ? (
            <ErrorBanner message={describeError(checkins.error)} onRetry={checkins.reload} />
          ) : checkinList.length === 0 ? (
            <EmptyState>{t('clinician.checkinHistoryEmpty')}</EmptyState>
          ) : (
            checkinList.map((checkin) => (
              <div
                key={checkin.id}
                className="flex items-center gap-3 py-3 border-b border-hairline last:border-b-0"
              >
                <div className="min-w-0 flex-1">
                  <div className="text-[0.9rem] text-ink truncate">{checkin.template_name}</div>
                  <div className="text-xs text-ink-muted">{formatDateTime(checkin.scheduled_at)}</div>
                </div>
                <StatusPill status={checkin.status} label={t(`checkinStatus.${checkin.status}`)} />
              </div>
            ))
          )}
        </Card>

        <Card>
          <SectionTitle>{t('caregiver.medicationSchedule')}</SectionTitle>

          {medications.loading ? (
            <Spinner />
          ) : medications.error ? (
            <ErrorBanner message={describeError(medications.error)} onRetry={medications.reload} />
          ) : medicationList.length === 0 ? (
            <EmptyState>{t('clinician.medicationsEmpty')}</EmptyState>
          ) : (
            medicationList.map((medication) =>
              medication.logs.length > 0 ? (
                medication.logs.map((log) => (
                  <DoseRow key={log.id} medication={medication} log={log} />
                ))
              ) : (
                <DoseRow key={medication.id} medication={medication} log={null} />
              ),
            )
          )}
        </Card>

        <Card className="lg:col-span-2">
          <SectionTitle>{t('caregiver.alertHistory')}</SectionTitle>

          {alerts.loading ? (
            <Spinner />
          ) : alerts.error ? (
            <ErrorBanner message={describeError(alerts.error)} onRetry={alerts.reload} />
          ) : alertList.length === 0 ? (
            <EmptyState>{t('caregiver.alertHistoryEmpty')}</EmptyState>
          ) : (
            <div className="flex flex-col gap-3">
              {alertList.map((alert) => (
                // No onResolve: resolving is a clinician action.
                <AlertCard key={alert.id} alert={alert} />
              ))}
            </div>
          )}
        </Card>
      </div>

      <p className="text-xs text-ink-muted mt-6">
        {patient.full_name} · {t('clinician.dateOfBirth')}: {formatDate(patient.dob)}
      </p>
    </>
  )
}
