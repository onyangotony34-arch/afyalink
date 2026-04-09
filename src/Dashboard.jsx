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
    --amber: #BA7517;
    --amber-light: #FAEEDA;
    --red: #A32D2D;
    --red-light: #FCEBEB;
    --blue-light: #E6F1FB;
    --blue-mid: #185FA5;
    --text: #2C2C2A;
    --text-muted: #5F5E5A;
    --border: rgba(0,0,0,0.08);
    --bg: #FAFAF8;
    --white: #ffffff;
  }

  body { font-family: 'DM Sans', sans-serif; background: var(--bg); color: var(--text); min-height: 100vh; }

  /* LAYOUT */
  .dashboard { display: grid; grid-template-columns: 240px 1fr; min-height: 100vh; }

  /* SIDEBAR */
  .sidebar {
    background: var(--navy); padding: 2rem 0;
    display: flex; flex-direction: column;
    position: sticky; top: 0; height: 100vh;
  }
  .sidebar-logo { font-family: 'DM Serif Display', serif; font-size: 1.5rem; color: white; padding: 0 1.5rem; margin-bottom: 2rem; }
  .sidebar-logo span { color: var(--teal); }
  .sidebar-nav { flex: 1; }
  .nav-item {
    display: flex; align-items: center; gap: 10px;
    padding: 0.75rem 1.5rem; cursor: pointer;
    font-size: 0.88rem; color: rgba(255,255,255,0.5);
    transition: all 0.2s; border-left: 3px solid transparent;
    text-decoration: none;
  }
  .nav-item:hover { color: white; background: rgba(255,255,255,0.05); }
  .nav-item.active { color: white; border-left-color: var(--teal); background: rgba(29,158,117,0.1); }
  .nav-item .nav-icon { font-size: 1rem; width: 20px; text-align: center; }
  .sidebar-bottom { padding: 1.5rem; border-top: 1px solid rgba(255,255,255,0.08); }
  .sidebar-user { display: flex; align-items: center; gap: 10px; }
  .user-avatar { width: 36px; height: 36px; border-radius: 50%; background: var(--teal); display: flex; align-items: center; justify-content: center; font-size: 0.8rem; font-weight: 500; color: white; flex-shrink: 0; }
  .user-name { font-size: 0.85rem; color: white; font-weight: 500; }
  .user-role { font-size: 0.72rem; color: rgba(255,255,255,0.4); }

  /* MAIN */
  .main { padding: 2rem 2.5rem; overflow-y: auto; }
  .main-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 2rem; }
  .main-header h1 { font-family: 'DM Serif Display', serif; font-size: 1.6rem; color: var(--navy); }
  .main-header p { font-size: 0.85rem; color: var(--text-muted); margin-top: 2px; }
  .header-date { font-size: 0.82rem; color: var(--text-muted); background: white; border: 1px solid var(--border); padding: 0.45rem 0.9rem; border-radius: 50px; }

  /* STAT CARDS */
  .stats-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 1rem; margin-bottom: 1.75rem; }
  .stat-card { background: white; border: 1px solid var(--border); border-radius: 14px; padding: 1.25rem; }
  .stat-card .stat-label { font-size: 0.78rem; color: var(--text-muted); margin-bottom: 0.5rem; }
  .stat-card .stat-value { font-family: 'DM Serif Display', serif; font-size: 2rem; color: var(--navy); line-height: 1; }
  .stat-card .stat-sub { font-size: 0.75rem; color: var(--text-muted); margin-top: 0.4rem; }
  .stat-card .stat-badge { display: inline-flex; align-items: center; gap: 4px; font-size: 0.72rem; font-weight: 500; padding: 0.2rem 0.6rem; border-radius: 50px; margin-top: 0.5rem; }
  .badge-green { background: var(--teal-light); color: var(--teal-dark); }
  .badge-amber { background: var(--amber-light); color: var(--amber); }
  .badge-red { background: var(--red-light); color: var(--red); }

  /* GRID */
  .content-grid { display: grid; grid-template-columns: 1fr 340px; gap: 1.5rem; }

  /* SECTION TITLES */
  .section-title { font-size: 0.9rem; font-weight: 500; color: var(--navy); margin-bottom: 1rem; display: flex; align-items: center; justify-content: space-between; }
  .section-title a { font-size: 0.78rem; color: var(--teal); text-decoration: none; font-weight: 400; }

  /* MEDICATION LIST */
  .card { background: white; border: 1px solid var(--border); border-radius: 16px; padding: 1.5rem; margin-bottom: 1.25rem; }
  .med-item { display: flex; align-items: center; gap: 12px; padding: 0.85rem 0; border-bottom: 1px solid var(--border); }
  .med-item:last-child { border-bottom: none; padding-bottom: 0; }
  .med-icon { width: 40px; height: 40px; border-radius: 10px; background: var(--blue-light); display: flex; align-items: center; justify-content: center; font-size: 1.1rem; flex-shrink: 0; }
  .med-icon.taken { background: var(--teal-light); }
  .med-icon.missed { background: var(--red-light); }
  .med-info { flex: 1; }
  .med-name { font-size: 0.9rem; font-weight: 500; color: var(--text); }
  .med-detail { font-size: 0.75rem; color: var(--text-muted); margin-top: 2px; }
  .med-status { flex-shrink: 0; }
  .status-pill { font-size: 0.72rem; font-weight: 500; padding: 0.25rem 0.7rem; border-radius: 50px; }
  .pill-taken { background: var(--teal-light); color: var(--teal-dark); }
  .pill-pending { background: var(--amber-light); color: var(--amber); }
  .pill-missed { background: var(--red-light); color: var(--red); }

  /* CHECKIN */
  .checkin-card { background: white; border: 1px solid var(--border); border-radius: 16px; padding: 1.5rem; margin-bottom: 1.25rem; }
  .checkin-question { font-size: 0.9rem; color: var(--text); margin-bottom: 0.75rem; line-height: 1.5; }
  .checkin-buttons { display: flex; gap: 8px; }
  .checkin-btn {
    flex: 1; padding: 0.6rem; border-radius: 8px; border: 1.5px solid var(--border);
    font-family: 'DM Sans', sans-serif; font-size: 0.85rem; font-weight: 500;
    cursor: pointer; background: white; transition: all 0.2s;
  }
  .checkin-btn.yes:hover, .checkin-btn.yes.selected { background: var(--teal-light); border-color: var(--teal); color: var(--teal-dark); }
  .checkin-btn.no:hover, .checkin-btn.no.selected { background: var(--red-light); border-color: var(--red); color: var(--red); }

  /* RECOVERY PROGRESS */
  .progress-wrap { margin: 1rem 0; }
  .progress-label { display: flex; justify-content: space-between; font-size: 0.8rem; margin-bottom: 0.4rem; }
  .progress-label span:first-child { color: var(--text-muted); }
  .progress-label span:last-child { color: var(--teal-dark); font-weight: 500; }
  .progress-bar { height: 8px; background: var(--teal-light); border-radius: 50px; overflow: hidden; }
  .progress-fill { height: 100%; background: var(--teal); border-radius: 50px; transition: width 0.5s ease; }

  /* TIMELINE */
  .timeline { }
  .timeline-item { display: flex; gap: 12px; padding-bottom: 1.25rem; position: relative; }
  .timeline-item::before { content: ''; position: absolute; left: 15px; top: 28px; bottom: 0; width: 1px; background: var(--border); }
  .timeline-item:last-child::before { display: none; }
  .timeline-dot { width: 30px; height: 30px; border-radius: 50%; display: flex; align-items: center; justify-content: center; font-size: 0.8rem; flex-shrink: 0; }
  .dot-green { background: var(--teal-light); }
  .dot-blue { background: var(--blue-light); }
  .dot-amber { background: var(--amber-light); }
  .timeline-content { flex: 1; }
  .timeline-title { font-size: 0.85rem; font-weight: 500; color: var(--text); }
  .timeline-time { font-size: 0.73rem; color: var(--text-muted); margin-top: 2px; }

  /* CLINICIAN CARD */
  .clinician-card { background: var(--navy); border-radius: 16px; padding: 1.25rem; color: white; }
  .clinician-card h3 { font-size: 0.82rem; color: rgba(255,255,255,0.5); margin-bottom: 0.75rem; }
  .clinician-info { display: flex; align-items: center; gap: 10px; }
  .clinician-avatar { width: 40px; height: 40px; border-radius: 50%; background: var(--teal); display: flex; align-items: center; justify-content: center; font-size: 0.82rem; font-weight: 500; color: white; flex-shrink: 0; }
  .clinician-name { font-size: 0.9rem; font-weight: 500; color: white; }
  .clinician-role { font-size: 0.75rem; color: rgba(255,255,255,0.5); }
  .contact-btn { width: 100%; margin-top: 1rem; padding: 0.65rem; background: rgba(29,158,117,0.2); border: 1px solid rgba(29,158,117,0.3); color: var(--teal); border-radius: 8px; font-family: 'DM Sans', sans-serif; font-size: 0.85rem; font-weight: 500; cursor: pointer; transition: all 0.2s; }
  .contact-btn:hover { background: rgba(29,158,117,0.3); }

  /* NEXT APPOINTMENT */
  .appt-card { background: var(--teal); border-radius: 16px; padding: 1.25rem; color: white; margin-top: 1rem; }
  .appt-card h3 { font-size: 0.78rem; color: rgba(255,255,255,0.7); margin-bottom: 0.5rem; }
  .appt-date { font-family: 'DM Serif Display', serif; font-size: 1.4rem; color: white; }
  .appt-detail { font-size: 0.78rem; color: rgba(255,255,255,0.7); margin-top: 0.3rem; }

  /* MOBILE */
  @media (max-width: 900px) {
    .dashboard { grid-template-columns: 1fr; }
    .sidebar { display: none; }
    .stats-row { grid-template-columns: 1fr 1fr; }
    .content-grid { grid-template-columns: 1fr; }
    .main { padding: 1.5rem; }
  }
