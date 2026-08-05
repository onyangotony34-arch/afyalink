import { Link } from 'react-router-dom'

import { t } from '../lib/i18n'
import { Button } from '../components/ui'

/**
 * Marketing landing page, ported from the original hand-written CSS to
 * Tailwind with the same content and layout.
 *
 * Note the feature copy mentions SMS delivery, which is a Phase 2 capability
 * (spec §7) and is deliberately absent from the schema and API. It is left in
 * place because it is existing marketing copy describing the product vision,
 * not a claim about what this build implements.
 */

const FEATURES = [
  {
    icon: '💊',
    tint: 'bg-info-light',
    title: 'Smart medication reminders',
    body: 'Timely SMS and app nudges for every dose — so patients never miss a critical medication after discharge.',
  },
  {
    icon: '📋',
    tint: 'bg-teal-light',
    title: 'Daily check-ins',
    body: 'Simple yes/no questions that surface red flags early, before they become readmission events.',
  },
  {
    icon: '🩺',
    tint: 'bg-amber-light',
    title: 'Clinician dashboard',
    body: "A live view of every patient's recovery progress — with alerts for those who need follow-up.",
  },
  {
    icon: '📍',
    tint: 'bg-danger-light',
    title: 'Works anywhere in Kenya',
    body: 'Designed for low-connectivity environments. SMS-first, so every patient is reachable.',
  },
  {
    icon: '📈',
    tint: 'bg-teal-light',
    title: 'Recovery analytics',
    body: 'Trends and patterns that help hospitals reduce readmission rates and improve outcomes.',
  },
  {
    icon: '🔒',
    tint: 'bg-info-light',
    title: 'Privacy by design',
    body: "Patient data stays secure and compliant — built around Kenya's healthcare data standards.",
  },
]

const STEPS = [
  { num: '1', title: 'Patient discharged', body: 'Hospital registers the patient in AfyaLink at discharge.' },
  { num: '2', title: 'Reminders kick in', body: 'Patient receives daily medication and check-in prompts.' },
  { num: '3', title: 'Clinician monitors', body: 'Doctor sees real-time recovery status and gets alerted to concerns.' },
  { num: '4', title: 'Better outcomes', body: 'Fewer readmissions. Healthier patients. Less burden on the system.' },
]

const STATS = [
  { value: '1 in 5', label: 'patients readmitted within 30 days' },
  { value: '72hrs', label: 'most critical window post-discharge' },
  { value: '60%', label: 'of readmissions are preventable' },
]

function Wordmark({ className }) {
  return (
    <span className={className}>
      {t('brand.namePrefix')}
      <span className="text-teal">{t('brand.nameSuffix')}</span>
    </span>
  )
}

