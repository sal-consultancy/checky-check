import React, { useEffect, useState } from 'react';
import CheckReport from './components/CheckReport';
import './App.css';
import 'bulma/css/bulma.css';
import PageMenu from './components/PageMenu';
import Footer from './components/Footer';
import heartIcon from './images/heartpulse.svg';
import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';
import HelpPage from './pages/HelpPage';
import SummaryPage from './pages/SummaryPage';
import RunTestsPage from './pages/RunTestsPage';  // Importeer de nieuwe component
import CheckTemplatesPage from './pages/CheckTemplatesPage';  // Importeer de nieuwe component
import HostsPage from './pages/HostsPage';
import HistoryPage from './pages/HistoryPage';
import DiagnosticsPage from './pages/DiagnosticsPage';
import AvatarMenu from './components/AvatarMenu';

const formatErrorType = (errorType) => {
  if (!errorType) return '';
  return errorType.replaceAll('_', ' ');
};

const AppShell = () => {
  const [results, setResults] = useState({
    checks: {},
    results: {},
    url_checks: {},
    url_results: {},
    report: {
      title: 'Checky Check',
      subtitle: '',
    },
    status: '',
    errors: [],
    generated_at: '',
  });
  const [authSession, setAuthSession] = useState({
    mode: 'loading',
    authenticated: false,
    role: 'viewer',
    permissions: {
      view: false,
      operate: false,
      admin: false,
    },
  });
  const [authLoaded, setAuthLoaded] = useState(false);
  const [themePreference, setThemePreference] = useState(() => localStorage.getItem('theme-preference') || 'system');
  const [systemTheme, setSystemTheme] = useState(() => (
    window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  ));

  const theme = themePreference === 'system' ? systemTheme : themePreference;

  const fetchResults = () => {
    fetch('/results')
      .then(response => {
        if (response.status === 401) {
          return Promise.resolve(null);
        }
        return response.json();
      })
      .then(data => {
        if (data) {
          setResults(data);
        }
      })
      .catch(error => console.error('Error fetching results:', error));
  };

  useEffect(() => {
    fetch('/api/auth/session')
      .then((response) => response.ok ? response.json() : Promise.reject(new Error('Could not load auth session.')))
      .then((data) => setAuthSession(data))
      .catch((error) => {
        console.error('Error fetching auth session:', error);
        setAuthSession({
          mode: 'proxy',
          authenticated: false,
          role: 'unauthenticated',
          permissions: {
            view: false,
            operate: false,
            admin: false,
          },
        });
      });
  }, []);

  useEffect(() => {
    if (authSession.mode === 'loading') {
      return;
    }
    setAuthLoaded(true);
  }, [authSession]);

  useEffect(() => {
    if (!authLoaded) {
      return;
    }

    if (authSession.mode === 'proxy' && !authSession.authenticated) {
      return;
    }

    fetchResults();
  }, [authLoaded, authSession.mode, authSession.authenticated]);

  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const updateSystemTheme = (event) => {
      setSystemTheme(event.matches ? 'dark' : 'light');
    };

    setSystemTheme(mediaQuery.matches ? 'dark' : 'light');
    mediaQuery.addEventListener('change', updateSystemTheme);

    return () => {
      mediaQuery.removeEventListener('change', updateSystemTheme);
    };
  }, []);

  useEffect(() => {
    localStorage.setItem('theme-preference', themePreference);
  }, [themePreference]);

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
  }, [theme]);

  useEffect(() => {
    const styleId = 'report-css-overrides';
    let styleElement = document.getElementById(styleId);

    if (!styleElement) {
      styleElement = document.createElement('style');
      styleElement.id = styleId;
      document.head.appendChild(styleElement);
    }

    styleElement.textContent = results.report?.css || '';

    return () => {
      if (styleElement) {
        styleElement.textContent = '';
      }
    };
  }, [results.report?.css]);

  const handleTestsComplete = () => {
    fetchResults();
  };

  const hasErrors = results.status === 'config_error' || (results.errors && results.errors.length > 0);
  const hostCount = Object.keys(results.results || {}).length;
  const executedCheckNames = Array.from(
    new Set(
      Object.values(results.results || {}).flatMap((hostResults) => Object.keys(hostResults || {}))
    )
  );
  const visibleChecks = executedCheckNames.reduce((acc, checkName) => {
    if (results.checks?.[checkName]) {
      acc[checkName] = results.checks[checkName];
    }
    return acc;
  }, {});
  const visibleURLChecks = Object.keys(results.url_results || {}).reduce((acc, checkName) => {
    if (results.url_checks?.[checkName]) {
      acc[checkName] = results.url_checks[checkName];
    }
    return acc;
  }, {});
  const checkCount = Object.keys(visibleChecks).length;
  const executedChecks = Object.values(results.results || {}).reduce((total, hostResults) => total + Object.keys(hostResults).length, 0);
  const failedChecks = Object.values(results.results || {}).reduce(
    (total, hostResults) => total + Object.values(hostResults).filter(result => result.status === 'failed').length,
    0
  );
  const passedChecks = executedChecks - failedChecks;
  const errorTypeCounts = Object.values(results.results || {}).reduce((acc, hostResults) => {
    Object.values(hostResults).forEach((result) => {
      if (!result.error_type) return;
      acc[result.error_type] = (acc[result.error_type] || 0) + 1;
    });
    return acc;
  }, {});
  Object.values(results.url_results || {}).forEach((result) => {
    if (!result.error_type) return;
    errorTypeCounts[result.error_type] = (errorTypeCounts[result.error_type] || 0) + 1;
  });
  const errorSummary = Object.entries(errorTypeCounts)
    .sort((a, b) => b[1] - a[1])
    .map(([type, count]) => ({ type, label: formatErrorType(type), count }));
  const requiresAuthentication = authLoaded && authSession?.mode === 'proxy' && !authSession?.authenticated;
  const requiresAuthorization = authLoaded && authSession?.mode === 'proxy' && authSession?.authenticated && !authSession?.permissions?.view;
  const canRenderNavigation = authLoaded && !requiresAuthentication && !requiresAuthorization;
  const headerTitle = results.report?.title || 'Checky Check';
  const headerSubtitle = requiresAuthentication ? 'Protected environment' : (results.report?.subtitle || '');

  return (
    <>
      <div className={`App ${theme}`}>
        <section className={`hero is-${theme}`}>
          <div className="hero-body">
            <div className="hero-header-row">
              <div className="hero-header-spacer" aria-hidden="true" />
              <div className="hero-brand-block">
                <div className="header-brand">
                  <img className="header-brand-icon" src={heartIcon} alt="court icon" width="50" />
                  <p className="title header-brand-title">{headerTitle}</p>
                </div>
                {headerSubtitle && <p className="subtitle brand-script">{headerSubtitle}</p>}
                {canRenderNavigation && <PageMenu />}
              </div>
              <div className="hero-theme-control no-print">
                <AvatarMenu
                  authSession={authSession}
                  themePreference={themePreference}
                  onThemeChange={setThemePreference}
                />
              </div>
            </div>
            {canRenderNavigation && results.report.description && (
              <p className="report-description mt-4">{results.report.description}</p>
            )}
          </div>
        </section>
        <section className="app-content-shell">
          {!authLoaded && (
            <div className="notification is-light">
              Loading session…
            </div>
          )}
          {requiresAuthentication && (
            <div className="auth-guard-card">
              <h3 className="title is-4">Sign in required</h3>
              <p className="auth-guard-copy">
                This CheckyCheck environment is protected by proxy authentication. Sign in through the access proxy to continue.
              </p>
              {authSession?.logout_url && (
                <div className="buttons is-centered mt-4">
                  <a className="button is-dark" href={authSession.logout_url}>Go to sign in</a>
                </div>
              )}
            </div>
          )}
          {requiresAuthorization && (
            <div className="auth-guard-card">
              <h3 className="title is-4">Access denied</h3>
              <p className="auth-guard-copy">
                Your account is authenticated, but it does not belong to a CheckyCheck viewer, operator, or admin group.
              </p>
              {authSession?.logout_url && (
                <div className="buttons is-centered mt-4">
                  <a className="button is-light" href={authSession.logout_url}>Switch account</a>
                </div>
              )}
            </div>
          )}
          {!requiresAuthentication && hasErrors && (
            <div className="notification is-danger is-light">
              <h4 className="title is-5">Configuration Error</h4>
              <p className="mb-3">The latest run could not start because the configuration is invalid.</p>
              <ul>
                {results.errors.map((message, index) => (
                  <li key={index}>{message}</li>
                ))}
              </ul>
            </div>
          )}
          {authLoaded && !requiresAuthentication && !requiresAuthorization && (
            <Routes>
              <Route path="/report" element={<CheckReport results={results.results} checks={visibleChecks} urlResults={results.url_results} urlChecks={visibleURLChecks} theme={theme} status={results.status} authSession={authSession} />} />
              <Route
                path="/"
                element={
                  <SummaryPage
                    results={results.results}
                    checks={visibleChecks}
                    urlResults={results.url_results}
                    urlChecks={visibleURLChecks}
                    status={results.status}
                    stats={{ hostCount, checkCount, executedChecks, failedChecks, passedChecks }}
                    errorSummary={errorSummary}
                  />
                }
              />
              <Route path="/hosts" element={<HostsPage results={results.results} checks={visibleChecks} status={results.status} />} />
              <Route path="/history" element={<HistoryPage />} />
              <Route path="/diagnostics" element={<DiagnosticsPage />} />
              <Route path="/help" element={<HelpPage />} />
              <Route path="/templates" element={<CheckTemplatesPage />} />
              <Route path="/run-tests" element={<RunTestsPage onTestsComplete={handleTestsComplete} authSession={authSession} />} />
            </Routes>
          )}
        </section>
      </div>
      <Footer copyright={results.report?.copyright} />
    </>
  );
};

const App = () => (
  <Router>
    <AppShell />
  </Router>
);

export default App;
