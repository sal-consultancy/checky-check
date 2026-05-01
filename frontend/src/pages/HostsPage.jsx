import React, { useEffect, useMemo, useState } from 'react';
import { FaChevronDown, FaChevronUp, FaTimes } from 'react-icons/fa';
import CheckHistoryModal from '../components/CheckHistoryModal';

const formatErrorType = (errorType) => {
  if (!errorType) return '';
  return errorType.replaceAll('_', ' ');
};

const Sparkline = ({ points }) => {
  if (!points || points.length < 2) {
    return <span className="host-sparkline-empty">--</span>;
  }

  const width = 92;
  const height = 24;
  const padding = 2;
  const values = points.map((point) => point.value);
  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;

  const polylinePoints = points.map((point, index) => {
    const x = padding + (index * (width - padding * 2)) / Math.max(points.length - 1, 1);
    const y = height - padding - ((point.value - min) / range) * (height - padding * 2);
    return `${x},${y}`;
  }).join(' ');

  const isRising = values[values.length - 1] > values[0];

  return (
    <svg className="host-sparkline" viewBox={`0 0 ${width} ${height}`} aria-hidden="true">
      <polyline
        fill="none"
        stroke={isRising ? '#c0392b' : '#4d8650'}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        points={polylinePoints}
      />
    </svg>
  );
};

const HostsPage = ({ results, checks, status }) => {
  const [expandedHosts, setExpandedHosts] = useState({});
  const [showOnlyFailedHosts, setShowOnlyFailedHosts] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [sparklineData, setSparklineData] = useState({});
  const [detailTarget, setDetailTarget] = useState(null);

  useEffect(() => {
    fetch('/api/history/sparklines?limit=14')
      .then((response) => response.ok ? response.json() : {})
      .then((data) => setSparklineData(data || {}))
      .catch(() => setSparklineData({}));
  }, []);

  const hosts = useMemo(() => {
    return Object.keys(results).map((host) => {
      const hostResults = results[host] || {};
      const checkRows = Object.keys(hostResults).map((checkName) => ({
        checkName,
        title: checks[checkName]?.title || checkName,
        description: checks[checkName]?.description || '',
        sparkline: checks[checkName]?.sparkline || {},
        sparklinePoints: sparklineData?.[host]?.[checkName] || [],
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
  }, [results, checks, sparklineData]);

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
                      <th>Trend</th>
                      <th>Issue</th>
                      <th>Timestamp</th>
                      <th className="no-print">History</th>
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
                          {row.sparkline?.enabled ? <Sparkline points={row.sparklinePoints} /> : '--'}
                        </td>
                        <td>
                          {row.error_type ? (
                            <>
                              <span className="tag is-warning is-light">{formatErrorType(row.error_type)}</span>
                              {row.error_message && <div className="error-detail-text">{row.error_message}</div>}
                            </>
                          ) : '--'}
                        </td>
                        <td>{row.timestamp}</td>
                        <td className="no-print">
                          <button
                            className="button is-small is-light"
                            onClick={() => setDetailTarget({
                              host: host.host,
                              checkName: row.checkName,
                              checkTitle: row.title,
                              status: row.status,
                              value: row.value,
                            })}
                          >
                            View history
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        ))}
      </div>

      <CheckHistoryModal detailTarget={detailTarget} onClose={() => setDetailTarget(null)} />
    </div>
  );
};

export default HostsPage;
