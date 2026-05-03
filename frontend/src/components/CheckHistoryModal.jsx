import React, { useEffect, useMemo, useState } from 'react';
import { formatFailValues, formatFailWhen } from '../utils/checkFormatting';

const formatErrorType = (errorType) => {
  if (!errorType) return '';
  return errorType.replaceAll('_', ' ');
};

const StatusTrendChart = ({ points }) => {
  if (!points || points.length < 2) {
    return <p className="history-modal-empty">Not enough numeric history yet.</p>;
  }

  const width = 560;
  const height = 180;
  const paddingX = 12;
  const paddingY = 16;
  const values = points.map((point) => point.value);
  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;
  const oldestPoint = points[0];
  const latestPoint = points[points.length - 1];

  const polylinePoints = points.map((point, index) => {
    const x = paddingX + (index * (width - paddingX * 2)) / Math.max(points.length - 1, 1);
    const y = height - paddingY - ((point.value - min) / range) * (height - paddingY * 2);
    return `${x},${y}`;
  }).join(' ');

  const dots = points.map((point, index) => {
    const x = paddingX + (index * (width - paddingX * 2)) / Math.max(points.length - 1, 1);
    const y = height - paddingY - ((point.value - min) / range) * (height - paddingY * 2);
    return {
      key: `${point.run_id}-${point.generated_at}-${index}`,
      x,
      y,
      status: point.status,
    };
  });

  return (
    <div className="history-modal-chart-shell">
      <svg className="history-modal-chart" viewBox={`0 0 ${width} ${height}`} aria-hidden="true">
        <polyline
          fill="none"
          stroke="#4d8650"
          strokeWidth="2.5"
          strokeLinecap="round"
          strokeLinejoin="round"
          points={polylinePoints}
        />
        {dots.map((dot) => (
          <circle
            key={dot.key}
            cx={dot.x}
            cy={dot.y}
            r={dot.key === `${latestPoint.run_id}-${latestPoint.generated_at}-${points.length - 1}` ? '5' : '3.5'}
            fill={dot.status === 'failed' ? '#a44b4b' : '#4d8650'}
            stroke={dot.key === `${latestPoint.run_id}-${latestPoint.generated_at}-${points.length - 1}` ? '#1f3b8f' : 'none'}
            strokeWidth={dot.key === `${latestPoint.run_id}-${latestPoint.generated_at}-${points.length - 1}` ? '1.5' : '0'}
          />
        ))}
      </svg>
      <div className="history-modal-chart-range">
        <span className="history-modal-chart-meta">
          <strong>Oldest</strong> {oldestPoint.generated_at} · {oldestPoint.value}
        </span>
        <span className="history-modal-chart-meta">
          <strong>Range</strong> {min} - {max}
        </span>
        <span className="history-modal-chart-meta">
          <strong>Latest</strong> {latestPoint.generated_at} · {latestPoint.value}
        </span>
      </div>
    </div>
  );
};

