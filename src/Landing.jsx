import { useState } from "react"
import { useNavigate } from "react-router-dom"

const styles = `
  @import url('https://fonts.googleapis.com/css2?family=DM+Serif+Display:ital@0;1&family=DM+Sans:wght@300;400;500&display=swap');

  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

  :root {
    --teal: #1D9E75;
    --teal-light: #E1F5EE;
    --teal-dark: #085041;
    --navy: #042C53;
    --blue-mid: #185FA5;
    --blue-light: #E6F1FB;
    --text: #2C2C2A;
    --text-muted: #5F5E5A;
    --border: rgba(0,0,0,0.08);
    --bg: #FAFAF8;
  }

  body { font-family: 'DM Sans', sans-serif; background: var(--bg); color: var(--text); }

  /* NAV */
  nav {
    display: flex; align-items: center; justify-content: space-between;
    padding: 1.25rem 5%;
    border-bottom: 1px solid var(--border);
    background: rgba(250,250,248,0.92);
    backdrop-filter: blur(8px);
    position: sticky; top: 0; z-index: 100;
  }
  .logo { font-family: 'DM Serif Display', serif; font-size: 1.6rem; color: var(--teal-dark); letter-spacing: -0.5px; }
  .logo span { color: var(--teal); }
  .nav-links { display: flex; gap: 2rem; list-style: none; }
  .nav-links a { text-decoration: none; font-size: 0.9rem; color: var(--text-muted); font-weight: 400; transition: color 0.2s; }
  .nav-links a:hover { color: var(--teal); }
  .nav-cta {
    background: var(--teal); color: white;
    border: none; padding: 0.55rem 1.3rem;
    border-radius: 50px; font-family: 'DM Sans', sans-serif;
    font-size: 0.88rem; font-weight: 500; cursor: pointer;
    transition: background 0.2s, transform 0.15s;
  }
  .nav-cta:hover { background: var(--teal-dark); transform: translateY(-1px); }

  /* HERO */
  .hero {
    padding: 6rem 5% 5rem;
    max-width: 1100px; margin: 0 auto;
    display: grid; grid-template-columns: 1fr 1fr; gap: 4rem; align-items: center;
  }
  .hero-eyebrow {
    display: inline-flex; align-items: center; gap: 8px;
    background: var(--teal-light); color: var(--teal-dark);
    padding: 0.35rem 0.9rem; border-radius: 50px;
    font-size: 0.8rem; font-weight: 500; margin-bottom: 1.5rem;
    letter-spacing: 0.02em;
  }
  .hero-eyebrow::before { content: ''; width: 6px; height: 6px; background: var(--teal); border-radius: 50%; display: block; }
  .hero h1 {
    font-family: 'DM Serif Display', serif;
    font-size: clamp(2.4rem, 4vw, 3.4rem);
    line-height: 1.15; letter-spacing: -0.03em;
    color: var(--navy); margin-bottom: 1.25rem;
  }
  .hero h1 em { font-style: italic; color: var(--teal); }
  .hero-sub { font-size: 1.05rem; color: var(--text-muted); line-height: 1.75; margin-bottom: 2rem; font-weight: 300; }
  .hero-actions { display: flex; gap: 1rem; align-items: center; flex-wrap: wrap; }
  .btn-primary {
    background: var(--teal); color: white;
    border: none; padding: 0.8rem 1.8rem;
    border-radius: 50px; font-family: 'DM Sans', sans-serif;
    font-size: 0.95rem; font-weight: 500; cursor: pointer;
    transition: background 0.2s, transform 0.15s;
  }
  .btn-primary:hover { background: var(--teal-dark); transform: translateY(-2px); }
  .btn-secondary {
    background: transparent; color: var(--text-muted);
    border: 1.5px solid var(--border); padding: 0.8rem 1.8rem;
    border-radius: 50px; font-family: 'DM Sans', sans-serif;
    font-size: 0.95rem; font-weight: 400; cursor: pointer;
    transition: border-color 0.2s, color 0.2s;
  }
  .btn-secondary:hover { border-color: var(--teal); color: var(--teal); }

  /* HERO CARD */
  .hero-card {
    background: white; border-radius: 20px;
    border: 1px solid var(--border);
    padding: 1.5rem; box-shadow: 0 4px 32px rgba(0,0,0,0.06);
  }
  .card-header { display: flex; align-items: center; gap: 10px; margin-bottom: 1.25rem; padding-bottom: 1rem; border-bottom: 1px solid var(--border); }
  .avatar { width: 40px; height: 40px; border-radius: 50%; background: var(--teal-light); display: flex; align-items: center; justify-content: center; font-weight: 500; font-size: 0.85rem; color: var(--teal-dark); flex-shrink: 0; }
  .card-name { font-weight: 500; font-size: 0.95rem; }
  .card-sub { font-size: 0.78rem; color: var(--text-muted); }
  .status-badge { margin-left: auto; background: #EAF3DE; color: #3B6D11; font-size: 0.72rem; padding: 0.25rem 0.65rem; border-radius: 50px; font-weight: 500; }
  .med-row { display: flex; align-items: center; gap: 10px; padding: 0.7rem 0; border-bottom: 1px solid var(--border); }
  .med-row:last-child { border-bottom: none; padding-bottom: 0; }
  .med-icon { width: 32px; height: 32px; border-radius: 8px; background: var(--blue-light); display: flex; align-items: center; justify-content: center; font-size: 0.9rem; flex-shrink: 0; }
  .med-name { font-size: 0.88rem; font-weight: 500; }
  .med-time { font-size: 0.75rem; color: var(--text-muted); }
  .med-check { margin-left: auto; width: 22px; height: 22px; border-radius: 50%; background: var(--teal); display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
  .med-check::after { content: '✓'; color: white; font-size: 0.7rem; font-weight: 700; }
  .med-pending { margin-left: auto; width: 22px; height: 22px; border-radius: 50%; border: 1.5px solid var(--border); flex-shrink: 0; }

  /* PROBLEM */
  .problem { background: var(--navy); padding: 5rem 5%; text-align: center; }
  .problem h2 { font-family: 'DM Serif Display', serif; font-size: clamp(1.8rem, 3vw, 2.6rem); color: white; margin-bottom: 1rem; }
  .problem p { color: rgba(255,255,255,0.65); max-width: 580px; margin: 0 auto 3rem; font-size: 1rem; line-height: 1.8; font-weight: 300; }
  .stats-row { display: flex; justify-content: center; gap: 3rem; flex-wrap: wrap; max-width: 800px; margin: 0 auto; }
  .stat { text-align: center; }
  .stat-num { font-family: 'DM Serif Display', serif; font-size: 2.8rem; color: var(--teal); display: block; line-height: 1; }
  .stat-label { font-size: 0.82rem; color: rgba(255,255,255,0.5); margin-top: 0.4rem; }

  /* FEATURES */
  .features { padding: 5rem 5%; max-width: 1100px; margin: 0 auto; }
  .section-label { text-align: center; margin-bottom: 0.75rem; }
  .section-label span { background: var(--teal-light); color: var(--teal-dark); font-size: 0.78rem; font-weight: 500; padding: 0.3rem 0.8rem; border-radius: 50px; letter-spacing: 0.05em; }
  .section-title { font-family: 'DM Serif Display', serif; font-size: clamp(1.8rem, 3vw, 2.4rem); text-align: center; color: var(--navy); margin-bottom: 0.75rem; }
  .section-sub { text-align: center; color: var(--text-muted); max-width: 500px; margin: 0 auto 3rem; font-size: 0.95rem; line-height: 1.75; font-weight: 300; }
  .features-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 1.25rem; }
  .feature-card {
    background: white; border: 1px solid var(--border);
    border-radius: 16px; padding: 1.5rem;
    transition: transform 0.2s, box-shadow 0.2s;
  }
  .feature-card:hover { transform: translateY(-3px); box-shadow: 0 8px 28px rgba(0,0,0,0.07); }
  .feature-icon { width: 44px; height: 44px; border-radius: 12px; margin-bottom: 1rem; display: flex; align-items: center; justify-content: center; font-size: 1.2rem; }
  .feature-card h3 { font-size: 1rem; font-weight: 500; margin-bottom: 0.5rem; color: var(--navy); }
  .feature-card p { font-size: 0.88rem; color: var(--text-muted); line-height: 1.7; font-weight: 300; }

  /* HOW IT WORKS */
  .how { background: var(--teal-light); padding: 5rem 5%; }
  .how-inner { max-width: 900px; margin: 0 auto; }
  .steps { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 2rem; margin-top: 3rem; }
  .step { text-align: center; }
  .step-num { width: 44px; height: 44px; border-radius: 50%; background: var(--teal); color: white; font-family: 'DM Serif Display', serif; font-size: 1.2rem; display: flex; align-items: center; justify-content: center; margin: 0 auto 1rem; }
  .step h3 { font-size: 0.95rem; font-weight: 500; color: var(--teal-dark); margin-bottom: 0.4rem; }
  .step p { font-size: 0.83rem; color: #0F6E56; line-height: 1.6; font-weight: 300; }

  /* CTA */
  .cta-section { padding: 6rem 5%; text-align: center; background: white; border-top: 1px solid var(--border); }
  .cta-section h2 { font-family: 'DM Serif Display', serif; font-size: clamp(2rem, 4vw, 3rem); color: var(--navy); margin-bottom: 1rem; }
  .cta-section p { color: var(--text-muted); max-width: 440px; margin: 0 auto 2rem; font-size: 0.95rem; line-height: 1.8; font-weight: 300; }
  .email-form { display: flex; gap: 0.75rem; max-width: 420px; margin: 0 auto; flex-wrap: wrap; justify-content: center; }
  .email-form input {
    flex: 1; min-width: 220px; padding: 0.8rem 1.1rem;
    border: 1.5px solid var(--border); border-radius: 50px;
    font-family: 'DM Sans', sans-serif; font-size: 0.9rem;
    outline: none; transition: border-color 0.2s; background: var(--bg);
    color: var(--text);
  }
  .email-form input:focus { border-color: var(--teal); }
  .email-form input::placeholder { color: var(--text-muted); }

  /* FOOTER */
  footer { background: var(--navy); padding: 2.5rem 5%; display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem; }
  .footer-logo { font-family: 'DM Serif Display', serif; font-size: 1.3rem; color: white; }
  .footer-logo span { color: var(--teal); }
  footer p { color: rgba(255,255,255,0.35); font-size: 0.8rem; }

  @media (max-width: 680px) {
    .hero { grid-template-columns: 1fr; gap: 2.5rem; padding: 3rem 5%; }
    .hero-card { display: none; }
    .nav-links { display: none; }
    .stats-row { gap: 2rem; }
  }
`

