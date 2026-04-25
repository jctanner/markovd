import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, ProtectedRoute } from './auth';
import Layout from './components/Layout';
import Login from './pages/Login';
import Runs from './pages/Runs';
import RunDetail from './pages/RunDetail';
import Workflows from './pages/Workflows';
import Projects from './pages/Projects';
import TriggerRun from './pages/TriggerRun';

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route element={<ProtectedRoute><Layout /></ProtectedRoute>}>
            <Route path="/" element={<Navigate to="/runs" replace />} />
            <Route path="/runs" element={<Runs />} />
            <Route path="/runs/:runID" element={<RunDetail />} />
            <Route path="/workflows" element={<Workflows />} />
            <Route path="/projects" element={<Projects />} />
            <Route path="/trigger" element={<TriggerRun />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  );
}