const CheckHistoryModal = ({ detailTarget, onClose }) => {
  const [loading, setLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');
  const [detail, setDetail] = useState({ metrics: [], events: [] });

  useEffect(() => {
    if (!detailTarget) {
      return undefined;
    }

    let active = true;
    setLoading(true);
    setErrorMessage('');
    setDetail({ metrics: [], events: [] });

    const params = new URLSearchParams({
      host: detailTarget.host,
      check_name: detailTarget.checkName,
      limit: '20',
    });

    fetch(`/api/history/check-detail?${params.toString()}`)
      .then((response) => {
        if (!response.ok) {
          throw new Error('Could not load check history.');
        }
        return response.json();
      })
      .then((data) => {
        if (!active) {
          return;
        }
        setDetail({
          metrics: data.metrics || [],
          events: data.events || [],
        });
      })
      .catch((error) => {
        if (!active) {
          return;
        }
        setErrorMessage(error.message || 'Could not load check history.');
      })
      .finally(() => {
        if (active) {
          setLoading(false);
        }
      });

    return () => {
      active = false;
    };
  }, [detailTarget]);

  const latestMetric = useMemo(() => {
    if (!detail.metrics || detail.metrics.length === 0) {
      return null;
    }
    return detail.metrics[detail.metrics.length - 1];
  }, [detail.metrics]);

  if (!detailTarget) {
    return null;
  }

  const scopeLabel = detailTarget.scopeLabel || detailTarget.host;
  const measurementCopy = detailTarget.kind === 'url_check'
    ? 'Recent numeric values captured for this central URL check during full runs.'
    : 'Numeric values from the latest full runs for this host and check.';
  const emptyMeasurementCopy = detailTarget.kind === 'url_check'
    ? 'No stored numeric measurements for this URL check yet.'
    : 'No stored numeric measurements for this check yet.';
  const emptyEventCopy = detailTarget.kind === 'url_check'
    ? 'No state changes have been recorded for this URL check yet.'
    : 'No state changes have been recorded for this check yet.';

  return (
    <div className={`modal ${detailTarget ? 'is-active' : ''}`}>
      <div className="modal-background" onClick={onClose} />
      <div className="modal-card history-detail-modal">
        <header className="modal-card-head">
          <div>
            <p className="modal-card-title">{detailTarget.checkTitle}</p>
            <p className="history-modal-subtitle">{scopeLabel}</p>
          </div>
          <button className="delete" aria-label="close" onClick={onClose} />
        </header>
        <section className="modal-card-body">
          <div className="history-modal-summary">
            <span className={`history-status-pill is-${detailTarget.status || 'unknown'}`}>
              {detailTarget.status || 'unknown'}
            </span>
            <span className="history-pill">Current value: {detailTarget.value ?? 'n/a'}</span>
            <span className="history-pill">Failed when: result {formatFailWhen(detailTarget.failWhen)}</span>
            <span className="history-pill">Failed value(s): {formatFailValues(detailTarget.failValue)}</span>
            {latestMetric && (
              <span className="history-pill">Latest full run: {latestMetric.generated_at}</span>
            )}
          </div>

          {loading && <p>Loading history...</p>}
          {errorMessage && !loading && <div className="notification is-danger is-light">{errorMessage}</div>}

          {!loading && !errorMessage && (
            <>
              <div className="history-modal-section">
                <div className="history-section-header">
                  <div>
                    <h4>Recent measurements</h4>
                    <p className="history-section-copy">{measurementCopy}</p>
                  </div>
                </div>
                <StatusTrendChart points={detail.metrics} />
                {detail.metrics.length > 0 ? (
                  <div className="history-table-wrapper">
                    <table className="table history-data-table is-fullwidth">
                      <thead>
                        <tr>
                          <th>Time</th>
                          <th>Value</th>
                          <th>Failed when</th>
                          <th>Failed value(s)</th>
                          <th>Status</th>
                          <th>Run</th>
                        </tr>
                      </thead>
                      <tbody>
                        {[...detail.metrics].reverse().map((metric) => (
                          <tr key={`${metric.run_id}-${metric.generated_at}`}>
                            <td>{metric.generated_at}</td>
                            <td>{metric.value}</td>
                            <td><code>result {formatFailWhen(detailTarget.failWhen)}</code></td>
                            <td><code>{formatFailValues(detailTarget.failValue)}</code></td>
                            <td>
                              <span className={`history-status-pill is-${metric.status || 'unknown'}`}>
                                {metric.status || 'unknown'}
                              </span>
                            </td>
                            <td>#{metric.run_id}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                ) : (
                  <p className="history-modal-empty">{emptyMeasurementCopy}</p>
                )}
              </div>

              <div className="history-modal-section">
                <div className="history-section-header">
                  <div>
                    <h4>State changes</h4>
                    <p className="history-section-copy">Recent failures, recoveries, issue changes and other recorded events.</p>
                  </div>
                </div>
                {detail.events.length > 0 ? (
                  <div className="history-table-wrapper">
                    <table className="table history-data-table is-fullwidth">
                      <thead>
                        <tr>
                          <th>Time</th>
                          <th>Event</th>
                          <th>Status</th>
                          <th>Value</th>
                          <th>Failed when</th>
                          <th>Failed value(s)</th>
                          <th>Issue</th>
                          <th>Run</th>
                        </tr>
                      </thead>
                      <tbody>
                        {detail.events.map((event) => (
                          <tr key={event.id}>
                            <td>{event.event_time}</td>
                            <td>{event.event_type}</td>
                            <td>
                              <span className={`history-status-pill is-${event.status || 'unknown'}`}>
                                {event.status || 'unknown'}
                              </span>
                            </td>
                            <td>{event.value || '--'}</td>
                            <td><code>result {formatFailWhen(detailTarget.failWhen)}</code></td>
                            <td><code>{formatFailValues(detailTarget.failValue)}</code></td>
                            <td>
                              {event.error_type ? (
                                <>
                                  <span className="tag is-warning is-light">{formatErrorType(event.error_type)}</span>
                                  {event.error_message && <div className="error-detail-text">{event.error_message}</div>}
                                </>
                              ) : '--'}
                            </td>
                            <td>#{event.run_id}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                ) : (
                  <p className="history-modal-empty">{emptyEventCopy}</p>
                )}
              </div>
            </>
          )}
        </section>
      </div>
    </div>
  );
};

export default CheckHistoryModal;