const features = [
  {
    icon: "💊",
    bg: "#E6F1FB",
    title: "Smart medication reminders",
    desc: "Timely SMS and app nudges for every dose — so patients never miss a critical medication after discharge."
  },
  {
    icon: "📋",
    bg: "#E1F5EE",
    title: "Daily check-ins",
    desc: "Simple yes/no questions that surface red flags early, before they become readmission events."
  },
  {
    icon: "🩺",
    bg: "#FAEEDA",
    title: "Clinician dashboard",
    desc: "A live view of every patient's recovery progress — with alerts for those who need follow-up."
  },
  {
    icon: "📍",
    bg: "#FBEAF0",
    title: "Works anywhere in Kenya",
    desc: "Designed for low-connectivity environments. SMS-first, so every patient is reachable."
  },
  {
    icon: "📈",
    bg: "#EAF3DE",
    title: "Recovery analytics",
    desc: "Trends and patterns that help hospitals reduce readmission rates and improve outcomes."
  },
  {
    icon: "🔒",
    bg: "#FAECE7",
    title: "Privacy by design",
    desc: "Patient data stays secure and compliant — built around Kenya's healthcare data standards."
  }
]

const steps = [
  { num: "1", title: "Patient discharged", desc: "Hospital registers the patient in AfyaLink at discharge." },
  { num: "2", title: "Reminders kick in", desc: "Patient receives daily medication and check-in prompts via SMS or app." },
  { num: "3", title: "Clinician monitors", desc: "Doctor sees real-time recovery status and gets alerted to concerns." },
  { num: "4", title: "Better outcomes", desc: "Fewer readmissions. Healthier patients. Less burden on the system." },
]

