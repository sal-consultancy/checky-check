import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Chart } from 'chart.js/auto';

const formatErrorType = (errorType) => {
  if (!errorType) return '';
  return errorType.replaceAll('_', ' ');
};

const formatEventType = (eventType) => {
  if (!eventType) return '';
  return eventType.replaceAll('_', ' ');
};

const renderErrorSummary = (errorSummary) => {
  const entries = Object.entries(errorSummary || {}).filter(([, count]) => count > 0);
  if (entries.length === 0) {
    return 'No technical issues';
  }

  return entries
    .map(([type, count]) => `${formatErrorType(type)}: ${count}`)
    .join(', ');
};

const formatRunLabel = (generatedAt) => {
  if (!generatedAt) {
    return 'Unknown run';
  }

  return generatedAt;
};

const formatRunType = (runType) => {
  if (!runType) return 'full';
  return runType.replaceAll('_', ' ');
};

const formatRunTarget = (run) => {
  if (!run?.target_name) {
    return '-';
  }

  const parts = [run.target_name];
  if (run.target_scope) {
    parts.push(`(${run.target_scope})`);
  }
  return parts.join(' ');
};

const formatEventHost = (host) => {
  if (!host) return '-';
  if (host === 'url_checks') return 'URL Checks';
  return host;
};

