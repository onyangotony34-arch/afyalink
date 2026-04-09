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
    --text: #2C2C2A;
    --text-muted: #5F5E5A;
    --border: rgba(0,0,0,0.08);
    --bg: #FAFAF8;
  }

  body { font-family: 'DM Sans', sans-serif; background: var(--bg); color: var(--text); min-height: 100vh; }

  .login-page { min-height: 100vh; display: grid; grid-template-columns: 1fr 1fr; }

  .login-left {
    background: var(--navy); padding: 3rem;
    display: flex; flex-direction: column;
    justify-content: space-between;
    position: relative; overflow: hidden;
  }
  .login-left::before {
    content: ''; position: absolute; width: 400px; height: 400px;
    border-radius: 50%; background: rgba(29,158,117,0.12); top: -100px; left: -100px;
  }
  .login-left::after {
    content: ''; position: absolute; width: 300px; height: 300px;
    border-radius: 50%; background: rgba(29,158,117,0.08); bottom: -80px; right: -80px;
  }
  .left-logo { font-family: 'DM Serif Display', serif; font-size: 1.8rem; color: white; position: relative; z-index: 1; }
  .left-logo span { color: var(--teal); }
  .left-content { position: relative; z-index: 1; }
  .left-content h2 { font-family: 'DM Serif Display', serif; font-size: 2rem; color: white; line-height: 1.2; margin-bottom: 1rem; }
  .left-content h2 em { font-style: italic; color: var(--teal); }
  .left-content p { color: rgba(255,255,255,0.55); font-size: 0.92rem; line-height: 1.75; font-weight: 300; }
  .left-stats { display: flex; gap: 1.5rem; position: relative; z-index: 1; flex-wrap: wrap; }
  .left-stat span { display: block; font-family: 'DM Serif Display', serif; font-size: 1.6rem; color: var(--teal); }
  .left-stat p { font-size: 0.72rem; color: rgba(255,255,255,0.4); margin-top: 2px; }

  .login-right {
    display: flex; align-items: center; justify-content: center;
    padding: 2.5rem 2rem; background: var(--bg); overflow-y: auto;
  }
  .login-box { width: 100%; max-width: 430px; padding: 1rem 0; }
  .login-box h1 { font-family: 'DM Serif Display', serif; font-size: 1.8rem; color: var(--navy); margin-bottom: 0.3rem; }
  .subtitle { font-size: 0.88rem; color: var(--text-muted); margin-bottom: 1.75rem; font-weight: 300; }

  .role-tabs { display: flex; gap: 6px; margin-bottom: 1.75rem; background: white; border: 1px solid var(--border); border-radius: 12px; padding: 4px; }
  .role-tab {
    flex: 1; padding: 0.55rem 0.25rem; border: none; background: transparent;
    border-radius: 8px; cursor: pointer; font-family: 'DM Sans', sans-serif;
    font-size: 0.8rem; font-weight: 400; color: var(--text-muted); transition: all 0.2s;
    display: flex; flex-direction: column; align-items: center; gap: 3px;
  }
  .role-tab .tab-icon { font-size: 1rem; }
  .role-tab.active { background: var(--teal); color: white; font-weight: 500; }
  .role-tab:hover:not(.active) { background: var(--teal-light); color: var(--teal-dark); }

  .patient-toggle { display: flex; background: white; border: 1px solid var(--border); border-radius: 50px; padding: 3px; margin-bottom: 1.5rem; }
  .patient-toggle-btn {
    flex: 1; padding: 0.5rem; border: none; background: transparent;
    border-radius: 50px; cursor: pointer; font-family: 'DM Sans', sans-serif;
    font-size: 0.82rem; color: var(--text-muted); transition: all 0.2s;
  }
  .patient-toggle-btn.active { background: var(--teal); color: white; font-weight: 500; }

  .profession-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 7px; margin-bottom: 1.25rem; }
  .profession-card {
    border: 1.5px solid var(--border); border-radius: 10px;
    padding: 0.65rem 0.75rem; cursor: pointer; background: white; transition: all 0.2s;
  }
  .profession-card:hover { border-color: var(--teal); }
  .profession-card.selected { border-color: var(--teal); background: var(--teal-light); }
  .profession-card .prof-icon { font-size: 1.1rem; margin-bottom: 3px; }
  .profession-card .prof-name { font-size: 0.82rem; font-weight: 500; color: var(--text); }
  .profession-card .prof-body { font-size: 0.68rem; color: var(--text-muted); margin-top: 1px; }
  .profession-card.selected .prof-name { color: var(--teal-dark); }
  .profession-card.selected .prof-body { color: var(--teal); }

  .licence-badge {
    display: inline-flex; align-items: center; gap: 5px;
    background: var(--teal-light); color: var(--teal-dark);
    font-size: 0.72rem; font-weight: 500;
    padding: 0.25rem 0.65rem; border-radius: 50px; margin-bottom: 0.4rem;
  }
  .licence-badge::before { content: ''; width: 5px; height: 5px; background: var(--teal); border-radius: 50%; display: block; }

  .info-box {
    background: var(--teal-light); border-radius: 10px;
    padding: 0.75rem 1rem; margin-bottom: 1.25rem;
    font-size: 0.8rem; color: var(--teal-dark); line-height: 1.6;
  }
  .info-box strong { font-weight: 500; }

  .form-group { margin-bottom: 1rem; }
  .form-group label { display: block; font-size: 0.8rem; font-weight: 500; color: var(--text); margin-bottom: 0.35rem; }
  .form-group .optional { font-size: 0.72rem; color: var(--text-muted); font-weight: 300; margin-left: 4px; }
  .form-group input {
    width: 100%; padding: 0.72rem 1rem;
    border: 1.5px solid var(--border); border-radius: 10px;
    font-family: 'DM Sans', sans-serif; font-size: 0.9rem;
    color: var(--text); background: white; outline: none; transition: border-color 0.2s;
  }
  .form-group input:focus { border-color: var(--teal); }
  .form-group input::placeholder { color: #B4B2A9; }

  .phone-wrap { display: flex; gap: 8px; }
  .phone-prefix {
    padding: 0.72rem 0.85rem; border: 1.5px solid var(--border);
    border-radius: 10px; background: white; font-size: 0.9rem;
    color: var(--text-muted); white-space: nowrap; flex-shrink: 0;
  }
  .phone-wrap input { flex: 1; }

  .otp-wrap { display: flex; gap: 10px; }
  .otp-wrap input {
    flex: 1; text-align: center; font-size: 1.3rem;
    font-weight: 500; padding: 0.75rem 0.5rem; letter-spacing: 0.1em;
  }

  .btn-main {
    width: 100%; padding: 0.82rem; background: var(--teal); color: white;
    border: none; border-radius: 10px; font-family: 'DM Sans', sans-serif;
    font-size: 0.92rem; font-weight: 500; cursor: pointer; margin-top: 0.4rem;
    transition: background 0.2s, transform 0.15s;
  }
  .btn-main:hover { background: var(--teal-dark); transform: translateY(-1px); }
  .btn-main:disabled { background: #B4B2A9; cursor: not-allowed; transform: none; }

  .btn-ghost {
    width: 100%; padding: 0.72rem; background: transparent; color: var(--teal);
    border: 1.5px solid var(--teal); border-radius: 10px;
    font-family: 'DM Sans', sans-serif; font-size: 0.88rem; font-weight: 500;
    cursor: pointer; margin-top: 0.75rem; transition: all 0.2s;
  }
  .btn-ghost:hover { background: var(--teal-light); }

  .back-link {
    display: flex; align-items: center; gap: 6px; background: none; border: none;
    color: var(--text-muted); font-size: 0.82rem; cursor: pointer;
    font-family: 'DM Sans', sans-serif; margin-bottom: 1.25rem; padding: 0;
  }
  .back-link:hover { color: var(--teal); }

  .divider { text-align: center; margin: 1rem 0; position: relative; }
  .divider::before { content: ''; position: absolute; left: 0; top: 50%; width: 100%; height: 1px; background: var(--border); }
  .divider span { position: relative; background: var(--bg); padding: 0 0.75rem; font-size: 0.78rem; color: var(--text-muted); }

  .resend { text-align: center; margin-top: 0.85rem; font-size: 0.8rem; color: var(--text-muted); }
  .resend button { background: none; border: none; color: var(--teal); cursor: pointer; font-size: 0.8rem; font-family: 'DM Sans', sans-serif; font-weight: 500; }

  .footer-note { text-align: center; margin-top: 1.25rem; font-size: 0.78rem; color: var(--text-muted); }
  .footer-note a { color: var(--teal); text-decoration: none; }

  .basic-mode-note {
    background: #FAEEDA; border-radius: 10px; padding: 0.75rem 1rem;
    font-size: 0.78rem; color: #633806; line-height: 1.6; margin-top: 1rem;
  }
  .basic-mode-note strong { font-weight: 500; }

  @media (max-width: 700px) {
    .login-page { grid-template-columns: 1fr; }
    .login-left { display: none; }
    .login-right { padding: 2rem 1.5rem; align-items: flex-start; padding-top: 2.5rem; }
  }
`

const PROFESSIONS = [
  { id: "doctor",     name: "Doctor",     icon: "🩺", body: "KMPDC",      placeholder: "KMPDC/DOC/XXXXX" },
  { id: "nurse",      name: "Nurse",      icon: "💉", body: "NCK",        placeholder: "NCK/XXXXX" },
  { id: "pharmacist", name: "Pharmacist", icon: "💊", body: "PPB",        placeholder: "PPB/PHARM/XXXXX" },
  { id: "dentist",    name: "Dentist",    icon: "🦷", body: "KMPDC",      placeholder: "KMPDC/DEN/XXXXX" },
  { id: "chw",        name: "CHW / CHV",  icon: "🏘️", body: "MOH County", placeholder: "CHW/COUNTY/XXXXX" },
]

export default function Login() {const navigate = useNavigate()
  const [role, setRole] = useState("patient")
  const [patientType, setPatientType] = useState("referred")
  const [profession, setProfession] = useState(null)
  const [step, setStep] = useState(1)
  const [phone, setPhone] = useState("")
  const [email, setEmail] = useState("")
  const [nationalId, setNationalId] = useState("")
  const [password, setPassword] = useState("")
  const [licenceNo, setLicenceNo] = useState("")
  const [facilityCode, setFacilityCode] = useState("")
  const [otp, setOtp] = useState(["", "", "", ""])

  const handleOtp = (val, idx) => {
    const next = [...otp]
    next[idx] = val.slice(-1)
    setOtp(next)
    if (val && idx < 3) document.getElementById(`otp-${idx + 1}`)?.focus()
  }

  const switchRole = (r) => {
    setRole(r); setStep(1); setProfession(null)
    setOtp(["", "", "", ""]); setPhone(""); setEmail("")
    setNationalId(""); setPassword(""); setLicenceNo(""); setFacilityCode("")
  }

  const selectedProf = PROFESSIONS.find(p => p.id === profession)

  return (
    <>
      <style>{styles}</style>
      <div className="login-page">

        {/* LEFT */}
        <div className="login-left">
          <div className="left-logo">Afya<span>Link</span></div>
          <div className="left-content">
            <h2>Healing continues <em>beyond</em> the ward.</h2>
            <p>AfyaLink connects patients, clinicians, and hospitals across Kenya — so recovery never happens alone, no matter where you are.</p>
          </div>
          <div className="left-stats">
            <div className="left-stat"><span>60%</span><p>readmissions preventable</p></div>
            <div className="left-stat"><span>72hrs</span><p>critical post-discharge window</p></div>
            <div className="left-stat"><span>47</span><p>counties covered</p></div>
          </div>
        </div>

        {/* RIGHT */}
        <div className="login-right">
          <div className="login-box">

            {/* OTP SCREEN */}
            {step === 2 ? (
              <>
                <button className="back-link" onClick={() => setStep(1)}>← Back</button>
                <h1>Enter OTP</h1>
                <p className="subtitle">We sent a 4-digit code to +254 {phone}</p>
                <div className="form-group">
                  <div className="otp-wrap">
                    {otp.map((v, i) => (
                      <input key={i} id={`otp-${i}`} type="text" inputMode="numeric"
                        maxLength={1} value={v} onChange={e => handleOtp(e.target.value, i)} />
                    ))}
                  </div>
                </div>
                <button className="btn-main" onClick={() => navigate('/dashboard')}>Verify & Sign in</button>
                {patientType === "self" && (
                  <div className="basic-mode-note">
                    ⚠️ Your account starts in <strong>basic mode</strong> until a verified clinician links you to a facility. You can still use medication reminders and health tips.
                  </div>
                )}
                <div className="resend">Didn't receive it? <button>Resend code</button></div>
              </>

            ) : (
              <>
                <h1>Welcome to AfyaLink</h1>
                <p className="subtitle">Sign in to your account</p>

                {/* TABS */}
                <div className="role-tabs">
                  {[
                    { id: "patient",   icon: "🧑", label: "Patient" },
                    { id: "clinician", icon: "👨‍⚕️", label: "Clinician" },
                    { id: "hospital",  icon: "🏥", label: "Hospital" },
                  ].map(r => (
                    <button key={r.id} className={`role-tab ${role === r.id ? "active" : ""}`}
                      onClick={() => switchRole(r.id)}>
                      <span className="tab-icon">{r.icon}</span>
                      {r.label}
                    </button>
                  ))}
                </div>

                {/* PATIENT */}
                {role === "patient" && (
                  <>
                    <div className="patient-toggle">
                      <button className={`patient-toggle-btn ${patientType === "referred" ? "active" : ""}`}
                        onClick={() => setPatientType("referred")}>Hospital referred</button>
                      <button className={`patient-toggle-btn ${patientType === "self" ? "active" : ""}`}
                        onClick={() => setPatientType("self")}>Self registered</button>
                    </div>

                    <div className="info-box">
                      {patientType === "referred" ? (
                        <><strong>Hospital-referred patient</strong><br />Your clinician registered you at discharge. Sign in with the phone number they used.</>
                      ) : (
                        <><strong>Self-registered patient</strong><br />Sign up with your phone and National ID. Your account starts in basic mode until linked to a facility.</>
                      )}
                    </div>

                    <div className="form-group">
                      <label>Phone number</label>
                      <div className="phone-wrap">
                        <div className="phone-prefix">🇰🇪 +254</div>
                        <input type="tel" placeholder="712 345 678" value={phone}
                          onChange={e => setPhone(e.target.value)} />
                      </div>
                    </div>

                    {patientType === "self" && (
                      <div className="form-group">
                        <label>National ID number</label>
                        <input type="text" placeholder="e.g. 12345678" value={nationalId}
                          onChange={e => setNationalId(e.target.value)} />
                      </div>
                    )}

                    <div className="form-group">
                      <label>Email address <span className="optional">(optional)</span></label>
                      <input type="email" placeholder="your@email.com" value={email}
                        onChange={e => setEmail(e.target.value)} />
                    </div>

                    <button className="btn-main"
                      disabled={!phone || (patientType === "self" && !nationalId)}
                      onClick={() => phone && setStep(2)}>
                      Send OTP
                    </button>
                  </>
                )}

                {/* CLINICIAN */}
                {role === "clinician" && (
                  <>
                    <p style={{ fontSize: "0.8rem", color: "var(--text-muted)", marginBottom: "0.75rem" }}>
                      Select your profession
                    </p>
                    <div className="profession-grid">
                      {PROFESSIONS.map(p => (
                        <div key={p.id}
                          className={`profession-card ${profession === p.id ? "selected" : ""}`}
                          onClick={() => { setProfession(p.id); setLicenceNo("") }}>
                          <div className="prof-icon">{p.icon}</div>
                          <div className="prof-name">{p.name}</div>
                          <div className="prof-body">{p.body}</div>
                        </div>
                      ))}
                    </div>

                    {profession && (
                      <>
                        <div className="form-group">
                          <div className="licence-badge">{selectedProf.body} verified</div>
                          <label>{selectedProf.body} Licence Number</label>
                          <input type="text" placeholder={selectedProf.placeholder}
                            value={licenceNo} onChange={e => setLicenceNo(e.target.value)} />
                        </div>
                        <div className="form-group">
                          <label>Email address</label>
                          <input type="email" placeholder="doctor@hospital.ke"
                            value={email} onChange={e => setEmail(e.target.value)} />
                        </div>
                        <div className="form-group">
                          <label>Password</label>
                          <input type="password" placeholder="••••••••"
                            value={password} onChange={e => setPassword(e.target.value)} />
                        </div>
                       <button className="btn-main" disabled={!licenceNo || !email || !password} onClick={() => navigate('/clinician-dashboard')}>
  Sign in
</button>
                        <div className="divider"><span>new to AfyaLink?</span></div>
                        <button className="btn-ghost">Request clinician access</button>
                      </>
                    )}
                  </>
                )}

                {/* HOSPITAL */}
                {role === "hospital" && (
                  <>
                    <div className="info-box">
                      <strong>Hospital administrator account</strong><br />
                      Use your MOH Facility Code to verify your institution. Every registered Kenyan facility has one — check with your county health office if unsure.
                    </div>
                    <div className="form-group">
                      <div className="licence-badge">MOH verified</div>
                      <label>MOH Facility Code</label>
                      <input type="text" placeholder="e.g. 14000" value={facilityCode}
                        onChange={e => setFacilityCode(e.target.value)} />
                    </div>
                    <div className="form-group">
                      <label>Hospital email</label>
                      <input type="email" placeholder="admin@hospital.ke" value={email}
                        onChange={e => setEmail(e.target.value)} />
                    </div>
                    <div className="form-group">
                      <label>Password</label>
                      <input type="password" placeholder="••••••••" value={password}
                        onChange={e => setPassword(e.target.value)} />
                    </div>
                    <button className="btn-main" disabled={!facilityCode || !email || !password} onClick={() => navigate('/hospital-dashboard')}>
  Sign in
</button>
                      
                    <div className="divider"><span>not registered yet?</span></div>
                    <button className="btn-ghost">Register your facility</button>
                  </>
                )}

                <p className="footer-note">
                  By signing in you agree to our <a href="#">Terms</a> and <a href="#">Privacy Policy</a>
                </p>
              </>
            )}
          </div>
        </div>
      </div>
    </>
  )
}
