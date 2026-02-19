import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { useEffect } from 'react';
import { useAuthStore } from './stores/authStore';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import APIKeys from './pages/APIKeys';
import Endpoints from './pages/Endpoints';
import Layout from './components/Layout';

function App() {
  const { checkAuth, isAuthenticated, isLoading } = useAuthStore();

  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-100">
        <div className="text-xl">Loading...</div>
      </div>
    );
  }

  return (
    <BrowserRouter>
      <Routes>
        <Route
          path="/login"
          element={!isAuthenticated ? <Login /> : <Navigate to="/" />}
        />
        <Route
          path="/"
          element={
            isAuthenticated ? (
              <Layout>
                <Dashboard />
              </Layout>
            ) : (
              <Navigate to="/login" />
            )
          }
        />
        <Route
          path="/api-keys"
          element={
            isAuthenticated ? (
              <Layout>
                <APIKeys />
              </Layout>
            ) : (
              <Navigate to="/login" />
            )
          }
        />
        <Route
          path="/endpoints"
          element={
            isAuthenticated ? (
              <Layout>
                <Endpoints />
              </Layout>
            ) : (
              <Navigate to="/login" />
            )
          }
        />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
