import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { Chart } from 'chart.js/auto';

const resolveTemplateValue = (template, vars) => {
  if (!template) return '';
  return template.replace(/\$\{([a-zA-Z_][a-zA-Z0-9_.-]*)\}/g, (match, key) => (
    Object.prototype.hasOwnProperty.call(vars || {}, key) ? vars[key] : match
  ));
};

const SummaryPage = ({ results, checks, urlResults, urlChecks, status, stats, errorSummary }) => {
  const chartRef = useRef(null);
  const chartInstanceRef = useRef(null);
  const [showPassingURLExamples, setShowPassingURLExamples] = useState(false);
  const hasChecks = Object.keys(checks).length > 0;
  const hasURLChecks = Object.keys(urlChecks || {}).length > 0;
  const hasAnyChecks = hasChecks || hasURLChecks;

  const summary = useMemo(() => {
    return Object.keys(checks).reduce((acc, checkName) => {
      acc[checkName] = { passed: 0, failed: 0, details: [] };

      Object.keys(results).forEach((host) => {
        if (!results[host]?.[checkName]) {
          return;
        }

        const result = results[host][checkName];
        if (result.status === 'passed') {
          acc[checkName].passed += 1;
        } else {
          acc[checkName].failed += 1;
        }
        acc[checkName].details.push({ host, ...result });
      });

      return acc;
    }, {});
  }, [checks, results]);

  const checksWithFailures = useMemo(() => {
    return Object.keys(summary)
      .filter((checkName) => summary[checkName].failed > 0)
      .sort((left, right) => {
        const failedDelta = summary[right].failed - summary[left].failed;
        if (failedDelta !== 0) {
          return failedDelta;
        }
        return summary[right].passed - summary[left].passed;
      });
  }, [summary]);

  const chartCheckNames = useMemo(() => {
    if (checksWithFailures.length > 0) {
      return checksWithFailures;
    }

    return Object.keys(summary).sort((left, right) => summary[right].passed - summary[left].passed);
  }, [checksWithFailures, summary]);

  const failedHostCount = useMemo(() => {
    return Object.values(results).filter((hostResults) =>
      Object.values(hostResults || {}).some((result) => result.status === 'failed')
    ).length;
  }, [results]);

  const urlCheckCount = Object.keys(urlChecks || {}).length;
  const failedURLChecks = Object.values(urlResults || {}).filter((result) => result.status === 'failed').length;
  const totalExecutedChecks = stats.executedChecks + Object.keys(urlResults || {}).length;
  const totalDistinctChecks = stats.checkCount + urlCheckCount;
  const totalFailedChecks = stats.failedChecks + failedURLChecks;
  const totalFailPercentage = totalExecutedChecks === 0
    ? 0
    : (totalFailedChecks / totalExecutedChecks) * 100;

  const statusTitle = (() => {
    if (totalFailedChecks === 0) {
      return 'All checks are currently passing';
    }

    const parts = [];
    if (stats.failedChecks > 0) {
      parts.push(`${stats.failedChecks} failed host results across ${failedHostCount} hosts`);
    }
    if (failedURLChecks > 0) {
      parts.push(`${failedURLChecks} failing URL checks`);
    }
    return parts.join(' and ');
  })();

  const statusCopy = totalFailedChecks > 0
    ? 'Focus on the failing host checks and URL checks below to see what needs attention first.'
    : 'The latest run completed without operational issues.';
  const failingURLCheckNames = Object.keys(urlChecks || {}).filter((checkName) => urlResults?.[checkName]?.status === 'failed');
  const urlPreviewNames = failingURLCheckNames.length > 0
    ? failingURLCheckNames
    : Object.keys(urlChecks || {}).slice(0, 3);

  useEffect(() => {
    if (status === 'config_error' || !hasChecks || !chartRef.current) {
      return undefined;
    }

    if (chartInstanceRef.current) {
      chartInstanceRef.current.destroy();
    }

    chartInstanceRef.current = new Chart(chartRef.current, {
      type: 'bar',
      data: {
        labels: chartCheckNames.map((checkName) => checks[checkName]?.title || checkName),
        datasets: [
          {
            label: 'Failed',
            data: chartCheckNames.map((checkName) => summary[checkName].failed),
            backgroundColor: '#a44b4b',
            borderColor: '#8f3c3c',
            borderWidth: 1,
          },
          {
            label: 'Passed',
            data: chartCheckNames.map((checkName) => summary[checkName].passed),
            backgroundColor: '#6aa36d',
            borderColor: '#4d8650',
            borderWidth: 1,
          },
        ],
      },
      options: {
        indexAxis: 'y',
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: {
            position: 'bottom',
          },
          tooltip: {
            callbacks: {
              label(context) {
                return `${context.dataset.label}: ${context.raw}`;
              },
            },
          },
        },
        scales: {
          x: {
            beginAtZero: true,
            stacked: true,
            ticks: {
              precision: 0,
            },
            grid: {
              color: 'rgba(25, 35, 55, 0.08)',
            },
          },
          y: {
            stacked: true,
            grid: {
              display: false,
            },
          },
        },
      },
    });

    return () => {
      if (chartInstanceRef.current) {
        chartInstanceRef.current.destroy();
        chartInstanceRef.current = null;
      }
    };
  }, [chartCheckNames, checks, hasChecks, status, summary]);

  if (status === 'config_error') {
    return (
      <div className="notification is-danger is-light">
        <h5 className="is-size-5 write py-2">Configuration Error</h5>
        <p>The summary is unavailable because the latest run failed validation.</p>
      </div>
    );
  }

  if (!hasAnyChecks) {
    return (
      <div className="notification is-warning is-light">
        No check results are available yet.
      </div>
    );
  }

  return (
    <div className="summary-dashboard">
      <div className={`summary-status-panel ${totalFailedChecks > 0 ? 'has-issues' : 'is-healthy'}`}>
        <div>
          <div className="summary-status-title">{statusTitle}</div>
          <p className="summary-status-copy">{statusCopy}</p>
        </div>
        <div className="summary-status-actions">
          <Link className="button is-small is-dark" to="/run-tests">Run Checks</Link>
          <Link className="button is-small is-light" to="/report">Open Report</Link>
          <Link className="button is-small is-light" to="/hosts">Open Hosts</Link>
          <Link className="button is-small is-light" to="/history">Run History</Link>
        </div>
      </div>

      <div className="summary-kpi-grid">
        <div className="summary-kpi-card">
          <div className="summary-metric-label">Executed Checks</div>
          <div className="summary-kpi-value">{totalExecutedChecks}</div>
        </div>
        <div className="summary-kpi-card">
          <div className="summary-metric-label">Distinct Check Types</div>
          <div className="summary-kpi-value">{totalDistinctChecks}</div>
        </div>
        <div className="summary-kpi-card">
          <div className="summary-metric-label">Fail Percentage</div>
          <div className="summary-kpi-value">{totalFailPercentage.toFixed(1)}%</div>
        </div>
        <div className="summary-kpi-card">
          <div className="summary-metric-label">Hosts Checked</div>
          <div className="summary-kpi-value">{stats.hostCount}</div>
        </div>
      </div>

      {errorSummary && errorSummary.length > 0 && (
        <div className="summary-breakdown-box">
          <div className="summary-metric-label">Failure Breakdown</div>
          <div className="summary-breakdown-tags">
            {errorSummary.map((item) => (
              <span key={item.type} className="tag is-warning is-light summary-breakdown-tag">
                {item.label}: {item.count}
              </span>
            ))}
          </div>
        </div>
      )}

      {hasURLChecks && (
        <div className="summary-url-card">
          <div className="summary-chart-header">
            <div>
              <div className="summary-metric-label">URL Checks</div>
              <h3 className="summary-section-title">Central website checks</h3>
              <p className="summary-section-copy">
                {failedURLChecks > 0
                  ? `${failedURLChecks} failing, ${urlCheckCount - failedURLChecks} passing`
                  : `All ${urlCheckCount} URL checks are currently passing.`}
              </p>
            </div>
            <span className="tag is-light">
              {failedURLChecks > 0 ? `${failedURLChecks} failing` : 'All passing'}
            </span>
          </div>
          {failedURLChecks > 0 ? (
            <div className="summary-url-list">
              {urlPreviewNames.map((checkName) => {
                const check = urlChecks[checkName];
                const result = urlResults?.[checkName];
                const isFailed = result?.status === 'failed';
                const resolvedURL = resolveTemplateValue(check.url, result?.vars);
                const summaryBits = [
                  result?.value || 'n/a',
                  Number.isFinite(result?.latency_ms) ? `${result.latency_ms} ms` : null,
                ].filter(Boolean);

                return (
                  <Link key={checkName} className={`summary-url-row${isFailed ? ' is-failed' : ''}`} to={`/report#url-${checkName}`}>
                    <div>
                      <strong>{check.title}</strong>
                      <div className="summary-url-meta">
                        <span>{resolvedURL}</span>
                        <span>{summaryBits.join(' · ')}</span>
                        {result?.final_url && result.final_url !== resolvedURL && (
                          <span>Final: {result.final_url}</span>
                        )}
                      </div>
                    </div>
                    <span className={`tag ${isFailed ? 'is-danger' : 'is-success'} is-light`}>
                      {result?.status || 'unknown'}
                    </span>
                  </Link>
                );
              })}
            </div>
          ) : (
            <>
              <div className="summary-success-card summary-success-card-compact">
                <p>All {urlCheckCount} URL checks are healthy.</p>
                <button
                  type="button"
                  className="button is-small is-light"
                  onClick={() => setShowPassingURLExamples((current) => !current)}
                >
                  {showPassingURLExamples ? 'Hide examples' : 'Show examples'}
                </button>
              </div>
              {showPassingURLExamples && (
                <div className="summary-url-list mt-3">
                  {urlPreviewNames.map((checkName) => {
                    const check = urlChecks[checkName];
                    const result = urlResults?.[checkName];
                    const resolvedURL = resolveTemplateValue(check.url, result?.vars);
                    const summaryBits = [
                      result?.value || 'n/a',
                      Number.isFinite(result?.latency_ms) ? `${result.latency_ms} ms` : null,
                    ].filter(Boolean);

                    return (
                      <Link key={checkName} className="summary-url-row" to={`/report#url-${checkName}`}>
                        <div>
                          <strong>{check.title}</strong>
                          <div className="summary-url-meta">
                            <span>{resolvedURL}</span>
                            <span>{summaryBits.join(' · ')}</span>
                            {result?.final_url && result.final_url !== resolvedURL && (
                              <span>Final: {result.final_url}</span>
                            )}
                          </div>
                        </div>
                        <span className="tag is-success is-light">
                          {result?.status || 'unknown'}
                        </span>
                      </Link>
                    );
                  })}
                </div>
              )}
            </>
          )}
          {(failedURLChecks === 0 || urlCheckCount > urlPreviewNames.length) && (
            <div className="summary-url-footer">
              <Link className="button is-small is-light" to="/report#url-checks">View all URL checks</Link>
            </div>
          )}
        </div>
      )}

      {checksWithFailures.length > 0 ? (
        <section className="summary-issues-card">
          <div className="summary-metric-label">Current Issues</div>
          <h3 className="summary-section-title">Most affected checks right now</h3>
          <div className="summary-issue-list">
            {checksWithFailures.slice(0, 6).map((checkName) => (
              <Link key={checkName} className="summary-issue-row" to={`/report#${checkName}`}>
                <div>
                  <strong>{checks[checkName]?.title || checkName}</strong>
                  <div className="summary-issue-meta">
                    {summary[checkName].failed} failed, {summary[checkName].passed} passed
                  </div>
                </div>
                <span className="tag is-danger is-light">{summary[checkName].failed}</span>
              </Link>
            ))}
          </div>
        </section>
      ) : (
        <section className="summary-chart-card">
          <div className="summary-chart-header">
            <div>
              <div className="summary-metric-label">Results Per Check</div>
              <h3 className="summary-section-title">Per-check outcome</h3>
            </div>
            <span className="tag is-light">No failing checks</span>
          </div>
          {hasChecks ? (
            <div className="summary-chart-canvas">
              <canvas ref={chartRef}></canvas>
            </div>
          ) : (
            <div className="summary-success-card">
              <p>No host-based checks ran in this report. Only central URL checks are available.</p>
            </div>
          )}
        </section>
      )}
    </div>
  );
};

export default SummaryPage;
