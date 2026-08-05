import { useState } from 'react'
import { Link } from 'react-router-dom'

import CheckinForm from '../../components/CheckinForm'
import { Button, Card, ErrorBanner, Notice, PageHeader, Spinner } from '../../components/ui'
import { ApiError, api } from '../../lib/api'
import { t } from '../../lib/i18n'
import { describeError, useResource } from '../../lib/useResource'

/**
 * The patient's check-in flow.
 *
 * The red-flag outcome comes back on the same response as the submission —
 * the backend evaluates the rules synchronously inside the request — so the
 * patient is told their care team has been notified immediately, rather than
 * after a poll that might never happen on a flaky connection.
 */
export default function PatientCheckIn() {
  const patients = useResource((signal) => api.listPatients(signal), [])
  const patient = patients.data?.patients?.[0]

  const checkins = useResource(
    (signal) => (patient ? api.listCheckins(patient.id, signal) : Promise.resolve(null)),
    [patient?.id],
  )

  const [submitting, setSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState(null)
  const [result, setResult] = useState(null)

  if (patients.loading || checkins.loading) return <Spinner />
  if (patients.error) {
    return <ErrorBanner message={describeError(patients.error)} onRetry={patients.reload} />
  }
  if (checkins.error) {
    return <ErrorBanner message={describeError(checkins.error)} onRetry={checkins.reload} />
  }

  const dueCheckin = (checkins.data?.checkins ?? []).find((checkin) => checkin.status === 'pending')

  const handleSubmit = async (answers) => {
    setSubmitting(true)
    setSubmitError(null)

    try {
      const response = await api.respondToCheckin(dueCheckin.id, answers)
      setResult(response)
    } catch (cause) {
      if (cause instanceof ApiError && cause.isConflict) {
        setSubmitError(t('checkin.alreadyCompleted'))
      } else if (cause instanceof ApiError && cause.status === 400) {
        setSubmitError(cause.message)
      } else {
        setSubmitError(describeError(cause))
      }
    } finally {
      setSubmitting(false)
    }
  }

  // Completed: acknowledge, and say plainly whether an alert went out.
  if (result) {
    return (
      <>
        <PageHeader title={t('checkin.completeTitle')} />
        <Card className="max-w-xl">
          <p className="text-[0.95rem] text-ink mb-4">{t('checkin.completeBody')}</p>

          {result.alert && (
            <div className="mb-4">
              <Notice tone="amber">
                <strong className="block mb-1">{t('checkin.alertRaisedTitle')}</strong>
                {t('checkin.alertRaisedBody')}
              </Notice>
            </div>
          )}

          <Link to="/patient">
            <Button variant="subtle">← {t('nav.dashboard')}</Button>
          </Link>
        </Card>
      </>
    )
  }

  if (!dueCheckin) {
    return (
      <>
        <PageHeader title={t('checkin.title')} />
        <Card className="max-w-xl">
          <Notice>{t('patient.noCheckinDue')}</Notice>
          <div className="mt-4">
            <Link to="/patient">
              <Button variant="subtle">← {t('nav.dashboard')}</Button>
            </Link>
          </div>
        </Card>
      </>
    )
  }

  return (
    <>
      <PageHeader title={t('checkin.title')} subtitle={t('checkin.intro')} />
      <Card className="max-w-xl">
        <CheckinForm
          checkin={dueCheckin}
          onSubmit={handleSubmit}
          submitting={submitting}
          error={submitError}
        />
      </Card>
    </>
  )
}
