import { useState } from "react"
import { useNavigate } from "react-router-dom"

const styles = `
  @import url('https://fonts.googleapis.com/css2?family=DM+Serif+Display:ital@0;1&family=DM+Sans:wght@300;400;500;600&display=swap');

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
    --text: #2C2C2A;
    --text-muted: #5F5E5A;
    --border: rgba(0,0,0,0.08);
    --bg: #FAFAF8;
    --white: #ffffff;
  }

  body { font-family: 'DM Sans', sans-serif; background: var(--bg); color: var(--text); }

  .checkin-page {
    min-height: 100vh;
    background: var(--bg);
    display: flex;
    flex-direction: column;
  }

  /* TOPBAR */
  .checkin-topbar {
    background: var(--white);
    border-bottom: 1px solid var(--border);
    padding: 1rem 2rem;
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .back-btn {
    background: none;
    border: none;
    cursor: pointer;
    color: var(--text-muted);
    font-size: 0.9rem;
    display: flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.4rem 0.8rem;
    border-radius: 8px;
    transition: background 0.2s;
  }

  .back-btn:hover { background: var(--bg); }

  .topbar-title {
    font-family: 'DM Serif Display', serif;
    font-size: 1.2rem;
    color: var(--navy);
  }

  .topbar-date {
    margin-left: auto;
    font-size: 0.85rem;
    color: var(--text-muted);
    background: var(--bg);
    padding: 0.4rem 0.8rem;
    border-radius: 20px;
    border: 1px solid var(--border);
  }

  /* MAIN CONTENT */
  .checkin-content {
    max-width: 680px;
    margin: 2rem auto;
    padding: 0 1.5rem 4rem;
    width: 100%;
  }

  /* STATUS BANNER */
  .status-banner {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 1rem 1.25rem;
    border-radius: 12px;
    margin-bottom: 2rem;
    font-size: 0.9rem;
    font-weight: 500;
  }

  .status-banner.pending {
    background: var(--amber-light);
    color: var(--amber);
    border: 1px solid rgba(186,117,23,0.2);
  }

  .status-banner.done {
    background: var(--teal-light);
    color: var(--teal-dark);
    border: 1px solid rgba(29,158,117,0.2);
  }

  /* CARD */
  .checkin-card {
    background: var(--white);
    border-radius: 16px;
    border: 1px solid var(--border);
    overflow: hidden;
    margin-bottom: 1.25rem;
  }

  .card-header {
    padding: 1.25rem 1.5rem;
    border-bottom: 1px solid var(--border);
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .card-icon {
    width: 36px;
    height: 36px;
    border-radius: 10px;
    background: var(--teal-light);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 1rem;
  }

  .card-title {
    font-weight: 600;
    font-size: 0.95rem;
    color: var(--navy);
  }

  .card-subtitle {
    font-size: 0.8rem;
    color: var(--text-muted);
    margin-top: 0.1rem;
  }

  .card-body { padding: 1.5rem; }

  /* SYMPTOMS */
  .symptoms-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
    gap: 0.75rem;
    margin-bottom: 1rem;
  }

  .symptom-chip {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.6rem 0.9rem;
    border-radius: 10px;
    border: 1.5px solid var(--border);
    cursor: pointer;
    font-size: 0.85rem;
    background: var(--white);
    transition: all 0.15s;
    user-select: none;
  }

  .symptom-chip:hover { border-color: var(--teal); background: var(--teal-light); }

  .symptom-chip.selected {
    border-color: var(--teal);
    background: var(--teal-light);
    color: var(--teal-dark);
    font-weight: 500;
  }

  .symptom-chip .chip-icon { font-size: 1rem; }

  .other-input {
    width: 100%;
    padding: 0.7rem 1rem;
    border: 1.5px solid var(--border);
    border-radius: 10px;
    font-size: 0.9rem;
    font-family: inherit;
    outline: none;
    transition: border 0.2s;
    margin-top: 0.5rem;
    background: var(--bg);
  }

  .other-input:focus { border-color: var(--teal); background: var(--white); }

  /* MOOD */
  .mood-options {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 0.75rem;
  }

  .mood-btn {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.4rem;
    padding: 1rem;
    border-radius: 12px;
    border: 1.5px solid var(--border);
    cursor: pointer;
    background: var(--white);
    transition: all 0.15s;
    font-size: 0.85rem;
    color: var(--text-muted);
  }

  .mood-btn .mood-emoji { font-size: 1.8rem; }
  .mood-btn:hover { border-color: var(--teal); }

  .mood-btn.selected.good {
    border-color: var(--teal);
    background: var(--teal-light);
    color: var(--teal-dark);
    font-weight: 500;
  }

  .mood-btn.selected.okay {
    border-color: var(--amber);
    background: var(--amber-light);
    color: var(--amber);
    font-weight: 500;
  }

  .mood-btn.selected.bad {
    border-color: var(--red);
    background: var(--red-light);
    color: var(--red);
    font-weight: 500;
  }

  /* MEDICATION */
  .med-options {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.75rem;
  }

  .med-btn {
    padding: 1rem;
    border-radius: 12px;
    border: 1.5px solid var(--border);
    cursor: pointer;
    background: var(--white);
    font-size: 0.9rem;
    font-family: inherit;
    font-weight: 500;
    transition: all 0.15s;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
  }

  .med-btn:hover { border-color: var(--teal); }

  .med-btn.selected.yes {
    border-color: var(--teal);
    background: var(--teal-light);
    color: var(--teal-dark);
  }

  .med-btn.selected.no {
    border-color: var(--red);
    background: var(--red-light);
    color: var(--red);
  }

  /* PHOTO UPLOAD */
  .photo-upload-area {
    border: 2px dashed var(--border);
    border-radius: 12px;
    padding: 2rem;
    text-align: center;
    cursor: pointer;
    transition: all 0.2s;
    position: relative;
  }

  .photo-upload-area:hover { border-color: var(--teal); background: var(--teal-light); }
  .photo-upload-area input { position: absolute; inset: 0; opacity: 0; cursor: pointer; }
  .upload-icon { font-size: 2rem; margin-bottom: 0.5rem; }
  .upload-text { font-size: 0.85rem; color: var(--text-muted); }
  .upload-hint { font-size: 0.75rem; color: var(--text-muted); margin-top: 0.25rem; }

  .photo-preview {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    margin-top: 1rem;
  }

  .photo-thumb {
    width: 80px;
    height: 80px;
    border-radius: 10px;
    object-fit: cover;
    border: 2px solid var(--border);
  }

  /* NOTES */
  .notes-textarea {
    width: 100%;
    padding: 0.9rem 1rem;
    border: 1.5px solid var(--border);
    border-radius: 12px;
    font-size: 0.9rem;
    font-family: inherit;
    outline: none;
    resize: vertical;
    min-height: 100px;
    transition: border 0.2s;
    background: var(--bg);
    color: var(--text);
  }

  .notes-textarea:focus { border-color: var(--teal); background: var(--white); }

  /* SUBMIT */
  .submit-btn {
    width: 100%;
    padding: 1rem;
    background: var(--teal);
    color: white;
    border: none;
    border-radius: 12px;
    font-size: 1rem;
    font-weight: 600;
    font-family: inherit;
    cursor: pointer;
    transition: background 0.2s, transform 0.1s;
    margin-top: 0.5rem;
  }

  .submit-btn:hover { background: var(--teal-dark); }
  .submit-btn:active { transform: scale(0.99); }
  .submit-btn:disabled { background: #ccc; cursor: not-allowed; }

  /* SUCCESS STATE */
  .success-screen {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 60vh;
    text-align: center;
    padding: 2rem;
  }

  .success-icon {
    width: 80px;
    height: 80px;
    background: var(--teal-light);
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 2.5rem;
    margin-bottom: 1.5rem;
    animation: popIn 0.4s ease;
  }

  @keyframes popIn {
    0% { transform: scale(0); opacity: 0; }
    70% { transform: scale(1.1); }
    100% { transform: scale(1); opacity: 1; }
  }

  .success-title {
    font-family: 'DM Serif Display', serif;
    font-size: 1.6rem;
    color: var(--navy);
    margin-bottom: 0.5rem;
  }

  .success-subtitle {
    color: var(--text-muted);
    font-size: 0.95rem;
    margin-bottom: 0.5rem;
  }

  .success-timestamp {
    font-size: 0.8rem;
    color: var(--text-muted);
    background: var(--bg);
    padding: 0.4rem 0.9rem;
    border-radius: 20px;
    border: 1px solid var(--border);
    margin-bottom: 2rem;
  }

  .success-actions {
    display: flex;
    gap: 0.75rem;
    flex-wrap: wrap;
    justify-content: center;
  }

  .btn-outline {
    padding: 0.75rem 1.5rem;
    border-radius: 10px;
    border: 1.5px solid var(--border);
    background: var(--white);
    font-family: inherit;
    font-size: 0.9rem;
    cursor: pointer;
    transition: all 0.15s;
  }

  .btn-outline:hover { border-color: var(--teal); color: var(--teal); }

  .btn-primary {
    padding: 0.75rem 1.5rem;
    border-radius: 10px;
    border: none;
    background: var(--teal);
    color: white;
    font-family: inherit;
    font-size: 0.9rem;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.15s;
  }

  .btn-primary:hover { background: var(--teal-dark); }

  /* PAIN SLIDER */
  .pain-slider-wrap { padding: 0.5rem 0; }

  .pain-labels {
    display: flex;
    justify-content: space-between;
    font-size: 0.75rem;
    color: var(--text-muted);
    margin-bottom: 0.5rem;
  }

  .pain-slider {
    width: 100%;
    -webkit-appearance: none;
    height: 6px;
    border-radius: 3px;
    background: linear-gradient(to right, var(--teal) 0%, var(--teal) var(--val, 0%), var(--border) var(--val, 0%));
    outline: none;
    cursor: pointer;
  }

  .pain-slider::-webkit-slider-thumb {
    -webkit-appearance: none;
    width: 20px;
    height: 20px;
    border-radius: 50%;
    background: var(--teal);
    border: 2px solid white;
    box-shadow: 0 1px 4px rgba(0,0,0,0.2);
  }

  .pain-value {
    text-align: center;
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--teal);
    margin-top: 0.5rem;
  }

  .section-label {
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-bottom: 0.75rem;
  }
`

