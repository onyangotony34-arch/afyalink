import { BrowserRouter, Routes, Route } from "react-router-dom"
import Landing from "./Landing"
import Login from "./login"
import Dashboard from "./Dashboard"
import CheckIn from "./CheckIn"

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Landing />} />
        <Route path="/login" element={<Login />} />
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/checkin" element={<CheckIn />} />
        <Route path="/clinician-dashboard" element={<h1>Clinician Dashboard - Coming Soon</h1>} />
        <Route path="/hospital-dashboard" element={<h1>Hospital Dashboard - Coming Soon</h1>} />
      </Routes>
    </BrowserRouter>
  )
}