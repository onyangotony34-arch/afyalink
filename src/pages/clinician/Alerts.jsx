import { useState } from 'react'

import AlertCard from '../../components/AlertCard'
import { Button, Card, EmptyState, ErrorBanner, PageHeader, Spinner } from '../../components/ui'
import { api } from '../../lib/api'
import { t } from '../../lib/i18n'
import { describeError, useResource } from '../../lib/useResource'

/** The full alert inbox, with an option to include resolved history. */
export default function ClinicianAlerts() {
  const [includeResolved, setIncludeResolved] = useState(false)

  const alerts = useResource(
    (signal) => api.listAlerts({ includeResolved }, signal),
    [includeResolved],
  )

  const [resolvingId, setResolvingId] = useState(null)
  const [resolveError, setResolveError] = useState(null)

  const alertList = alerts.data?.alerts ?? []

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
      <PageHeader
        title={t('clinician.alertInboxTitle')}
        action={
          <Button variant="subtle" size="sm" onClick={() => setIncludeResolved((value) => !value)}>
            {includeResolved ? t('clinician.hideResolved') : t('clinician.showResolved')}
          </Button>
        }
      />

      <ErrorBanner message={resolveError} />

      <Card>
        {alerts.loading ? (
          <Spinner />
        ) : alerts.error ? (
          <ErrorBanner message={describeError(alerts.error)} onRetry={alerts.reload} />
        ) : alertList.length === 0 ? (
          <EmptyState>{t('clinician.alertInboxEmpty')}</EmptyState>
        ) : (
          <div className="flex flex-col gap-3">
            {alertList.map((alert) => (
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
    </>
  )
}
