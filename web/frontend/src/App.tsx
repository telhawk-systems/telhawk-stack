import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider } from './components/AuthProvider';
import { ProtectedRoute } from './components/ProtectedRoute';
import { LoginPage } from './pages/LoginPage';
import { DashboardPage } from './pages/DashboardPage';
import { UsersPage } from './pages/UsersPage';
import { TokensPage } from './pages/TokensPage';
import { EventsPage } from './pages/EventsPage';
import { RulesPage } from './pages/RulesPage';
import { RuleDetailPage } from './pages/RuleDetailPage';
import { AlertsPage } from './pages/AlertsPage';
import { EntityDetailPage } from './pages/EntityDetailPage';
import { SavedSearchesPage } from './pages/SavedSearchesPage';
import { SavedSearchDetailPage } from './pages/SavedSearchDetailPage';

function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route
            path="/"
            element={
              <ProtectedRoute>
                <DashboardPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/users"
            element={
              <ProtectedRoute>
                <UsersPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/tokens"
            element={
              <ProtectedRoute>
                <TokensPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/events"
            element={
              <ProtectedRoute>
                <EventsPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/rules"
            element={
              <ProtectedRoute>
                <RulesPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/rules/:id"
            element={
              <ProtectedRoute>
                <RuleDetailPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/alerts"
            element={
              <ProtectedRoute>
                <AlertsPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/saved-searches"
            element={
              <ProtectedRoute>
                <SavedSearchesPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/saved-searches/:id"
            element={
              <ProtectedRoute>
                <SavedSearchDetailPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/entity/:entityType/:entityValue"
            element={
              <ProtectedRoute>
                <EntityDetailPage />
              </ProtectedRoute>
            }
          />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  );
}

export default App;
