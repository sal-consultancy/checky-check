import React, { useState } from 'react';
import ChartComponent from './ChartComponent';
import { FaChevronDown, FaChevronUp } from 'react-icons/fa';

const CheckReport = ({ results, checks, theme }) => {
  const [expandedSections, setExpandedSections] = useState({});
  const [showDetails, setShowDetails] = useState({});

  const toggleSection = section => {
    setExpandedSections(prevState => ({
      ...prevState,
      [section]: !prevState[section],
    }));
  };

  const toggleDetails = checkName => {
    setShowDetails(prevState => ({
      ...prevState,
      [checkName]: !prevState[checkName],
    }));
  };

  const summary = Object.keys(checks).reduce((acc, checkName) => {
    acc[checkName] = { passed: 0, failed: 0, details: [] };

    Object.keys(results).forEach(host => {
      if (results[host][checkName]) {
        const result = results[host][checkName];
        if (result.status === 'passed') {
          acc[checkName].passed += 1;
        } else {
          acc[checkName].failed += 1;
        }
        acc[checkName].details.push({ host, ...result });
      }
    });

    return acc;
  }, {});

  return (
    <div className="check-report">
      {Object.keys(summary).map((checkName, index) => {
        const check = checks[checkName];

        const graphData = summary[checkName].details.reduce((acc, detail) => {
          if (check.graph?.type === 'bar_grouped_by_10_percentile') {
            const value = parseInt(detail.value, 10);
            const bucket = Math.min(Math.floor(value / 10), 10);
            const label = `${bucket * 10}-${bucket * 10 + 9}%`;
            acc[label] = (acc[label] || 0) + 1;
          } else {
            acc[detail.value] = (acc[detail.value] || 0) + 1;
          }
          return acc;
        }, {});

        // Ensure all 10-percentile buckets are included
        if (check.graph?.type === 'bar_grouped_by_10_percentile') {
          for (let i = 0; i <= 10; i++) {
            const label = i === 10 ? '100%' : `${i * 10}-${i * 10 + 9}%`;
            if (!graphData[label]) {
              graphData[label] = 0;
            }
          }
        }

        const labels = Object.keys(graphData).sort((a, b) => {
          if (a === '100%') return 1;
          if (b === '100%') return -1;
          return parseInt(a) - parseInt(b);
        });
        const data = labels.map(label => ({
          value: graphData[label],
          failed: summary[checkName].details.filter(d => d.value === label && d.status === 'failed').length
        }));

        const hasPassedDetails = summary[checkName].details.some(detail => detail.status === 'passed');
        const hasFailedDetails = summary[checkName].details.some(detail => detail.status === 'failed');

        return (
          <React.Fragment key={checkName}>
            {index > 0 && <hr className="separator" />}
            <div className="check-section">
              <div className="check-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <h3 className="is-size-3 write mt-5" id={checkName}>{check.title}</h3>
                <a
                  onClick={() => toggleDetails(checkName)}
                  style={{ cursor: 'pointer', color: '#3273dc' }}
                >
                  {showDetails[checkName] ? 'Hide Check Details' : 'Show Check Details'}
                  {showDetails[checkName] ? <FaChevronUp className="ml-2" /> : <FaChevronDown className="ml-2" />}
                </a>
              </div>

              {showDetails[checkName] && (
                <div className='check_details has-text-left'>
                  <h5 className="is-size-5 write mt-3">Description</h5>
                  <p className="is-size-6">{check.description}</p>
                  <h5 className="is-size-5 write mt-3">Failed when </h5>
                  <p><code className="is-size-7">result {check.fail_when} {check.fail_value}</code></p>
                  <h5 className="is-size-5 write mt-3">Command</h5>
                  <p><code className="is-size-7">{check.command}</code></p>
                </div>
              )}

              <div className="columns is-multiline mt-5">
                <ChartComponent
                  data={data}
                  labels={labels}
                  title={check.graph.title}
                  theme={theme}
                  type={check.graph?.type === 'pie_grouped_by_value' ? 'pie' : 'bar'}
                  colors={check.graph?.colors || { failed: ['red'], passed: ['green'] }}
                />
              </div>
             
              <div className="buttons-container mb-5">
                {hasPassedDetails && (
                  <button onClick={() => toggleSection(`${checkName}-passed`)} className="button is-grey is-light is-small">
                    {expandedSections[`${checkName}-passed`] ? 'Hide Passed Hosts' : 'Show Passed Hosts'}
                    <span className="tag is-success is-light ml-2">
                      {summary[checkName].details.filter(detail => detail.status === 'passed').length}
                    </span>
                    {expandedSections[`${checkName}-passed`] ? <FaChevronUp className="ml-2" /> : <FaChevronDown className="ml-2" />}
                  </button>
                )}
                {hasFailedDetails && (
                  <button onClick={() => toggleSection(`${checkName}-failed`)} className="button is-grey is-light is-small ml-2">
                    {expandedSections[`${checkName}-failed`] ? 'Hide Failed Hosts' : 'Show Failed Hosts'}
                    <span className="tag is-danger is-light ml-2">
                      {summary[checkName].details.filter(detail => detail.status === 'failed').length}
                    </span>
                    {expandedSections[`${checkName}-failed`] ? <FaChevronUp className="ml-2" /> : <FaChevronDown className="ml-2" />}
                  </button>
                )}
              </div>

              {expandedSections[`${checkName}-passed`] && (
                <>
                  <h5 className="is-size-5 write mt-3 has-text-left">Passed hosts</h5>
                  <table className="table is-striped is-bordered is-size-7 mt-2">
                    <thead>
                      <tr>
                        <th>Host</th>
                        <th>Status</th>
                        <th>Value</th>
                        <th>Timestamp</th>
                      </tr>
                    </thead>
                    <tbody>
                      {summary[checkName].details.filter(detail => detail.status === 'passed').map(detail => (
                        <tr key={detail.host} className="">
                          <td>{detail.host}</td>
                          <td>{detail.status}</td>
                          <td>{detail.value}</td>
                          <td>{detail.timestamp}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </>
              )}

              {expandedSections[`${checkName}-failed`] && (
                <>
                  <h5 className="is-size-5 write mt-3 has-text-left">Failed hosts</h5>
                  <table className="table is-striped is-bordered is-size-7 mt-2">
                    <thead>
                      <tr>
                        <th>Host</th>
                        <th>Status</th>
                        <th>Value</th>
                        <th>Timestamp</th>
                      </tr>
                    </thead>
                    <tbody>
                      {summary[checkName].details.filter(detail => detail.status === 'failed').map(detail => (
                        <tr key={detail.host} className="">
                          <td>{detail.host}</td>
                          <td>{detail.status}</td>
                          <td>{detail.value}</td>
                          <td>{detail.timestamp}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </>
              )}
            </div>
          </React.Fragment>
        );
      })}
    </div>
  );
};

export default CheckReport;