import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'

import DoseRow from '../../components/MedicationRow'
import {
  Badge,
  Button,
  Card,
  EmptyState,
  ErrorBanner,
  PageHeader,
  SectionTitle,
  Spinner,
  StatusPill,
  Textarea,
} from '../../components/ui'
import { api } from '../../lib/api'
import { formatDate, formatDateTime } from '../../lib/format'
import { t } from '../../lib/i18n'
import { describeError, useResource } from '../../lib/useResource'

/**
 * The clinician's full view of one patient: record, discharge, medications,
 * check-in history, and clinical notes.
 *
 * Each section loads independently. A patient with no medications should still
 * render their check-in history, so one empty or failing section never blanks
 * the page.
 */
export default function PatientDetail() {
  const { patientId } = useParams()

  const patient = useResource((signal) => api.getPatient(patientId, signal), [patientId])
  const discharges = useResource((signal) => api.listDischarges(patientId, signal), [patientId])
  const medications = useResource((signal) => api.listMedications(patientId, signal), [patientId])
  const checkins = useResource((signal) => api.listCheckins(patientId, signal), [patientId])
  const notes = useResource((signal) => api.listNotes(patientId, signal), [patientId])

  if (patient.loading) return <Spinner />

  if (patient.error) {
    return (
      <>
        <ErrorBanner message={describeError(patient.error)} onRetry={patient.reload} />
        <Link to="/clinician" className="text-sm text-teal hover:text-teal-dark">
          ← {t('nav.dashboard')}
        </Link>
      </>
    )
  }

  const record = patient.data.patient

  return (
    <>
      <Link to="/clinician" className="inline-block text-sm text-ink-muted hover:text-teal mb-4">
        ← {t('nav.patients')}
      </Link>

      <PageHeader title={record.full_name} subtitle={t('clinician.patientDetailTitle')} />

      <div className="grid lg:grid-cols-[1fr_360px] gap-6 items-start">
        <div className="flex flex-col gap-6">
          <Card>
            <SectionTitle>{t('clinician.patientDetailTitle')}</SectionTitle>
            <dl className="grid sm:grid-cols-2 gap-4 text-sm">
              <DetailField label={t('clinician.dateOfBirth')} value={formatDate(record.dob)} />
              <DetailField label={t('clinician.emergencyContact')} value={record.emergency_contact} />
              <DetailField label={t('clinician.diagnosis')} value={record.diagnosis} span />
              <DetailField label={t('clinician.allergies')} value={record.allergies} span />
            </dl>
          </Card>

          <Card>
            <SectionTitle>{t('clinician.checkinHistory')}</SectionTitle>
            <CheckinHistory resource={checkins} />
          </Card>

          <Card>
            <SectionTitle>{t('clinician.clinicalNotes')}</SectionTitle>
            <NotesSection patientId={patientId} resource={notes} />
          </Card>
        </div>

        <div className="flex flex-col gap-6">
          <Card>
            <SectionTitle>{t('clinician.dischargeRecords')}</SectionTitle>
            <DischargeSection resource={discharges} />
          </Card>

          <Card>
            <SectionTitle>{t('clinician.medications')}</SectionTitle>
            <MedicationSection resource={medications} />
          </Card>
        </div>
      </div>
    </>
  )
}

function DetailField({ label, value, span }) {
  return (
    <div className={span ? 'sm:col-span-2' : undefined}>
      <dt className="text-xs text-ink-muted mb-0.5">{label}</dt>
      <dd className="text-ink leading-relaxed">{value || t('common.none')}</dd>
    </div>
  )
}

function DischargeSection({ resource }) {
  if (resource.loading) return <Spinner />
  if (resource.error) return <ErrorBanner message={describeError(resource.error)} onRetry={resource.reload} />

  const records = resource.data?.discharge_records ?? []
  if (records.length === 0) return <EmptyState>{t('clinician.dischargeEmpty')}</EmptyState>

  return (
    <div className="flex flex-col gap-4">
      {records.map((record) => (
        <div key={record.id} className="border-b border-hairline last:border-b-0 pb-4 last:pb-0">
          <Badge tone="info">{t('clinician.dischargedOn', { date: formatDate(record.discharge_date) })}</Badge>
          {record.summary && (
            <p className="text-sm text-ink leading-relaxed mt-2.5">{record.summary}</p>
          )}
        </div>
      ))}
    </div>
  )
}

