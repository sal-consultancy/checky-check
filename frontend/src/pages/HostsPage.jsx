import React, { useMemo, useState } from 'react';
import { FaChevronDown, FaChevronUp, FaTimes } from 'react-icons/fa';

const formatErrorType = (errorType) => {
  if (!errorType) return '';
  return errorType.replaceAll('_', ' ');
};

const HostsPage = ({ results, checks, status }) => {
  const [expandedHosts, setExpandedHosts] = useState({});
  const [showOnlyFailedHosts, setShowOnlyFailedHosts] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');

  const hosts = useMemo(() => {
    return Object.keys(results).map((host) => {
      const hostResults = results[host] || {};
      const checkRows = Object.keys(hostResults).map((checkName) => ({
        checkName,
        title: checks[checkName]?.title || checkName,
        description: checks[checkName]?.description || '',
        ...hostResults[checkName],
      }));

      const passedCount = checkRows.filter((item) => item.status === 'passed').length;
      const failedCount = checkRows.filter((item) => item.status === 'failed').length;

      return {
        host,
        checkRows,
        passedCount,
        failedCount,
        totalCount: checkRows.length,
      };
    });
  }, [results, checks]);

  if (status === 'config_error') {
    return (
      <div className="notification is-danger is-light">
        <h5 className="is-size-5 write py-2">Configuration Error</h5>
        <p>The host view is unavailable because the latest run failed validation.</p>
      </div>
    );
  }

  if (hosts.length === 0) {
    return (
      <div className="notification is-warning is-light">
        No host results are available yet.
      </div>
    );
  }

  const filteredHosts = hosts.filter((host) => {
    const matchesSearch = host.host.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesFailedFilter = !showOnlyFailedHosts || host.failedCount > 0;
    return matchesSearch && matchesFailedFilter;
  });

  const toggleHost = (hostName) => {
    setExpandedHosts((prevState) => ({
      ...prevState,
      [hostName]: !prevState[hostName],
    }));
  };

  return (
    <div className="hosts-page">
      <div className="no-print">
        <h6 className="is-size-6 write my-3">Host Filter</h6>
        <div className="buttons-container mb-5">
          <button onClick={() => setShowOnlyFailedHosts((prev) => !prev)} className="button is-grey is-light is-small">
            {showOnlyFailedHosts ? 'Show All Hosts' : 'Show Only Failed Hosts'}
          </button>
        </div>
        <div className="field has-addons">
          <div className="control is-expanded">
            <input
              type="text"
              className="input is-small"
              placeholder="Search hosts..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />
          </div>
          <div className="control">
            <button className="button is-small" style={{ height: '100%' }} onClick={() => setSearchTerm('')}>
              <FaTimes />
            </button>
          </div>
        </div>
        <hr className="separator" />
      </div>

      <div className="host-summary-grid">
        {filteredHosts.map((host) => (
          <div key={host.host} className="host-summary-card">
            <div className="host-summary-top">
              <h4 className="is-size-5" id={host.host}>{host.host}</h4>
              <button
                className="no-print button is-small"
                onClick={() => toggleHost(host.host)}
                style={{ cursor: 'pointer', color: '#3273dc', background: 'none', border: 'none' }}
              >
                {expandedHosts[host.host] ? <FaChevronUp /> : <FaChevronDown />}
              </button>
            </div>
            <div className="host-summary-tags">
              <span className="tag is-light">Checks: {host.totalCount}</span>
              <span className="tag is-success is-light">Passed: {host.passedCount}</span>
              <span className="tag is-danger is-light">Failed: {host.failedCount}</span>
            </div>

            {expandedHosts[host.host] && (
              <div className="mt-4">
                <table className="table is-striped is-bordered is-size-7 is-fullwidth">
                  <thead>
                    <tr>
                      <th>Check</th>
                      <th>Status</th>
                      <th>Value</th>
                      <th>Issue</th>
                      <th>Timestamp</th>
                    </tr>
                  </thead>
                  <tbody>
                    {host.checkRows.map((row) => (
                      <tr key={`${host.host}-${row.checkName}`}>
                        <td>
                          <strong>{row.title}</strong>
                          {row.description && <div>{row.description}</div>}
                        </td>
                        <td>{row.status}</td>
                        <td>{row.value}</td>
                        <td>
                          {row.error_type ? (
                            <>
                              <span className="tag is-warning is-light">{formatErrorType(row.error_type)}</span>
                              {row.error_message && <div className="error-detail-text">{row.error_message}</div>}
                            </>
                          ) : '—'}
                        </td>
                        <td>{row.timestamp}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
};

export default HostsPage;
