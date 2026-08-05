import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'

import AppShell from './components/AppShell'
import { RedirectIfAuthenticated, RequireAuth } from './components/RouteGuard'
import AuthProvider from './components/AuthProvider'
import { ROLES } from './lib/roles'

import Landing from './pages/Landing'
import Login from './pages/Login'
import ClinicianAlerts from './pages/clinician/Alerts'
import ClinicianDashboard from './pages/clinician/Dashboard'
import PatientDetail from './pages/clinician/PatientDetail'
import CaregiverDashboard from './pages/caregiver/Dashboard'
import PatientCheckIn from './pages/patient/CheckIn'
import PatientDashboard from './pages/patient/Dashboard'

/**
 * Route table.
 *
 * The three role sections are separated at the URL level (/clinician, /caregiver,
 * /patient) so a guard can be applied once per section rather than per page.
 * These guards decide navigation only — the API authorises every request on its
 * own, and nothing here is load-bearing for access control.
 */
function Guarded({ role, children }) {
  return (
    <RequireAuth roles={[role]}>
      <AppShell>{children}</AppShell>
    </RequireAuth>
  )
}

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/" element={<Landing />} />

          <Route
            path="/login"
            element={
              <RedirectIfAuthenticated>
                <Login />
              </RedirectIfAuthenticated>
            }
          />

          {/* Clinician */}
          <Route
            path="/clinician"
            element={
              <Guarded role={ROLES.clinician}>
                <ClinicianDashboard />
              </Guarded>
            }
          />
          <Route
            path="/clinician/alerts"
            element={
              <Guarded role={ROLES.clinician}>
                <ClinicianAlerts />
              </Guarded>
            }
          />
          <Route
            path="/clinician/patients/:patientId"
            element={
              <Guarded role={ROLES.clinician}>
                <PatientDetail />
              </Guarded>
            }
          />

          {/* Caregiver */}
          <Route
            path="/caregiver"
            element={
              <Guarded role={ROLES.caregiver}>
                <CaregiverDashboard />
              </Guarded>
            }
          />

          {/* Patient */}
          <Route
            path="/patient"
            element={
              <Guarded role={ROLES.patient}>
                <PatientDashboard />
              </Guarded>
            }
          />
          <Route
            path="/patient/checkin"
            element={
              <Guarded role={ROLES.patient}>
                <PatientCheckIn />
              </Guarded>
            }
          />

          {/* Anything else goes home rather than to a blank screen. */}
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  )
}
