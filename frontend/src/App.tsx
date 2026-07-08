import { Navigate, Route, Routes } from 'react-router-dom'
import { ActivityPage } from './pages/app/ActivityPage'
import { HomePage } from './pages/app/HomePage'
import { ProfilePage } from './pages/app/ProfilePage'
import { QrPage } from './pages/app/QrPage'
import { LoginPage } from './pages/LoginPage'
import { RegisterPage } from './pages/RegisterPage'
import { WelcomePage } from './pages/WelcomePage'
import { AppLayout } from './routes/AppLayout'
import { ProtectedRoute } from './routes/ProtectedRoute'
import { useBackButton } from './hooks/useBackButton'
import { useThemeSync } from './hooks/useThemeSync'

function App() {
  useBackButton()
  useThemeSync()

  return (
    <Routes>
      <Route path="/" element={<Navigate to="/welcome" replace />} />
      <Route path="/welcome" element={<WelcomePage />} />
      <Route path="/register" element={<RegisterPage />} />
      <Route path="/login" element={<LoginPage />} />
      <Route
        path="/app"
        element={
          <ProtectedRoute>
            <AppLayout />
          </ProtectedRoute>
        }
      >
        <Route index element={<Navigate to="home" replace />} />
        <Route path="home" element={<HomePage />} />
        <Route path="qr" element={<QrPage />} />
        <Route path="activity" element={<ActivityPage />} />
        <Route path="profile" element={<ProfilePage />} />
      </Route>
      <Route path="*" element={<Navigate to="/welcome" replace />} />
    </Routes>
  )
}

export default App
