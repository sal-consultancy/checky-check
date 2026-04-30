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
import ThemeToggle from './components/ThemeToggle';

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
    report: {},
    status: '',
    errors: [],
    generated_at: '',
  });
  const [themePreference, setThemePreference] = useState(() => localStorage.getItem('theme-preference') || 'system');
  const [systemTheme, setSystemTheme] = useState(() => (
    window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  ));

  const theme = themePreference === 'system' ? systemTheme : themePreference;

  const fetchResults = () => {
    fetch('/results')
      .then(response => response.json())
      .then(data => setResults(data))
      .catch(error => console.error('Error fetching results:', error));
  };

  useEffect(() => {
    fetchResults();
  }, []);

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
                  <p className="title header-brand-title">{results.report.title}</p>
                </div>
                <p className="subtitle brand-script">{results.report.subtitle}</p>
                <PageMenu />
              </div>
              <div className="hero-theme-control no-print">
                <ThemeToggle value={themePreference} onChange={setThemePreference} />
              </div>
            </div>
            {results.report.description && (
              <p className="report-description mt-4">{results.report.description}</p>
            )}
          </div>
        </section>
        <section className="app-content-shell">
          {hasErrors && (
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
          <Routes>
            <Route path="/report" element={<CheckReport results={results.results} checks={visibleChecks} urlResults={results.url_results} urlChecks={visibleURLChecks} theme={theme} status={results.status} />} />
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
            <Route path="/help" element={<HelpPage />} />
            <Route path="/templates" element={<CheckTemplatesPage />} />
            <Route path="/run-tests" element={<RunTestsPage onTestsComplete={handleTestsComplete} />} />
          </Routes>
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
