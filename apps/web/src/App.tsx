import { Navigate, Route, Routes, useParams } from 'react-router-dom'
import { Layout } from './components/Layout'
import { ProtectedRoute } from './components/ProtectedRoute'
import { ApplicationDetailPage } from './pages/ApplicationDetailPage'
import { ApplicationsPage } from './pages/ApplicationsPage'
import { LoginPage } from './pages/LoginPage'
import { OpportunitiesPage } from './pages/OpportunitiesPage'
import { OpportunityDetailPage } from './pages/OpportunityDetailPage'
import { NotificationsPage } from './pages/NotificationsPage'
import { ProfilePage } from './pages/ProfilePage'
import { RegisterPage } from './pages/RegisterPage'
import { RecommendedPage } from './pages/RecommendedPage'
import { ResearchVerificationPage } from './pages/ResearchVerificationPage'
import { AdminPage } from './pages/AdminPage'
import { SavedPage } from './pages/SavedPage'

function LegacyOpportunityDetailRedirect() {
  const { id } = useParams<{ id: string }>()
  return (
    <Navigate
      to={`/browse/${id}`}
      replace
      state={{ from: 'browse', browseType: 'employment' }}
    />
  )
}

function LegacyResearchDetailRedirect() {
  const { id } = useParams<{ id: string }>()
  return (
    <Navigate
      to={`/browse/${id}`}
      replace
      state={{ from: 'browse', browseType: 'research' }}
    />
  )
}

export function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<Navigate to="/browse" replace />} />
        <Route path="login" element={<LoginPage />} />
        <Route path="register" element={<RegisterPage />} />

        <Route element={<ProtectedRoute />}>
          <Route path="recommended" element={<RecommendedPage />} />
          <Route path="browse" element={<OpportunitiesPage />} />
          <Route path="browse/:id" element={<OpportunityDetailPage />} />
          <Route path="opportunities" element={<Navigate to="/browse" replace />} />
          <Route path="opportunities/:id" element={<LegacyOpportunityDetailRedirect />} />
          <Route path="research" element={<Navigate to="/browse?type=research" replace />} />
          <Route path="research/:id" element={<LegacyResearchDetailRedirect />} />
          <Route path="admin" element={<AdminPage />} />
          <Route path="admin/research" element={<ResearchVerificationPage />} />
          <Route path="saved" element={<SavedPage />} />
          <Route path="applications" element={<ApplicationsPage />} />
          <Route path="applications/:id" element={<ApplicationDetailPage />} />
          <Route path="profile" element={<ProfilePage />} />
          <Route path="notifications" element={<NotificationsPage />} />
        </Route>
      </Route>
    </Routes>
  )
}