`

const MEDICATIONS = [
  { icon: "💊", name: "Amoxicillin 500mg", detail: "Morning · 8:00 AM · 1 capsule", status: "taken" },
  { icon: "💉", name: "Metformin 850mg", detail: "Afternoon · 1:00 PM · 1 tablet", status: "taken" },
  { icon: "🩺", name: "Lisinopril 10mg", detail: "Evening · 6:00 PM · 1 tablet", status: "pending" },
  { icon: "💊", name: "Vitamin D3", detail: "Night · 9:00 PM · 1 capsule", status: "pending" },
]

const CHECKINS = [
  { q: "Are you experiencing any chest pain or shortness of breath today?" },
  { q: "Have you taken all your morning medications as prescribed?" },
  { q: "Are you able to eat and drink normally today?" },
]

const TIMELINE = [
  { icon: "🏥", dot: "dot-blue", title: "Discharged from Kenyatta Hospital", time: "Monday, 3 days ago" },
  { icon: "💊", dot: "dot-green", title: "Started medication schedule", time: "Monday, 3 days ago" },
  { icon: "✅", dot: "dot-green", title: "Day 1 check-in completed", time: "Tuesday, 2 days ago" },
  { icon: "✅", dot: "dot-green", title: "Day 2 check-in completed", time: "Wednesday, yesterday" },
  { icon: "📋", dot: "dot-amber", title: "Day 3 check-in pending", time: "Today" },
]

export default function Dashboard() {
  const [checkins, setCheckins] = useState({})
  const [activeNav, setActiveNav] = useState("dashboard")
  const navigate = useNavigate()
  const handleCheckin = (idx, val) => {
    setCheckins(prev => ({ ...prev, [idx]: val }))
  }

  const answeredCount = Object.keys(checkins).length

  return (
    <>
      <style>{styles}</style>
      <div className="dashboard">

        {/* SIDEBAR */}
        <div className="sidebar">
          <div className="sidebar-logo">Afya<span>Link</span></div>
          <nav className="sidebar-nav">
            {[
              { id: "dashboard", icon: "🏠", label: "Dashboard" },
              { id: "medications", icon: "💊", label: "Medications" },
              { id: "checkins", icon: "📋", label: "Check-ins" },
              { id: "appointments", icon: "📅", label: "Appointments" },
              { id: "messages", icon: "💬", label: "Messages" },
              { id: "records", icon: "📁", label: "My Records" },
            ].map(item => (
              <div key={item.id}
                className={`nav-item ${activeNav === item.id ? "active" : ""}`}
                onClick={() => { setActiveNav(item.id); if(item.id === "checkins") navigate('/checkin'); }}>
                <span className="nav-icon">{item.icon}</span>
                {item.label}
              </div>
            ))}
          </nav>
          <div className="sidebar-bottom">
            <div className="sidebar-user">
              <div className="user-avatar">JO</div>
              <div>
                <div className="user-name">Jane Otieno</div>
                <div className="user-role">Patient · Day 4</div>
              </div>
            </div>
          </div>
        </div>

        {/* MAIN */}
        <div className="main">
          <div className="main-header">
            <div>
              <h1>Good morning, Jane 👋</h1>
              <p>Here's your recovery overview for today</p>
            </div>
            <div className="header-date">Thursday, 9 April 2026</div>
          </div>

          {/* STATS */}
          <div className="stats-row">
            <div className="stat-card">
              <div className="stat-label">Recovery day</div>
              <div className="stat-value">4</div>
              <div className="stat-sub">of 14 days</div>
              <div className="stat-badge badge-green">On track</div>
            </div>
            <div className="stat-card">
              <div className="stat-label">Medications today</div>
              <div className="stat-value">2/4</div>
              <div className="stat-sub">taken so far</div>
              <div className="stat-badge badge-amber">2 pending</div>
            </div>
            <div className="stat-card">
              <div className="stat-label">Check-ins done</div>
              <div className="stat-value">{answeredCount}/3</div>
              <div className="stat-sub">questions answered</div>
              <div className="stat-badge badge-green">{answeredCount === 3 ? "Complete" : "In progress"}</div>
            </div>
            <div className="stat-card">
              <div className="stat-label">Next appointment</div>
              <div className="stat-value" style={{ fontSize: "1.3rem" }}>Apr 14</div>
              <div className="stat-sub">Kenyatta Hospital</div>
              <div className="stat-badge badge-green">5 days away</div>
            </div>
          </div>

          <div className="content-grid">
            <div>
              {/* MEDICATIONS */}
              <div className="card">
                <div className="section-title">
                  Today's medications
                  <a href="#">View all →</a>
                </div>
                {MEDICATIONS.map((m, i) => (
                  <div className="med-item" key={i}>
                    <div className={`med-icon ${m.status}`}>{m.icon}</div>
                    <div className="med-info">
                      <div className="med-name">{m.name}</div>
                      <div className="med-detail">{m.detail}</div>
                    </div>
                    <div className="med-status">
                      <span className={`status-pill ${m.status === "taken" ? "pill-taken" : m.status === "missed" ? "pill-missed" : "pill-pending"}`}>
                        {m.status === "taken" ? "✓ Taken" : m.status === "missed" ? "Missed" : "Pending"}
                      </span>
                    </div>
                  </div>
                ))}
              </div>

              {/* CHECK-INS */}
              <div className="card">
                <div className="section-title">
                  Today's check-in
                  <span style={{ fontSize: "0.78rem", color: "var(--text-muted)" }}>{answeredCount} of 3 answered</span>
                </div>

                {/* PROGRESS */}
                <div className="progress-wrap">
                  <div className="progress-label">
                    <span>Daily check-in progress</span>
                    <span>{Math.round((answeredCount / 3) * 100)}%</span>
                  </div>
                  <div className="progress-bar">
                    <div className="progress-fill" style={{ width: `${(answeredCount / 3) * 100}%` }} />
                  </div>
                </div>

                {CHECKINS.map((c, i) => (
                  <div className="checkin-card" key={i} style={{ marginTop: "0.75rem", padding: "1rem", border: "1px solid var(--border)", borderRadius: "12px" }}>
                    <div className="checkin-question">{i + 1}. {c.q}</div>
                    <div className="checkin-buttons">
                      <button
                        className={`checkin-btn yes ${checkins[i] === "yes" ? "selected" : ""}`}
                        onClick={() => handleCheckin(i, "yes")}>
                        Yes
                      </button>
                      <button
                        className={`checkin-btn no ${checkins[i] === "no" ? "selected" : ""}`}
                        onClick={() => handleCheckin(i, "no")}>
                        No
                      </button>
                    </div>
                  </div>
                ))}

                {answeredCount === 3 && (
                  <div style={{ background: "var(--teal-light)", borderRadius: "10px", padding: "0.85rem 1rem", marginTop: "1rem", fontSize: "0.85rem", color: "var(--teal-dark)", fontWeight: 500 }}>
                    ✓ Today's check-in complete! Your clinician has been notified.
                  </div>
                )}
              </div>
            </div>

            {/* RIGHT COLUMN */}
            <div>
              {/* RECOVERY PROGRESS */}
              <div className="card" style={{ marginBottom: "1rem" }}>
                <div className="section-title">Recovery progress</div>
                {[
                  { label: "Overall recovery", val: 28 },
                  { label: "Medication adherence", val: 75 },
                  { label: "Check-in streak", val: 100 },
                ].map((p, i) => (
                  <div className="progress-wrap" key={i}>
                    <div className="progress-label">
                      <span>{p.label}</span>
                      <span>{p.val}%</span>
                    </div>
                    <div className="progress-bar">
                      <div className="progress-fill" style={{ width: `${p.val}%` }} />
                    </div>
                  </div>
                ))}
              </div>

              {/* CLINICIAN */}
              <div className="clinician-card">
                <h3>YOUR CLINICIAN</h3>
                <div className="clinician-info">
                  <div className="clinician-avatar">DK</div>
                  <div>
                    <div className="clinician-name">Dr. David Kamau</div>
                    <div className="clinician-role">General Physician · KMPDC</div>
                  </div>
                </div>
                <button className="contact-btn">📞 Contact Dr. Kamau</button>
              </div>

              {/* NEXT APPOINTMENT */}
              <div className="appt-card">
                <h3>NEXT APPOINTMENT</h3>
                <div className="appt-date">April 14, 2026</div>
                <div className="appt-detail">10:00 AM · Kenyatta National Hospital</div>
                <div className="appt-detail" style={{ marginTop: "0.5rem" }}>Outpatient Clinic, Room 4B</div>
              </div>

              {/* TIMELINE */}
              <div className="card" style={{ marginTop: "1rem" }}>
                <div className="section-title">Recovery timeline</div>
                <div className="timeline">
                  {TIMELINE.map((t, i) => (
                    <div className="timeline-item" key={i}>
                      <div className={`timeline-dot ${t.dot}`}>{t.icon}</div>
                      <div className="timeline-content">
                        <div className="timeline-title">{t.title}</div>
                        <div className="timeline-time">{t.time}</div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </>
  )
}