export default function App() {
  const [email, setEmail] = useState("")
  const [submitted, setSubmitted] = useState(false)
  const navigate = useNavigate()
  return (
    <>
      <style>{styles}</style>

      {/* NAV */}
      <nav>
        <div className="logo">Afya<span>Link</span></div>
        <ul className="nav-links">
          <li><a href="#features">Features</a></li>
          <li><a href="#how">How it works</a></li>
          <li><a href="#about">About</a></li>
        </ul>
        <button className="nav-cta" onClick={() => navigate('/login')}>Request access</button>
      </nav>

      {/* HERO */}
      <section className="hero">
        <div>
          <div className="hero-eyebrow">Post-discharge patient care</div>
          <h1>Care doesn't stop <em>at discharge.</em></h1>
          <p className="hero-sub">
            AfyaLink monitors patients after they leave hospital — medication reminders, daily check-ins, and real-time insights for clinicians across Kenya.
          </p>
          <div className="hero-actions">
            <button className="btn-primary" onClick={() => navigate('/login')}>Get early access</button>
            <button className="btn-secondary">See how it works</button>
          </div>
        </div>

        <div className="hero-card">
          <div className="card-header">
            <div className="avatar">JO</div>
            <div>
              <div className="card-name">Jane Otieno</div>
              <div className="card-sub">Day 4 of recovery · Nairobi</div>
            </div>
            <div className="status-badge">On track</div>
          </div>
          {[
            { icon: "💊", name: "Amoxicillin 500mg", time: "Morning · 8:00 AM", done: true },
            { icon: "💉", name: "Metformin 850mg", time: "Afternoon · 1:00 PM", done: true },
            { icon: "🩺", name: "Blood pressure check", time: "Evening · 6:00 PM", done: false },
          ].map((item, i) => (
            <div className="med-row" key={i}>
              <div className="med-icon">{item.icon}</div>
              <div>
                <div className="med-name">{item.name}</div>
                <div className="med-time">{item.time}</div>
              </div>
              {item.done ? <div className="med-check" /> : <div className="med-pending" />}
            </div>
          ))}
        </div>
      </section>

      {/* PROBLEM */}
      <section className="problem">
        <h2>Kenya's silent healthcare gap</h2>
        <p>
          Most patient complications happen after discharge — when there's no system watching. Hospitals are full, families are overwhelmed, and patients fall through the cracks.
        </p>
        <div className="stats-row">
          <div className="stat">
            <span className="stat-num">1 in 5</span>
            <div className="stat-label">patients readmitted within 30 days</div>
          </div>
          <div className="stat">
            <span className="stat-num">72hrs</span>
            <div className="stat-label">most critical window post-discharge</div>
          </div>
          <div className="stat">
            <span className="stat-num">60%</span>
            <div className="stat-label">of readmissions are preventable</div>
          </div>
        </div>
      </section>

      {/* FEATURES */}
      <section className="features" id="features">
        <div className="section-label"><span>FEATURES</span></div>
        <h2 className="section-title">Everything a patient needs after discharge</h2>
        <p className="section-sub">Built for Kenyan hospitals, clinics, and patients — on any device, any connection.</p>
        <div className="features-grid">
          {features.map((f, i) => (
            <div className="feature-card" key={i}>
              <div className="feature-icon" style={{ background: f.bg }}>{f.icon}</div>
              <h3>{f.title}</h3>
              <p>{f.desc}</p>
            </div>
          ))}
        </div>
      </section>

      {/* HOW IT WORKS */}
      <section className="how" id="how">
        <div className="how-inner">
          <div className="section-label"><span>HOW IT WORKS</span></div>
          <h2 className="section-title">Simple. Effective. Built for Africa.</h2>
          <div className="steps">
            {steps.map((s, i) => (
              <div className="step" key={i}>
                <div className="step-num">{s.num}</div>
                <h3>{s.title}</h3>
                <p>{s.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="cta-section">
        <h2>Ready to transform aftercare?</h2>
        <p>Join the waitlist. We're onboarding our first hospitals in Kenya now.</p>
        {submitted ? (
          <p style={{ color: "var(--teal)", fontWeight: 500, fontSize: "1rem" }}>
            ✓ You're on the list. We'll be in touch soon.
          </p>
        ) : (
          <div className="email-form">
            <input
              type="email"
              placeholder="your@hospital.ke"
              value={email}
              onChange={e => setEmail(e.target.value)}
            />
            <button
              className="btn-primary"
              onClick={() => { if (email) setSubmitted(true) }}
            >
              Join waitlist
            </button>
          </div>
        )}
      </section>

      {/* FOOTER */}
      <footer>
        <div className="footer-logo">Afya<span>Link</span></div>
        <p>© 2025 AfyaLink Health. Built in Kenya.</p>
      </footer>
    </>
  )
}
