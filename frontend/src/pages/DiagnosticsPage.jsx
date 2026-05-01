import React, { useEffect, useState } from 'react';

const statusClassName = (status) => {
  switch (status) {
    case 'ok':
      return 'is-success';
    case 'warning':
      return 'is-warning';
    case 'error':
      return 'is-danger';
    default:
      return 'is-light';
  }
};

const DiagnosticsPage = () => {
  const [report, setReport] = useState(null);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState('');

  useEffect(() => {
    const fetchPreflight = async () => {
      setLoading(true);
      setErrorMessage('');

      try {
        const response = await fetch('/api/preflight');
        if (!response.ok) {
          throw new Error('Could not load diagnostics.');
        }

        const data = await response.json();
        setReport(data);
      } catch (error) {
        console.error('Error loading diagnostics:', error);
        setErrorMessage(error.message || 'Could not load diagnostics.');
      } finally {
        setLoading(false);
      }
    };

    fetchPreflight();
  }, []);

  if (loading) {
    return <p>Loading diagnostics...</p>;
  }

  if (errorMessage) {
    return (
      <div className="notification is-danger is-light">
        <h5 className="title is-5">Diagnostics Error</h5>
        <p>{errorMessage}</p>
      </div>
    );
  }

  return (
    <div className="diagnostics-page">
      <div className="diagnostics-header">
        <div>
          <h2 className="title is-4 mb-2">Preflight Diagnostics</h2>
          <p className="diagnostics-subtitle">
            Validate config loading, runtime write access, and identity secrets before a run starts.
          </p>
        </div>
        <span className={`tag is-medium is-light ${statusClassName(report?.overall_status)}`}>
          {report?.overall_status || 'unknown'}
        </span>
      </div>

      <div className="diagnostics-meta-grid">
        <div className="diagnostics-meta-card">
          <span className="diagnostics-meta-label">Config Path</span>
          <strong>{report?.config_path || '-'}</strong>
        </div>
        <div className="diagnostics-meta-card">
          <span className="diagnostics-meta-label">Working Directory</span>
          <strong>{report?.working_dir || '-'}</strong>
        </div>
      </div>

      <div className="diagnostics-check-list">
        {(report?.checks || []).map((check) => (
          <div key={check.name} className="diagnostics-check-card">
            <div className="diagnostics-check-top">
              <h3 className="diagnostics-check-title">{check.name}</h3>
              <span className={`tag is-light ${statusClassName(check.status)}`}>
                {check.status}
              </span>
            </div>
            <p className="diagnostics-check-message">{check.message}</p>
          </div>
        ))}
      </div>
    </div>
  );
};

export default DiagnosticsPage;