function MedicationSection({ resource }) {
  if (resource.loading) return <Spinner />
  if (resource.error) return <ErrorBanner message={describeError(resource.error)} onRetry={resource.reload} />

  const medications = resource.data?.medications ?? []
  if (medications.length === 0) return <EmptyState>{t('clinician.medicationsEmpty')}</EmptyState>

  // No onMarkTaken: only the patient may record a dose, and the API enforces it.
  return medications.map((medication) => (
    <DoseRow key={medication.id} medication={medication} log={null} />
  ))
}

function CheckinHistory({ resource }) {
  if (resource.loading) return <Spinner />
  if (resource.error) return <ErrorBanner message={describeError(resource.error)} onRetry={resource.reload} />

  const checkins = resource.data?.checkins ?? []
  if (checkins.length === 0) return <EmptyState>{t('clinician.checkinHistoryEmpty')}</EmptyState>

  return (
    <div className="flex flex-col gap-4">
      {checkins.map((checkin) => {
        const questionText = new Map(checkin.questions.map((question) => [question.id, question.text]))

        return (
          <div key={checkin.id} className="border-b border-hairline last:border-b-0 pb-4 last:pb-0">
            <div className="flex flex-wrap items-center gap-2 mb-2">
              <span className="text-sm font-medium text-ink">{checkin.template_name}</span>
              <StatusPill status={checkin.status} label={t(`checkinStatus.${checkin.status}`)} />
              <span className="text-xs text-ink-muted ml-auto">{formatDateTime(checkin.scheduled_at)}</span>
            </div>

            {checkin.responses.length > 0 && (
              <ul className="flex flex-col gap-1.5 mt-2">
                {checkin.responses.map((response) => (
                  <li
                    key={response.question_id}
                    className="flex items-start justify-between gap-3 text-[0.82rem]"
                  >
                    <span className="text-ink-muted">
                      {questionText.get(response.question_id) ?? response.question_id}
                    </span>
                    <span
                      className={
                        response.is_red_flag ? 'font-medium text-danger shrink-0' : 'text-ink shrink-0'
                      }
                    >
                      {response.answer}
                      {response.is_red_flag && ' ⚠'}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        )
      })}
    </div>
  )
}

function NotesSection({ patientId, resource }) {
  const [body, setBody] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState(null)

  const handleSave = async () => {
    const trimmed = body.trim()
    if (!trimmed) return

    setSaving(true)
    setError(null)
    try {
      await api.createNote(patientId, trimmed)
      setBody('')
      resource.reload()
    } catch (cause) {
      setError(describeError(cause))
    } finally {
      setSaving(false)
    }
  }

  const notes = resource.data?.notes ?? []

  return (
    <>
      <div className="mb-5">
        <Textarea
          value={body}
          onChange={(event) => setBody(event.target.value)}
          placeholder={t('clinician.notePlaceholder')}
          aria-label={t('clinician.addNote')}
        />
        <ErrorBanner message={error} />
        <div className="mt-2.5">
          <Button size="sm" onClick={handleSave} disabled={saving || !body.trim()}>
            {saving ? t('common.saving') : t('clinician.saveNote')}
          </Button>
        </div>
      </div>

      {resource.loading ? (
        <Spinner />
      ) : resource.error ? (
        <ErrorBanner message={describeError(resource.error)} onRetry={resource.reload} />
      ) : notes.length === 0 ? (
        <EmptyState>{t('clinician.clinicalNotesEmpty')}</EmptyState>
      ) : (
        <div className="flex flex-col gap-3">
          {notes.map((note) => (
            <div key={note.id} className="bg-canvas rounded-xl p-3.5">
              <p className="text-sm text-ink leading-relaxed">{note.body}</p>
              <p className="text-xs text-ink-muted mt-1.5">{formatDateTime(note.created_at)}</p>
            </div>
          ))}
        </div>
      )}
    </>
  )
}
