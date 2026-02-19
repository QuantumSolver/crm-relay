import { useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { useAuthStore } from './stores/authStore';
import Layout from './components/Layout';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Config from './pages/Config';
import RetryConfig from './pages/RetryConfig';
import DLQ from './pages/DLQ';

function App() {
  const { checkAuth, isAuthenticated, isLoading } = useAuthStore();

  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-lg">Loading...</div>
      </div>
    );
  }

  return (
    <BrowserRouter>
      <Routes>
        <Route
          path="/login"
          element={!isAuthenticated ? <Login /> : <Navigate to="/" replace />}
        />
        <Route
          path="/"
          element={
            isAuthenticated ? (
              <Layout>
                <Dashboard />
              </Layout>
            ) : (
              <Navigate to="/login" replace />
            )
          }
        />
        <Route
          path="/config"
          element={
            isAuthenticated ? (
              <Layout>
                <Config />
              </Layout>
            ) : (
              <Navigate to="/login" replace />
            )
          }
        />
        <Route
          path="/retry-config"
          element={
            isAuthenticated ? (
              <Layout>
                <RetryConfig />
              </Layout>
            ) : (
              <Navigate to="/login" replace />
            )
          }
        />
        <Route
          path="/dlq"
          element={
            isAuthenticated ? (
              <Layout>
                <DLQ />
              </Layout>
            ) : (
              <Navigate to="/login" replace />
            )
          }
        />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
