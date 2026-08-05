import { useMemo, useState } from 'react'

import { t } from '../lib/i18n'
import { cx } from '../lib/cx'
import { Button, Notice } from './ui'

/**
 * The patient's question flow.
 *
 * One question per screen rather than a long scrolling form: this is answered
 * on a phone by someone recovering from surgery, and a single clear question
 * gets more honest answers than a wall of them.
 *
 * The component knows nothing about red flags. Which answers are concerning is
 * decided server-side and never sent to the client, so there is nothing here
 * for a patient to read off and answer around.
 */
export default function CheckinForm({ checkin, onSubmit, submitting, error }) {
  // Memoised so the `unanswered` memo below is not invalidated by a fresh []
  // literal on every render.
  const questions = useMemo(() => checkin.questions ?? [], [checkin.questions])

  const [answers, setAnswers] = useState({})
  const [index, setIndex] = useState(0)

  const question = questions[index]
  const isLast = index === questions.length - 1

  const unanswered = useMemo(
    () => questions.filter((q) => answers[q.id] === undefined),
    [questions, answers],
  )

  const setAnswer = (questionId, value) => {
    setAnswers((previous) => ({ ...previous, [questionId]: value }))
  }

  const handleNext = () => {
    if (isLast) {
      onSubmit(answers)
    } else {
      setIndex((current) => Math.min(current + 1, questions.length - 1))
    }
  }

  if (!question) return null

  const currentAnswer = answers[question.id]
  const canAdvance = currentAnswer !== undefined && (!isLast || unanswered.length === 0)

  return (
    <div>
      <div className="flex items-center justify-between mb-2 text-xs text-ink-muted">
        <span>{t('checkin.progress', { current: index + 1, total: questions.length })}</span>
        <span>{checkin.template_name}</span>
      </div>

      <div className="h-1.5 bg-teal-light rounded-full overflow-hidden mb-6">
        <div
          className="h-full bg-teal transition-[width] duration-300"
          style={{ width: `${((index + (currentAnswer !== undefined ? 1 : 0)) / questions.length) * 100}%` }}
        />
      </div>

      <fieldset>
        <legend className="text-lg text-ink leading-snug mb-5">{question.text}</legend>

        {question.type === 'yes_no' ? (
          <YesNoAnswer
            value={currentAnswer}
            onChange={(value) => setAnswer(question.id, value)}
          />
        ) : (
          <ScaleAnswer
            question={question}
            value={currentAnswer}
            onChange={(value) => setAnswer(question.id, value)}
          />
        )}
      </fieldset>

      {error && (
        <div className="mt-5">
          <Notice tone="danger">{error}</Notice>
        </div>
      )}

      {isLast && unanswered.length > 0 && (
        <div className="mt-5">
          <Notice tone="amber">{t('checkin.unanswered')}</Notice>
        </div>
      )}

      <div className="flex gap-3 mt-6">
        {index > 0 && (
          <Button variant="subtle" onClick={() => setIndex((current) => current - 1)} disabled={submitting}>
            {t('common.back')}
          </Button>
        )}
        <Button className="flex-1" onClick={handleNext} disabled={!canAdvance || submitting}>
          {isLast ? (submitting ? t('checkin.submitting') : t('checkin.submit')) : '→'}
        </Button>
      </div>
    </div>
  )
}

function YesNoAnswer({ value, onChange }) {
  const options = [
    { value: 'yes', label: t('common.yes'), active: 'border-danger bg-danger-light text-danger' },
    { value: 'no', label: t('common.no'), active: 'border-teal bg-teal-light text-teal-dark' },
  ]

  // Note the colour assignment: on these questions "yes" is usually the
  // worrying answer ("are you in pain?"), so yes reads warm and no reads calm.
  return (
    <div className="grid grid-cols-2 gap-3">
      {options.map((option) => (
        <button
          key={option.value}
          type="button"
          aria-pressed={value === option.value}
          onClick={() => onChange(option.value)}
          className={cx(
            'py-4 rounded-xl border-2 font-medium transition-colors',
            value === option.value ? option.active : 'border-hairline bg-white text-ink hover:border-teal',
          )}
        >
          {option.label}
        </button>
      ))}
    </div>
  )
}

function ScaleAnswer({ question, value, onChange }) {
  const min = question.scale_min ?? 0
  const max = question.scale_max ?? 10
  const steps = Array.from({ length: max - min + 1 }, (_, offset) => min + offset)

  return (
    <div>
      <p className="text-xs text-ink-muted mb-3">{t('checkin.scaleHint', { min, max })}</p>
      <div className="flex flex-wrap gap-2">
        {steps.map((step) => (
          <button
            key={step}
            type="button"
            aria-pressed={value === String(step)}
            onClick={() => onChange(String(step))}
            className={cx(
              'size-11 rounded-lg border-2 font-medium transition-colors',
              value === String(step)
                ? 'border-teal bg-teal text-white'
                : 'border-hairline bg-white text-ink hover:border-teal',
            )}
          >
            {step}
          </button>
        ))}
      </div>
    </div>
  )
}