const HistoryPage = () => {
  const [runs, setRuns] = useState([]);
  const [events, setEvents] = useState([]);
  const [loading, setLoading] = useState(true);
  const [eventsLoading, setEventsLoading] = useState(false);
  const [pageErrorMessage, setPageErrorMessage] = useState('');
  const [eventsErrorMessage, setEventsErrorMessage] = useState('');
  const [selectedRunId, setSelectedRunId] = useState(null);
  const trendChartRef = useRef(null);
  const trendChartInstanceRef = useRef(null);
  const latestEventsRequestRef = useRef(0);

  const fetchEvents = async (runId) => {
    const requestId = latestEventsRequestRef.current + 1;
    latestEventsRequestRef.current = requestId;
    setEventsLoading(true);
    setEventsErrorMessage('');

    const eventsUrl = runId
      ? `/api/history/events?limit=200&run_id=${encodeURIComponent(runId)}`
      : '/api/history/events?limit=30';

    try {
      const eventsResponse = await fetch(eventsUrl);
      if (!eventsResponse.ok) {
        throw new Error('Could not load history events.');
      }

      const eventsData = await eventsResponse.json();
      if (latestEventsRequestRef.current === requestId) {
        setEvents(Array.isArray(eventsData) ? eventsData : []);
      }
    } catch (error) {
      console.error('Error fetching history events:', error);
      if (latestEventsRequestRef.current === requestId) {
        setEvents([]);
        setEventsErrorMessage(error.message || 'Could not load history events.');
      }
    } finally {
      if (latestEventsRequestRef.current === requestId) {
        setEventsLoading(false);
      }
    }
  };

  useEffect(() => {
    const fetchHistory = async () => {
      setLoading(true);
      setPageErrorMessage('');

      try {
        const runsResponse = await fetch('/api/history/runs?limit=20');
        if (!runsResponse.ok) {
          throw new Error('Could not load history data.');
        }

        const runsData = await runsResponse.json();
        setRuns(Array.isArray(runsData) ? runsData : []);
      } catch (error) {
        console.error('Error fetching history:', error);
        setPageErrorMessage(error.message || 'Could not load history data.');
      }

      setLoading(false);
    };

    fetchHistory();
  }, []);

  useEffect(() => {
    if (loading) {
      return;
    }

    fetchEvents(selectedRunId);
  }, [selectedRunId, loading]);

  useEffect(() => {
    if (selectedRunId === null) {
      return;
    }

    const hasSelectedRun = runs.some((run) => run.id === selectedRunId);
    if (!hasSelectedRun) {
      setSelectedRunId(null);
    }
  }, [runs, selectedRunId]);

  const selectedRun = useMemo(
    () => runs.find((run) => run.id === selectedRunId) || null,
    [runs, selectedRunId]
  );

  const selectedRunLabel = selectedRun ? `#${selectedRun.id}` : 'Recent Events';

  const trendRuns = useMemo(() => {
    const recentRuns = runs
      .filter((run) => !run.run_type || run.run_type === 'full')
      .slice(0, 12)
      .reverse();
    return recentRuns;
  }, [runs]);

  useEffect(() => {
    if (!trendChartRef.current || trendRuns.length === 0) {
      if (trendChartInstanceRef.current) {
        trendChartInstanceRef.current.destroy();
        trendChartInstanceRef.current = null;
      }
      return;
    }

    if (trendChartInstanceRef.current) {
      trendChartInstanceRef.current.destroy();
    }

    trendChartInstanceRef.current = new Chart(trendChartRef.current, {
      type: 'line',
      data: {
        labels: trendRuns.map((run) => formatRunLabel(run.generated_at)),
        datasets: [
          {
            label: 'Failed checks',
            data: trendRuns.map((run) => run.failed_count),
            borderColor: '#c0392b',
            backgroundColor: 'rgba(192, 57, 43, 0.18)',
            fill: true,
            tension: 0.25,
          },
          {
            label: 'Passed checks',
            data: trendRuns.map((run) => run.passed_count),
            borderColor: '#1f8a4c',
            backgroundColor: 'rgba(31, 138, 76, 0.08)',
            fill: false,
            tension: 0.2,
          },
        ],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          title: {
            display: true,
            text: 'Recent Run Trend',
          },
          legend: {
            display: true,
          },
        },
        scales: {
          y: {
            beginAtZero: true,
            ticks: {
              precision: 0,
            },
            title: {
              display: true,
              text: 'Check count',
            },
          },
          x: {
            title: {
              display: true,
              text: 'Run time',
            },
          },
        },
      },
    });

    return () => {
      if (trendChartInstanceRef.current) {
        trendChartInstanceRef.current.destroy();
        trendChartInstanceRef.current = null;
      }
    };
  }, [trendRuns]);

  if (loading) {
    return <p>Loading history...</p>;
  }

  if (pageErrorMessage) {
    return (
      <div className="notification is-danger is-light">
        <h5 className="is-size-5 write py-2">History Error</h5>
        <p>{pageErrorMessage}</p>
      </div>
    );
  }

  return (
    <div className="history-page">
      <section className="mb-6">
        <h3 className="title is-4">Run Trend</h3>
        {trendRuns.length === 0 ? (
          <div className="notification is-warning is-light">No run history is available yet.</div>
        ) : (
          <div className="history-trend-card">
            <p className="history-section-copy">
              Recent full runs show the failed-check trend alongside passed checks so you can spot spikes quickly.
            </p>
            <div className="history-trend-canvas">
              <canvas ref={trendChartRef}></canvas>
            </div>
          </div>
        )}
      </section>

      <section className="mb-6">
        <div className="history-section-header">
          <div>
            <h3 className="title is-4">Recent Runs</h3>
            <p className="history-section-copy">
              Select a run to inspect only the events that were recorded during that execution.
            </p>
          </div>
          {selectedRun ? (
            <button className="button is-small is-light" type="button" onClick={() => setSelectedRunId(null)}>
              Show All Recent Events
            </button>
          ) : null}
        </div>
        {runs.length === 0 ? (
          <div className="notification is-warning is-light">No run history is available yet.</div>
        ) : (
          <div className="history-table-wrapper">
            <table className="table is-striped is-bordered is-size-7 is-fullwidth">
              <thead>
                <tr>
                  <th>Run</th>
                  <th>Time</th>
                  <th>Type</th>
                  <th>Target</th>
                  <th>Status</th>
                  <th>Hosts</th>
                  <th>Checks</th>
                  <th>Passed</th>
                  <th>Failed</th>
                  <th>Duration</th>
                  <th>Issues</th>
                  <th>View</th>
                </tr>
              </thead>
              <tbody>
                {runs.map((run) => {
                  const isSelected = run.id === selectedRunId;
                  return (
                    <tr key={run.id} className={isSelected ? 'history-run-row is-selected' : 'history-run-row'}>
                      <td>#{run.id}</td>
                      <td>{run.generated_at}</td>
                      <td>{formatRunType(run.run_type)}</td>
                      <td>{formatRunTarget(run)}</td>
                      <td>{run.status}</td>
                      <td>{run.host_count}</td>
                      <td>{run.check_count}</td>
                      <td>{run.passed_count}</td>
                      <td>{run.failed_count}</td>
                      <td>{run.duration_ms} ms</td>
                      <td>{renderErrorSummary(run.error_summary)}</td>
                      <td>
                        <button
                          className={isSelected ? 'button is-small is-link' : 'button is-small is-light'}
                          type="button"
                          onClick={() => setSelectedRunId(isSelected ? null : run.id)}
                        >
                          {isSelected ? 'Viewing' : 'View Run'}
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section>
        <div className="history-section-header">
          <div>
            <h3 className="title is-4">
              {selectedRun ? `Events For Run ${selectedRunLabel}` : selectedRunLabel}
            </h3>
            <p className="history-section-copy">
              {selectedRun
                ? `Showing events captured during the run at ${selectedRun.generated_at}.`
                : 'Showing the latest failures, recoveries, and config issues across recent runs.'}
            </p>
          </div>
        </div>
        {eventsErrorMessage ? (
          <div className="notification is-danger is-light">
            <p>{eventsErrorMessage}</p>
          </div>
        ) : null}
        {eventsLoading ? (
          <p>Loading events...</p>
        ) : events.length === 0 ? (
          <div className="notification is-warning is-light">
            {selectedRun
              ? 'No history events were recorded for this run.'
              : 'No recent history events were recorded.'}
          </div>
        ) : (
          <div className="history-table-wrapper">
            <table className="table is-striped is-bordered is-size-7 is-fullwidth">
              <thead>
                <tr>
                  <th>Run</th>
                  <th>Time</th>
                  <th>Event</th>
                  <th>Host</th>
                  <th>Check</th>
                  <th>Status</th>
                  <th>Value</th>
                  <th>Issue</th>
                </tr>
              </thead>
              <tbody>
                {events.map((event) => (
                  <tr key={event.id}>
                    <td>#{event.run_id}</td>
                    <td>{event.event_time}</td>
                    <td>{formatEventType(event.event_type)}</td>
                    <td>{formatEventHost(event.host)}</td>
                    <td>{event.check_name || '-'}</td>
                    <td>{event.status || '-'}</td>
                    <td>{event.value || '-'}</td>
                    <td>
                      {event.error_type ? (
                        <>
                          <span className="tag is-warning is-light">{formatErrorType(event.error_type)}</span>
                          {event.error_message && <div className="error-detail-text">{event.error_message}</div>}
                        </>
                      ) : '-'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
};

export default HistoryPage;