export default function Landing() {
  return (
    <div className="bg-canvas">
      <nav className="flex items-center justify-between px-[5%] py-5">
        <Wordmark className="font-display text-2xl text-navy" />
        <Link to="/login">
          <Button size="sm">{t('landing.signIn')}</Button>
        </Link>
      </nav>

      <section className="grid md:grid-cols-2 gap-12 items-center px-[5%] py-16">
        <div>
          <span className="inline-block bg-teal-light text-teal-dark text-xs font-medium px-3 py-1 rounded-full mb-5">
            Post-discharge patient care
          </span>
          <h1 className="font-display text-4xl md:text-5xl text-navy leading-tight mb-5">
            Care doesn't stop <em className="text-teal not-italic md:italic">at discharge.</em>
          </h1>
          <p className="text-ink-muted leading-relaxed mb-7 max-w-lg">
            AfyaLink monitors patients after they leave hospital — medication reminders, daily check-ins,
            and real-time insights for clinicians across Kenya.
          </p>
          <Link to="/login">
            <Button>{t('landing.signIn')}</Button>
          </Link>
        </div>

        {/* Illustrative preview of the patient view. Static by design — this is
            marketing copy, not a live data surface. */}
        <div className="hidden md:block bg-white border border-hairline rounded-2xl p-6 shadow-sm">
          <div className="flex items-center gap-3 pb-4 border-b border-hairline mb-4">
            <div className="size-10 rounded-full bg-teal text-white grid place-items-center text-sm font-medium">
              JO
            </div>
            <div className="flex-1">
              <div className="text-[0.9rem] font-medium text-ink">Jane Otieno</div>
              <div className="text-xs text-ink-muted">Day 4 of recovery · Nairobi</div>
            </div>
            <span className="bg-teal-light text-teal-dark text-xs font-medium px-2.5 py-0.5 rounded-full">
              On track
            </span>
          </div>

          {[
            { icon: '💊', name: 'Amoxicillin 500mg', time: 'Morning · 8:00 AM', done: true },
            { icon: '💉', name: 'Metformin 850mg', time: 'Afternoon · 1:00 PM', done: true },
            { icon: '🩺', name: 'Blood pressure check', time: 'Evening · 6:00 PM', done: false },
          ].map((item) => (
            <div key={item.name} className="flex items-center gap-3 py-2.5">
              <div className="size-9 rounded-lg bg-info-light grid place-items-center">{item.icon}</div>
              <div className="flex-1">
                <div className="text-[0.85rem] text-ink">{item.name}</div>
                <div className="text-xs text-ink-muted">{item.time}</div>
              </div>
              <div
                className={`size-4 rounded-full ${item.done ? 'bg-teal' : 'border-2 border-hairline'}`}
                aria-hidden="true"
              />
            </div>
          ))}
        </div>
      </section>

      <section className="bg-white px-[5%] py-16 text-center border-y border-hairline">
        <h2 className="font-display text-3xl text-navy mb-4">Kenya's silent healthcare gap</h2>
        <p className="text-ink-muted max-w-2xl mx-auto leading-relaxed mb-10">
          Most patient complications happen after discharge — when there's no system watching. Hospitals are
          full, families are overwhelmed, and patients fall through the cracks.
        </p>
        <div className="flex flex-wrap justify-center gap-10">
          {STATS.map((stat) => (
            <div key={stat.label}>
              <span className="block font-display text-3xl text-teal">{stat.value}</span>
              <div className="text-xs text-ink-muted mt-1 max-w-[12rem]">{stat.label}</div>
            </div>
          ))}
        </div>
      </section>

      <section id="features" className="px-[5%] py-16">
        <p className="text-center text-xs font-medium tracking-widest text-teal mb-3">FEATURES</p>
        <h2 className="font-display text-3xl text-navy text-center mb-3">
          Everything a patient needs after discharge
        </h2>
        <p className="text-ink-muted text-center max-w-xl mx-auto mb-10">
          Built for Kenyan hospitals, clinics, and patients — on any device, any connection.
        </p>

        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-5 max-w-5xl mx-auto">
          {FEATURES.map((feature) => (
            <div key={feature.title} className="bg-white border border-hairline rounded-2xl p-6">
              <div className={`size-11 rounded-xl grid place-items-center text-xl mb-4 ${feature.tint}`}>
                {feature.icon}
              </div>
              <h3 className="text-[0.95rem] font-medium text-navy mb-2">{feature.title}</h3>
              <p className="text-sm text-ink-muted leading-relaxed">{feature.body}</p>
            </div>
          ))}
        </div>
      </section>

      <section id="how" className="bg-white px-[5%] py-16 border-y border-hairline">
        <p className="text-center text-xs font-medium tracking-widest text-teal mb-3">HOW IT WORKS</p>
        <h2 className="font-display text-3xl text-navy text-center mb-10">
          Simple. Effective. Built for Africa.
        </h2>

        <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-6 max-w-5xl mx-auto">
          {STEPS.map((step) => (
            <div key={step.num}>
              <div className="size-9 rounded-full bg-teal text-white grid place-items-center font-medium mb-3">
                {step.num}
              </div>
              <h3 className="text-[0.95rem] font-medium text-navy mb-1.5">{step.title}</h3>
              <p className="text-sm text-ink-muted leading-relaxed">{step.body}</p>
            </div>
          ))}
        </div>
      </section>

      <footer className="bg-navy px-[5%] py-10 flex flex-wrap items-center justify-between gap-4">
        <Wordmark className="font-display text-xl text-white" />
        <p className="text-white/35 text-sm">© 2026 AfyaLink Health. Built in Kenya.</p>
      </footer>
    </div>
  )
}
