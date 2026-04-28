import { useState, useEffect } from 'react';
import { Link, Outlet, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../auth';
import { useTheme } from '../theme';
import { api } from '../api';

function SunIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="5" />
      <line x1="12" y1="1" x2="12" y2="3" />
      <line x1="12" y1="21" x2="12" y2="23" />
      <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
      <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
      <line x1="1" y1="12" x2="3" y2="12" />
      <line x1="21" y1="12" x2="23" y2="12" />
      <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
      <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
    </svg>
  );
}

function MoonIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
    </svg>
  );
}

export default function Layout() {
  const { logout } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const { theme, toggle } = useTheme();
  const [activeJobs, setActiveJobs] = useState<{ total: number; running: number; pending: number } | null>(null);

  useEffect(() => {
    const poll = () => {
      api.getActiveJobs().then(setActiveJobs).catch(() => {});
    };
    poll();
    const interval = setInterval(poll, 10000);
    return () => clearInterval(interval);
  }, []);

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const linkClass = (path: string) => {
    const active =
      path === '/runs'
        ? location.pathname === '/runs' || location.pathname.startsWith('/runs/')
        : location.pathname === path;
    return `nav-link${active ? ' active' : ''}`;
  };

  return (
    <>
      <nav className="nav">
        <div className="nav-inner">
          <div className="nav-left">
            <Link to="/" className="nav-brand">
              <div className="nav-brand-icon">M</div>
              <span className="nav-brand-text">markov<span className="nav-brand-d">d</span></span>
            </Link>
            <div className="nav-links">
              <Link to="/runs" className={linkClass('/runs')}>Runs</Link>
              <Link to="/jobs" className={linkClass('/jobs')}>Jobs</Link>
              <Link to="/workflows" className={linkClass('/workflows')}>Workflows</Link>
              <Link to="/projects" className={linkClass('/projects')}>Projects</Link>
              <Link to="/trigger" className={linkClass('/trigger')}>Trigger</Link>
              <Link to="/settings" className={linkClass('/settings')}>Settings</Link>
            </div>
          </div>
          <div className="nav-right">
            {activeJobs && activeJobs.total > 0 && (
              <Link to="/jobs" className="nav-jobs-badge" title={`${activeJobs.running} running, ${activeJobs.pending} pending`}>
                {activeJobs.total} job{activeJobs.total !== 1 ? 's' : ''}
              </Link>
            )}
            <button
              onClick={toggle}
              className="theme-toggle"
              title={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
            >
              {theme === 'dark' ? <SunIcon /> : <MoonIcon />}
            </button>
            <button onClick={handleLogout} className="btn btn-ghost btn-sm">
              Logout
            </button>
          </div>
        </div>
      </nav>
      <main className="page page-enter">
        <Outlet />
      </main>
    </>
  );
}
