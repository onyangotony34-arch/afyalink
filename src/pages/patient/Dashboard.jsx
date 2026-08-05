import { useState } from 'react'
import { Link } from 'react-router-dom'

import DoseRow from '../../components/MedicationRow'
import {
  Button,
  Card,
  EmptyState,
  ErrorBanner,
  Notice,
  PageHeader,
  SectionTitle,
  Spinner,
  StatCard,
} from '../../components/ui'
import { api } from '../../lib/api'
import { daysSince, isToday } from '../../lib/format'
import { t } from '../../lib/i18n'
import { describeError, useResource } from '../../lib/useResource'

/** The patient's own view: today's check-in and today's doses. */
export default function PatientDashboard() {
  const patients = useResource((signal) => api.listPatients(signal), [])

  if (patients.loading) return <Spinner />
  if (patients.error) {
    return <ErrorBanner message={describeError(patients.error)} onRetry={patients.reload} />
  }

  const patient = patients.data?.patients?.[0]
  if (!patient) return <EmptyState>{t('errors.notFound')}</EmptyState>

  return <PatientHome patient={patient} />
}

function PatientHome({ patient }) {
  const checkins = useResource((signal) => api.listCheckins(patient.id, signal), [patient.id])
  const medications = useResource((signal) => api.listMedications(patient.id, signal), [patient.id])
  const discharges = useResource((signal) => api.listDischarges(patient.id, signal), [patient.id])

  const [markingLogId, setMarkingLogId] = useState(null)
  const [markError, setMarkError] = useState(null)

  const checkinList = checkins.data?.checkins ?? []
  const medicationList = medications.data?.medications ?? []

  // The next check-in the patient can actually act on. Missed ones are history,
  // completed ones are done.
  const dueCheckin = checkinList.find((checkin) => checkin.status === 'pending')
  const completedCount = checkinList.filter((checkin) => checkin.status === 'completed').length

  const todaysDoses = medicationList.flatMap((medication) =>
    medication.logs.filter((log) => isToday(log.scheduled_at)).map((log) => ({ medication, log })),
  )
  const takenToday = todaysDoses.filter(({ log }) => log.status === 'taken').length

  const latestDischarge = discharges.data?.discharge_records?.[0]
  const recoveryDay = latestDischarge ? daysSince(latestDischarge.discharge_date) : null

  const handleMarkTaken = async (medication, log) => {
    setMarkingLogId(log.id)
    setMarkError(null)
    try {
      await api.markDoseTaken(medication.id, log.id)
      medications.reload()
    } catch (cause) {
      setMarkError(describeError(cause))
    } finally {
      setMarkingLogId(null)
    }
  }

  const firstName = patient.full_name.split(' ')[0]

  return (
    <>
      <PageHeader
        title={t('patient.dashboardTitle', { name: firstName })}
        subtitle={t('patient.dashboardSubtitle')}
      />

      <div className="grid grid-cols-2 lg:grid-cols-3 gap-4 mb-7">
        <StatCard
          label={t('patient.recoveryDay')}
          value={recoveryDay ?? '—'}
          sub={latestDischarge ? undefined : t('common.none')}
        />
        <StatCard
          label={t('patient.dosesTaken')}
          value={`${takenToday}/${todaysDoses.length}`}
        />
        <StatCard label={t('patient.checkinsCompleted')} value={completedCount} />
      </div>

      <div className="grid lg:grid-cols-2 gap-6 items-start">
        <Card>
          <SectionTitle>{t('patient.todaysCheckin')}</SectionTitle>

          {checkins.loading ? (
            <Spinner />
          ) : checkins.error ? (
            <ErrorBanner message={describeError(checkins.error)} onRetry={checkins.reload} />
          ) : dueCheckin ? (
            <div>
              <p className="text-[0.9rem] text-ink mb-4">{dueCheckin.template_name}</p>
              <Link to="/patient/checkin">
                <Button>{t('patient.startCheckin')}</Button>
              </Link>
            </div>
          ) : (
            <Notice>{t('patient.noCheckinDue')}</Notice>
          )}
        </Card>

        <Card>
          <SectionTitle>{t('patient.todaysMedication')}</SectionTitle>

          <ErrorBanner message={markError} />

          {medications.loading ? (
            <Spinner />
          ) : medications.error ? (
            <ErrorBanner message={describeError(medications.error)} onRetry={medications.reload} />
          ) : todaysDoses.length === 0 ? (
            <EmptyState>{t('patient.medicationsEmpty')}</EmptyState>
          ) : (
            todaysDoses.map(({ medication, log }) => (
              <DoseRow
                key={log.id}
                medication={medication}
                log={log}
                onMarkTaken={handleMarkTaken}
                marking={markingLogId === log.id}
              />
            ))
          )}
        </Card>
      </div>
    </>
  )
}
