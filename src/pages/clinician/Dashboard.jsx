import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'

import AlertCard from '../../components/AlertCard'
import PatientRow from '../../components/PatientRow'
import { Card, EmptyState, ErrorBanner, PageHeader, SectionTitle, Spinner, StatCard } from '../../components/ui'
import { api } from '../../lib/api'
import { t } from '../../lib/i18n'
import { describeError, useResource } from '../../lib/useResource'

/**
 * Clinician home: patient list plus the open-alert inbox.
 *
 * The alert list arrives already sorted by the API (unresolved first, then
 * severity descending), so the ordering that matters clinically is decided
 * once, in SQL, rather than re-derived in every client that renders it.
 */
export default function ClinicianDashboard() {
  const patients = useResource((signal) => api.listPatients(signal), [])
  const alerts = useResource((signal) => api.listAlerts({}, signal), [])

  const [resolvingId, setResolvingId] = useState(null)
  const [resolveError, setResolveError] = useState(null)

  // Memoised so the per-patient alert counts below are not recomputed from a
  // fresh [] literal on every render.
  const alertList = useMemo(() => alerts.data?.alerts ?? [], [alerts.data])
  const patientList = patients.data?.patients ?? []

  // Open alerts per patient, so the patient list can carry a count badge.
  const openAlertsByPatient = useMemo(() => {
    const counts = new Map()
    for (const alert of alertList) {
      if (alert.resolved) continue
      counts.set(alert.patient_id, (counts.get(alert.patient_id) ?? 0) + 1)
    }
    return counts
  }, [alertList])

  const criticalCount = alertList.filter((alert) => !alert.resolved && alert.severity === 'critical').length

  const handleResolve = async (alert) => {
    setResolvingId(alert.id)
    setResolveError(null)
    try {
      await api.resolveAlert(alert.id)
      alerts.reload()
    } catch (cause) {
      setResolveError(describeError(cause))
    } finally {
      setResolvingId(null)
    }
  }

  return (
    <>
      <PageHeader title={t('clinician.dashboardTitle')} subtitle={t('clinician.dashboardSubtitle')} />

      <div className="grid grid-cols-2 lg:grid-cols-3 gap-4 mb-7">
        <StatCard label={t('clinician.totalPatients')} value={patientList.length} />
        <StatCard
          label={t('clinician.openAlerts')}
          value={alertList.filter((alert) => !alert.resolved).length}
        />
        <StatCard label={t('clinician.criticalAlerts')} value={criticalCount} />
      </div>

      <div className="grid lg:grid-cols-[1fr_380px] gap-6 items-start">
        <Card>
          <SectionTitle>{t('clinician.patientListTitle')}</SectionTitle>

          {patients.loading ? (
            <Spinner />
          ) : patients.error ? (
            <ErrorBanner message={describeError(patients.error)} onRetry={patients.reload} />
          ) : patientList.length === 0 ? (
            <EmptyState>{t('clinician.patientListEmpty')}</EmptyState>
          ) : (
            patientList.map((patient) => (
              <PatientRow
                key={patient.id}
                patient={patient}
                openAlertCount={openAlertsByPatient.get(patient.id) ?? 0}
              />
            ))
          )}
        </Card>

        <Card>
          <SectionTitle
            action={
              <Link to="/clinician/alerts" className="text-[0.78rem] text-teal hover:text-teal-dark">
                {t('nav.alerts')} →
              </Link>
            }
          >
            {t('clinician.alertInboxTitle')}
          </SectionTitle>

          <ErrorBanner message={resolveError} />

          {alerts.loading ? (
            <Spinner />
          ) : alerts.error ? (
            <ErrorBanner message={describeError(alerts.error)} onRetry={alerts.reload} />
          ) : alertList.length === 0 ? (
            <EmptyState>{t('clinician.alertInboxEmpty')}</EmptyState>
          ) : (
            <div className="flex flex-col gap-3">
              {alertList.slice(0, 6).map((alert) => (
                <AlertCard
                  key={alert.id}
                  alert={alert}
                  onResolve={handleResolve}
                  resolving={resolvingId === alert.id}
                  patientHref={`/clinician/patients/${alert.patient_id}`}
                />
              ))}
            </div>
          )}
        </Card>
      </div>
    </>
  )
}