const SYMPTOMS = [
  { id: "fever", label: "Fever", icon: "🌡️" },
  { id: "cough", label: "Cough", icon: "😮‍💨" },
  { id: "headache", label: "Headache", icon: "🤕" },
  { id: "fatigue", label: "Fatigue", icon: "😴" },
  { id: "pain", label: "Pain", icon: "⚡" },
  { id: "nausea", label: "Nausea", icon: "🤢" },
  { id: "dizziness", label: "Dizziness", icon: "💫" },
  { id: "swelling", label: "Swelling", icon: "🫧" },
]

export default function CheckIn() {
  const navigate = useNavigate()
  const [submitted, setSubmitted] = useState(false)
  const [submitTime, setSubmitTime] = useState(null)

  const [symptoms, setSymptoms] = useState([])
  const [otherSymptom, setOtherSymptom] = useState("")
  const [mood, setMood] = useState(null)
  const [medTaken, setMedTaken] = useState(null)
  const [painLevel, setPainLevel] = useState(0)
  const [notes, setNotes] = useState("")
  const [photos, setPhotos] = useState([])

  const today = new Date().toLocaleDateString("en-KE", {
    weekday: "long", year: "numeric", month: "long", day: "numeric"
  })

  const toggleSymptom = (id) => {
    setSymptoms(prev =>
      prev.includes(id) ? prev.filter(s => s !== id) : [...prev, id]
    )
  }

  const handlePhoto = (e) => {
    const files = Array.from(e.target.files)
    const urls = files.map(f => URL.createObjectURL(f))
    setPhotos(prev => [...prev, ...urls])
  }

  const canSubmit = mood !== null && medTaken !== null

  const handleSubmit = () => {
    const now = new Date().toLocaleString("en-KE", {
      day: "numeric", month: "short", year: "numeric",
      hour: "2-digit", minute: "2-digit"
    })
    setSubmitTime(now)
    setSubmitted(true)
  }

  if (submitted) {
    return (
      <>
        <style>{styles}</style>
        <div className="checkin-page">
          <div className="checkin-topbar">
            <span className="topbar-title">AfyaLink</span>
          </div>
          <div className="success-screen">
            <div className="success-icon">✅</div>
            <h2 className="success-title">Check-in submitted!</h2>
            <p className="success-subtitle">Your clinician has been notified.</p>
            <div className="success-timestamp">📅 {submitTime}</div>
            <div className="success-actions">
              <button className="btn-outline" onClick={() => setSubmitted(false)}>
                View my check-in
              </button>
              <button className="btn-primary" onClick={() => navigate('/dashboard')}>
                Back to Dashboard
              </button>
            </div>
          </div>
        </div>
      </>
    )
  }

  return (
    <>
      <style>{styles}</style>
      <div className="checkin-page">

        {/* TOPBAR */}
        <div className="checkin-topbar">
          <button className="back-btn" onClick={() => navigate('/dashboard')}>
            ← Back
          </button>
          <span className="topbar-title">Daily Check-in</span>
          <span className="topbar-date">📅 {today}</span>
        </div>

        <div className="checkin-content">

          {/* STATUS */}
          <div className="status-banner pending">
            <span>⏳</span>
            <span>You haven't checked in today — please complete your daily check-in below.</span>
          </div>

          {/* SYMPTOMS */}
          <div className="checkin-card">
            <div className="card-header">
              <div className="card-icon">🩺</div>
              <div>
                <div className="card-title">Symptoms Today</div>
                <div className="card-subtitle">Select all that apply</div>
              </div>
            </div>
            <div className="card-body">
              <div className="symptoms-grid">
                {SYMPTOMS.map(s => (
                  <div
                    key={s.id}
                    className={`symptom-chip ${symptoms.includes(s.id) ? "selected" : ""}`}
                    onClick={() => toggleSymptom(s.id)}
                  >
                    <span className="chip-icon">{s.icon}</span>
                    {s.label}
                  </div>
                ))}
              </div>
              <input
                className="other-input"
                placeholder='Other symptom (e.g. "chest tightness")'
                value={otherSymptom}
                onChange={e => setOtherSymptom(e.target.value)}
              />
            </div>
          </div>

          {/* PAIN LEVEL */}
          <div className="checkin-card">
            <div className="card-header">
              <div className="card-icon">⚡</div>
              <div>
                <div className="card-title">Pain Level</div>
                <div className="card-subtitle">Rate your pain from 0 to 10</div>
              </div>
            </div>
            <div className="card-body">
              <div className="pain-slider-wrap">
                <div className="pain-labels">
                  <span>No pain</span>
                  <span>Severe pain</span>
                </div>
                <input
                  type="range"
                  className="pain-slider"
                  min="0" max="10"
                  value={painLevel}
                  style={{ "--val": `${painLevel * 10}%` }}
                  onChange={e => setPainLevel(Number(e.target.value))}
                />
                <div className="pain-value">{painLevel}/10</div>
              </div>
            </div>
          </div>

          {/* MOOD */}
          <div className="checkin-card">
            <div className="card-header">
              <div className="card-icon">😊</div>
              <div>
                <div className="card-title">How are you feeling today?</div>
                <div className="card-subtitle">Choose one</div>
              </div>
            </div>
            <div className="card-body">
              <div className="mood-options">
                {[
                  { id: "good", emoji: "😊", label: "Good" },
                  { id: "okay", emoji: "😐", label: "Okay" },
                  { id: "bad", emoji: "😞", label: "Bad" },
                ].map(m => (
                  <button
                    key={m.id}
                    className={`mood-btn ${mood === m.id ? `selected ${m.id}` : ""}`}
                    onClick={() => setMood(m.id)}
                  >
                    <span className="mood-emoji">{m.emoji}</span>
                    {m.label}
                  </button>
                ))}
              </div>
            </div>
          </div>

          {/* MEDICATION */}
          <div className="checkin-card">
            <div className="card-header">
              <div className="card-icon">💊</div>
              <div>
                <div className="card-title">Medication Adherence</div>
                <div className="card-subtitle">Did you take your medication today?</div>
              </div>
            </div>
            <div className="card-body">
              <div className="med-options">
                <button
                  className={`med-btn ${medTaken === true ? "selected yes" : ""}`}
                  onClick={() => setMedTaken(true)}
                >
                  ✅ Yes, I did
                </button>
                <button
                  className={`med-btn ${medTaken === false ? "selected no" : ""}`}
                  onClick={() => setMedTaken(false)}
                >
                  ❌ No, I didn't
                </button>
              </div>
            </div>
          </div>

          {/* PHOTO UPLOAD */}
          <div className="checkin-card">
            <div className="card-header">
              <div className="card-icon">📷</div>
              <div>
                <div className="card-title">Photo Upload</div>
                <div className="card-subtitle">Optional — wound, rash, or injury photo</div>
              </div>
            </div>
            <div className="card-body">
              <div className="photo-upload-area">
                <input type="file" accept="image/*" multiple onChange={handlePhoto} />
                <div className="upload-icon">📸</div>
                <div className="upload-text">Tap to upload a photo</div>
                <div className="upload-hint">JPG, PNG up to 10MB</div>
              </div>
              {photos.length > 0 && (
                <div className="photo-preview">
                  {photos.map((url, i) => (
                    <img key={i} src={url} className="photo-thumb" alt="upload" />
                  ))}
                </div>
              )}
            </div>
          </div>

          {/* NOTES */}
          <div className="checkin-card">
            <div className="card-header">
              <div className="card-icon">📝</div>
              <div>
                <div className="card-title">Additional Notes</div>
                <div className="card-subtitle">Optional — any extra information for your clinician</div>
              </div>
            </div>
            <div className="card-body">
              <textarea
                className="notes-textarea"
                placeholder="Any extra information for your clinician… e.g. 'The wound looks more red today'"
                value={notes}
                onChange={e => setNotes(e.target.value)}
              />
            </div>
          </div>

          {/* SUBMIT */}
          <button
            className="submit-btn"
            onClick={handleSubmit}
            disabled={!canSubmit}
          >
            {canSubmit ? "Submit Check-in ✅" : "Please select your mood and medication status"}
          </button>

        </div>
      </div>
    </>
  )
}
