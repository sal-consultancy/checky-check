import React, { useEffect, useState } from 'react';
import ChartComponent from './ChartComponent';
import { FaChevronDown, FaChevronUp, FaPlus, FaMinus, FaTimes } from 'react-icons/fa';
import CheckHistoryModal from './CheckHistoryModal';
import { formatFailValues, formatFailWhen } from '../utils/checkFormatting';

const formatErrorType = (errorType) => {
  if (!errorType) return '';
  return errorType.replaceAll('_', ' ');
};

const resolveTemplateValue = (template, vars) => {
  if (!template) return '';
  return template.replace(/\$\{([a-zA-Z_][a-zA-Z0-9_.-]*)\}/g, (match, key) => (
    Object.prototype.hasOwnProperty.call(vars || {}, key) ? vars[key] : match
  ));
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

const CheckReport = ({ results, checks, urlResults, urlChecks, theme, status, authSession }) => {
  const [expandedSections, setExpandedSections] = useState({});
  const [showDetails, setShowDetails] = useState({});
  const [showURLIntro, setShowURLIntro] = useState(false);
  const [showAllFailedHosts, setShowAllFailedHosts] = useState(false);
  const [showOnlyFailedTests, setShowOnlyFailedTests] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [rerunLoading, setRerunLoading] = useState({});
  const [rerunFeedback, setRerunFeedback] = useState({});
  const [resultOverrides, setResultOverrides] = useState({ results: {}, urlResults: {} });
  const [sparklineData, setSparklineData] = useState({});
  const [detailTarget, setDetailTarget] = useState(null);

  useEffect(() => {
    fetch('/api/history/sparklines?limit=14')
      .then((response) => response.json())
      .then((data) => setSparklineData(data || {}))
      .catch(() => setSparklineData({}));
  }, []);

  const canRerunChecks = authSession?.permissions?.operate ?? true;

  if (status === 'config_error') {
    return (
      <div className="notification is-danger is-light">
        <h5 className="is-size-5 write py-2">Configuration Error</h5>
        <p>The report is unavailable because the latest run failed validation.</p>
      </div>
    );
  }

  if (Object.keys(checks).length === 0 && Object.keys(urlChecks || {}).length === 0) {
    return (
      <div className="notification is-warning is-light">
        No check results are available yet.
      </div>
    );
  }

  const toggleSection = (section) => {
    setExpandedSections((prevState) => ({
      ...prevState,
      [section]: !prevState[section],
    }));
  };

  const toggleDetails = (checkName) => {
    setShowDetails((prevState) => ({
      ...prevState,
      [checkName]: !prevState[checkName],
    }));
  };

  const clearSearch = () => {
    setSearchTerm('');
  };

  const mergedResults = Object.keys(results || {}).reduce((acc, host) => {
    acc[host] = {
      ...(results[host] || {}),
      ...(resultOverrides.results?.[host] || {}),
    };
    return acc;
  }, {});

  Object.keys(resultOverrides.results || {}).forEach((host) => {
    if (!mergedResults[host]) {
      mergedResults[host] = { ...(resultOverrides.results[host] || {}) };
    }
  });

  const mergedURLResults = {
    ...(urlResults || {}),
    ...(resultOverrides.urlResults || {}),
  };

  const summary = Object.keys(checks).reduce((acc, checkName) => {
    acc[checkName] = { passed: 0, failed: 0, details: [] };

    Object.keys(mergedResults).forEach((host) => {
      if (!mergedResults[host]?.[checkName]) {
        return;
      }

      const result = mergedResults[host][checkName];
      if (result.status === 'passed') {
        acc[checkName].passed += 1;
      } else {
        acc[checkName].failed += 1;
      }
      acc[checkName].details.push({ host, ...result });
    });

    return acc;
  }, {});

  const filteredChecks = Object.keys(summary).filter((checkName) =>
    checks[checkName].title.toLowerCase().includes(searchTerm.toLowerCase()) ||
    checks[checkName].description.toLowerCase().includes(searchTerm.toLowerCase())
  );
  const orderedChecks = [...filteredChecks].sort((left, right) => {
    const failedDelta = summary[right].failed - summary[left].failed;
    if (failedDelta !== 0) return failedDelta;
    const passedDelta = summary[left].passed - summary[right].passed;
    if (passedDelta !== 0) return passedDelta;
    return checks[left].title.localeCompare(checks[right].title);
  });

  const filteredURLChecks = Object.keys(urlChecks || {}).filter((checkName) => {
    const check = urlChecks[checkName];
    return (
      check.title.toLowerCase().includes(searchTerm.toLowerCase()) ||
      check.description.toLowerCase().includes(searchTerm.toLowerCase())
    );
  });

  const visibleURLChecks = filteredURLChecks.filter((checkName) =>
    !showOnlyFailedTests || mergedURLResults?.[checkName]?.status === 'failed'
  );

  const visibleURLDetails = visibleURLChecks.map((checkName) => {
    const check = urlChecks[checkName];
    const result = mergedURLResults?.[checkName];

    return {
      checkName,
      title: check.title,
      description: check.description,
      url: resolveTemplateValue(check.url, result?.vars),
      fail_when: check.fail_when,
      fail_value: check.fail_value,
      status: result?.status || 'unknown',
      value: result?.value || 'n/a',
      statusCode: result?.status_code,
      latencyMs: result?.latency_ms,
      redirected: Boolean(result?.redirected),
      location: result?.location || '',
      finalUrl: result?.final_url || '',
      timestamp: result?.timestamp || 'n/a',
      error_type: result?.error_type || '',
      error_message: result?.error_message || '',
    };
  });

  const passedURLDetails = visibleURLDetails.filter((detail) => detail.status === 'passed');
  const failedURLDetails = visibleURLDetails.filter((detail) => detail.status === 'failed');
  const hasPassedURLs = passedURLDetails.length > 0;
  const hasFailedURLs = failedURLDetails.length > 0;
  const urlGraphData = [
    { value: passedURLDetails.length, failed: 0, status: 'passed' },
    { value: failedURLDetails.length, failed: failedURLDetails.length, status: 'failed' },
  ];

  const toggleAllFailedHosts = () => {
    setShowAllFailedHosts((prevState) => !prevState);
    if (!showAllFailedHosts) {
      const newExpandedSections = {};
      Object.keys(summary).forEach((checkName) => {
        if (summary[checkName].failed > 0) {
          newExpandedSections[`${checkName}-failed`] = true;
        }
      });
      if (hasFailedURLs) {
        newExpandedSections['url-checks-failed'] = true;
      }
      setExpandedSections(newExpandedSections);
    } else {
      setExpandedSections({});
    }
  };

  const rerunTarget = async (payload, loadingKey, feedbackKey, successLabel) => {
    setRerunLoading((prev) => ({ ...prev, [loadingKey]: true }));
    setRerunFeedback((prev) => ({ ...prev, [feedbackKey]: '' }));

    try {
      const response = await fetch('/api/run-check', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        const message = await response.text();
        throw new Error(message || 'Could not rerun the check.');
      }

      const data = await response.json();
      setResultOverrides((prev) => {
        const nextHostResults = { ...(prev.results || {}) };
        (data.host_results || []).forEach((result) => {
          nextHostResults[result.host] = {
            ...(nextHostResults[result.host] || {}),
            [result.check]: result,
          };
        });

        const nextURLResults = { ...(prev.urlResults || {}) };
        (data.url_results || []).forEach((result) => {
          nextURLResults[result.check] = result;
        });

        return {
          results: nextHostResults,
          urlResults: nextURLResults,
        };
      });

      setRerunFeedback((prev) => ({
        ...prev,
        [feedbackKey]: `${successLabel} rerun completed in history run #${data.run.id}.`,
      }));
    } catch (error) {
      setRerunFeedback((prev) => ({
        ...prev,
        [feedbackKey]: error.message || 'Could not rerun the check.',
      }));
    } finally {
      setRerunLoading((prev) => ({ ...prev, [loadingKey]: false }));
    }
  };

  const renderURLTable = (details, heading, isFailedTable) => (
    <>
      <h5 className="is-size-5 write mt-3 has-text-left">{heading}</h5>
      <table className="table is-striped is-bordered is-size-7 mt-2 is-fullwidth">
        <thead>
          <tr>
            <th>Name</th>
            <th>URL</th>
            <th>Status</th>
            <th>Value</th>
            <th>Failed when</th>
            <th>Failed value(s)</th>
            <th>Latency</th>
            <th>Issue</th>
            <th>Timestamp</th>
            <th className="no-print">History</th>
            {canRerunChecks && <th className="no-print">Action</th>}
          </tr>
        </thead>
        <tbody>
          {details.map((detail) => (
            <tr key={detail.checkName} id={`url-${detail.checkName}`} className={isFailedTable ? 'url-check-row is-failed' : 'url-check-row'}>
              <td>
                <strong>{detail.title}</strong>
                {detail.description && <div>{detail.description}</div>}
              </td>
              <td>
                <code>{detail.url}</code>
                {detail.finalUrl && detail.finalUrl !== detail.url && (
                  <div className="error-detail-text">Final: {detail.finalUrl}</div>
                )}
                {detail.location && (
                  <div className="error-detail-text">Location: {detail.location}</div>
                )}
              </td>
              <td>{detail.status}</td>
              <td>
                {detail.value}
                {Number.isFinite(detail.statusCode) && detail.statusCode > 0 ? <div className="error-detail-text">HTTP {detail.statusCode}</div> : null}
              </td>
              <td><code>result {formatFailWhen(detail.fail_when)}</code></td>
              <td><code>{formatFailValues(detail.fail_value)}</code></td>
              <td>{Number.isFinite(detail.latencyMs) ? `${detail.latencyMs} ms` : '--'}</td>
              <td>
                {detail.error_type ? (
                  <>
                    <span className="tag is-warning is-light">{formatErrorType(detail.error_type)}</span>
                    {detail.error_message && <div className="error-detail-text">{detail.error_message}</div>}
                  </>
                ) : detail.redirected ? (
                  <span className="tag is-info is-light">redirected</span>
                ) : '--'}
              </td>
              <td>{detail.timestamp}</td>
              <td className="no-print">
                <button
                  className="button is-small is-light"
                  onClick={() => setDetailTarget({
                    host: 'url_checks',
                    scopeLabel: 'Central URL check',
                    kind: 'url_check',
                    checkName: detail.checkName,
                    checkTitle: detail.title,
                    status: detail.status,
                    value: detail.value,
                    failWhen: detail.fail_when,
                    failValue: detail.fail_value,
                  })}
                >
                  View history
                </button>
              </td>
              {canRerunChecks && (
                <td className="no-print">
                  <button
                    className={`button is-small is-light ${rerunLoading[`url:${detail.checkName}`] ? 'is-loading' : ''}`}
                    onClick={() => rerunTarget(
                      { kind: 'url_check', check_name: detail.checkName },
                      `url:${detail.checkName}`,
                      'url-checks',
                      detail.title
                    )}
                    disabled={rerunLoading[`url:${detail.checkName}`]}
                  >
                    Re-run
                  </button>
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </>
  );

  const renderHostTable = (checkName, checkTitle, details, heading, isFailedTable) => {
    const check = checks[checkName] || {};
    const sparklineEnabled = Boolean(check.sparkline?.enabled);

    return (
    <>
      <h5 className="is-size-5 write mt-3 has-text-left">{heading}</h5>
      <table className="table is-striped is-bordered is-size-7 mt-2">
        <thead>
          <tr>
            <th>Host</th>
            <th>Status</th>
            <th>Value</th>
            <th>Failed when</th>
            <th>Failed value(s)</th>
            {sparklineEnabled && <th>Trend</th>}
            <th>Issue</th>
            <th>Timestamp</th>
            <th className="no-print">History</th>
            {canRerunChecks && <th className="no-print">Action</th>}
          </tr>
        </thead>
        <tbody>
          {details.map((detail) => (
            <tr key={detail.host}>
              <td>{detail.host}</td>
              <td>{detail.status}</td>
              <td>{detail.value}</td>
              <td><code>result {formatFailWhen(check.fail_when)}</code></td>
              <td><code>{formatFailValues(check.fail_value)}</code></td>
              {sparklineEnabled && (
                <td>
                  <Sparkline points={sparklineData?.[detail.host]?.[checkName] || []} />
                </td>
              )}
              <td>
                {isFailedTable && detail.error_type ? (
                  <>
                    <span className="tag is-warning is-light">{formatErrorType(detail.error_type)}</span>
                    {detail.error_message && <div className="error-detail-text">{detail.error_message}</div>}
                  </>
                ) : '--'}
              </td>
              <td>{detail.timestamp}</td>
              <td className="no-print">
                <button
                  className="button is-small is-light"
                  onClick={() => setDetailTarget({
                    host: detail.host,
                    checkName,
                    checkTitle,
                    status: detail.status,
                    value: detail.value,
                    failWhen: check.fail_when,
                    failValue: check.fail_value,
                  })}
                >
                  View history
                </button>
              </td>
              {canRerunChecks && (
                <td className="no-print">
                  <button
                    className={`button is-small is-light ${rerunLoading[`check:${checkName}:host:${detail.host}`] ? 'is-loading' : ''}`}
                    onClick={() => rerunTarget(
                      { kind: 'host_check', check_name: checkName, host: detail.host },
                      `check:${checkName}:host:${detail.host}`,
                      `check:${checkName}`,
                      `${checkTitle} on ${detail.host}`
                    )}
                    disabled={rerunLoading[`check:${checkName}:host:${detail.host}`]}
                  >
                    Re-run host
                  </button>
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </>
    );
  };

  return (
    <div className="check-report">
      <div className="no-print">
        <h6 className="is-size-6 write my-3">Report Filter</h6>
        <div className="buttons-container mb-5">
          <button onClick={toggleAllFailedHosts} className="button is-grey is-light is-small">
            {showAllFailedHosts ? 'Collapse All Failed Hosts' : 'Expand All Failed Hosts'}
            {showAllFailedHosts ? <FaMinus className="ml-2" /> : <FaPlus className="ml-2" />}
          </button>
          <button onClick={() => setShowOnlyFailedTests((prevState) => !prevState)} className="button is-grey is-light is-small ml-2">
            {showOnlyFailedTests ? 'Show All Tests' : 'Show Only Failed Tests'}
            {showOnlyFailedTests ? <FaMinus className="ml-2" /> : <FaPlus className="ml-2" />}
          </button>
        </div>
        <div className="field has-addons">
          <div className="control is-expanded">
            <input
              type="text"
              className="input is-small"
              placeholder="Search checks..."
              value={searchTerm}
              onChange={(event) => setSearchTerm(event.target.value)}
            />
          </div>
          <div className="control">
            <button className="button is-small" style={{ height: '100%' }} onClick={clearSearch}>
              <FaTimes />
            </button>
          </div>
        </div>
        <hr className="separator" />
      </div>

      {visibleURLChecks.length > 0 && (
        <>
          <div className="url-check-section">
            <div className="url-check-header">
              <div>
                <div className="url-check-title-row">
                  <h4 className="is-size-4 write" id="url-checks">URL Checks</h4>
                  <button
                    type="button"
                    className="button is-small is-light no-print"
                    onClick={() => setShowURLIntro((current) => !current)}
                  >
                    {showURLIntro ? <FaChevronUp /> : <FaChevronDown />}
                  </button>
                </div>
                {showURLIntro && (
                  <p className="summary-section-copy">
                    Central website checks with quick rerun actions and passed versus failed breakdown.
                  </p>
                )}
              </div>
              <span className="tag is-light">{visibleURLChecks.length} shown</span>
            </div>
            {rerunFeedback['url-checks'] && (
              <div className="notification is-light mt-3">{rerunFeedback['url-checks']}</div>
            )}
            <div className="report-url-layout">
              <div className="report-url-chart">
                <ChartComponent
                  data={urlGraphData}
                  labels={['Passed URLs', 'Failed URLs']}
                  title="URL status overview"
                  theme={theme}
                  type="pie"
                />
              </div>
              <div className="report-url-side">
                <div className="report-check-badges">
                  <span className="tag is-success is-light">{passedURLDetails.length} passed</span>
                  <span className="tag is-danger is-light">{failedURLDetails.length} failed</span>
                </div>
                {!canRerunChecks && (
                  <p className="report-check-copy mb-3">Read-only access. URL reruns are limited to operators.</p>
                )}
                <div className="buttons-container mb-0 no-print report-inline-buttons">
                  {hasPassedURLs && (
                    <button onClick={() => toggleSection('url-checks-passed')} className="button is-grey is-light is-small">
                      {expandedSections['url-checks-passed'] ? 'Hide Passed URLs' : 'Show Passed URLs'}
                      <span className="tag is-success is-light ml-2">{passedURLDetails.length}</span>
                      {expandedSections['url-checks-passed'] ? <FaChevronUp className="ml-2" /> : <FaChevronDown className="ml-2" />}
                    </button>
                  )}
                  {hasFailedURLs && (
                    <button onClick={() => toggleSection('url-checks-failed')} className="button is-grey is-light is-small ml-2">
                      {expandedSections['url-checks-failed'] ? 'Hide Failed URLs' : 'Show Failed URLs'}
                      <span className="tag is-danger is-light ml-2">{failedURLDetails.length}</span>
                      {expandedSections['url-checks-failed'] ? <FaChevronUp className="ml-2" /> : <FaChevronDown className="ml-2" />}
                    </button>
                  )}
                </div>
              </div>
            </div>
            {expandedSections['url-checks-passed'] && hasPassedURLs && renderURLTable(passedURLDetails, 'Passed URLs', false)}
            {expandedSections['url-checks-failed'] && hasFailedURLs && renderURLTable(failedURLDetails, 'Failed URLs', true)}
          </div>
          {orderedChecks.length > 0 && <hr className="separator" />}
        </>
      )}

      <div className="report-check-list">
      {orderedChecks.map((checkName) => {
        const check = checks[checkName];

        if (showOnlyFailedTests && summary[checkName].failed === 0) {
          return null;
        }

        let graphData;
        if (check.graph.type === 'bar_grouped_by_10_percentile') {
          graphData = new Array(11).fill(0).map(() => ({
            value: 0,
            failed: 0,
          }));

          summary[checkName].details.forEach((detail) => {
            const percentile = Math.min(Math.floor(detail.value / 10), 10);
            graphData[percentile].value += 1;
            if (detail.status === 'failed') {
              graphData[percentile].failed += 1;
            }
          });
        } else {
          graphData = summary[checkName].details.reduce((acc, detail) => {
            acc[detail.value] = acc[detail.value] || { value: 0, failed: 0 };
            acc[detail.value].value += 1;
            if (detail.status === 'failed') {
              acc[detail.value].failed += 1;
            }
            return acc;
          }, {});
        }

        const labels = check.graph.type === 'bar_grouped_by_10_percentile'
          ? ['0-9%', '10-19%', '20-29%', '30-39%', '40-49%', '50-59%', '60-69%', '70-79%', '80-89%', '90-99%', '100%']
          : Object.keys(graphData);

        const data = labels.map((label, bucketIndex) => (
          check.graph.type === 'bar_grouped_by_10_percentile'
            ? graphData[bucketIndex]
            : graphData[label]
        ));

        const hasPassedDetails = summary[checkName].details.some((detail) => detail.status === 'passed');
        const hasFailedDetails = summary[checkName].details.some((detail) => detail.status === 'failed');
        const passedDetails = summary[checkName].details.filter((detail) => detail.status === 'passed');
        const failedDetails = summary[checkName].details.filter((detail) => detail.status === 'failed');

        return (
          <div key={checkName} className={`report-check-card${hasFailedDetails ? ' has-failures' : ''}`}>
            <div className="report-check-top">
              <div className="report-check-chart">
                <h4 className="is-size-4 write report-check-title" id={checkName}>{check.title}</h4>
                <ChartComponent
                  data={data}
                  labels={labels}
                  title={check.graph.title}
                  theme={theme}
                  type={check.graph?.type === 'pie_grouped_by_value' ? 'pie' : 'bar'}
                  colors={check.graph?.colors}
                />
              </div>
              <div className="report-check-side">
                <div className="report-check-actions no-print">
                  {canRerunChecks && (
                    <button
                      className={`button is-small is-light ${rerunLoading[`check:${checkName}`] ? 'is-loading' : ''}`}
                      onClick={() => rerunTarget(
                        { kind: 'host_check', check_name: checkName },
                        `check:${checkName}`,
                        `check:${checkName}`,
                        check.title
                      )}
                      disabled={rerunLoading[`check:${checkName}`]}
                    >
                      Re-run check
                    </button>
                  )}
                  <button
                    className="button is-small is-light"
                    onClick={() => toggleDetails(checkName)}
                  >
                    {showDetails[checkName] ? 'Hide details' : 'Show details'}
                    {showDetails[checkName] ? <FaChevronUp className="ml-2" /> : <FaChevronDown className="ml-2" />}
                  </button>
                </div>

                <div className="report-check-badges">
                  <span className="tag is-danger is-light">{failedDetails.length} failed</span>
                  <span className="tag is-success is-light">{passedDetails.length} passed</span>
                </div>

                <p className="report-check-copy">{check.description}</p>
                {!canRerunChecks && (
                  <p className="report-check-copy mb-3">Read-only access. Re-runs are limited to operators.</p>
                )}

                {showDetails[checkName] && (
                  <div className="check_details has-text-left py-3 px-3 my-3">
                    <h5 className="is-size-6 write">Description</h5>
                    <p className="is-size-6 has-text-weight-light">{check.description}</p>
                    <h5 className="is-size-6 write mt-3">
                      {check.url ? 'URL' : check.service ? 'Service' : 'Command'}
                    </h5>
                    <p><code className="is-size-7">{check.url || check.service || check.command}</code></p>
                    <h5 className="is-size-6 write mt-3">Failed when</h5>
                    <p>
                      <span className="is-size-7">
                        {Array.isArray(check.fail_value)
                          ? check.fail_value.map((val, valueIndex) => (
                              <span key={valueIndex}>
                                <span>{valueIndex > 0 ? ' or ' : ''}</span>
                                <code>result {formatFailWhen(check.fail_when)} {val}</code>
                              </span>
                            ))
                          : <code>result {formatFailWhen(check.fail_when)} {formatFailValues(check.fail_value)}</code>}
                      </span>
                    </p>
                  </div>
                )}

                <div className="buttons-container mb-0 no-print report-inline-buttons">
                  {hasFailedDetails && (
                    <button onClick={() => toggleSection(`${checkName}-failed`)} className="button is-grey is-light is-small">
                      {expandedSections[`${checkName}-failed`] ? 'Hide Failed Hosts' : 'Show Failed Hosts'}
                      <span className="tag is-danger is-light ml-2">{failedDetails.length}</span>
                      {expandedSections[`${checkName}-failed`] ? <FaChevronUp className="ml-2" /> : <FaChevronDown className="ml-2" />}
                    </button>
                  )}
                  {hasPassedDetails && (
                    <button onClick={() => toggleSection(`${checkName}-passed`)} className="button is-grey is-light is-small ml-2">
                      {expandedSections[`${checkName}-passed`] ? 'Hide Passed Hosts' : 'Show Passed Hosts'}
                      <span className="tag is-success is-light ml-2">{passedDetails.length}</span>
                      {expandedSections[`${checkName}-passed`] ? <FaChevronUp className="ml-2" /> : <FaChevronDown className="ml-2" />}
                    </button>
                  )}
                </div>
              </div>
            </div>

              {rerunFeedback[`check:${checkName}`] && (
                <div className="notification is-light mt-3">{rerunFeedback[`check:${checkName}`]}</div>
              )}

              {expandedSections[`${checkName}-passed`] && (
                renderHostTable(checkName, check.title, passedDetails, 'Passed hosts', false)
              )}

              {expandedSections[`${checkName}-failed`] && summary[checkName].failed > 0 && (
                renderHostTable(checkName, check.title, failedDetails, 'Failed hosts', true)
              )}
          </div>
        );
      })}
      </div>
      <CheckHistoryModal detailTarget={detailTarget} onClose={() => setDetailTarget(null)} />
    </div>
  );
};

export default CheckReport;
