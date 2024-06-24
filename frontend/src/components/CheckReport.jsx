import React, { useState } from 'react';
import ChartComponent from './ChartComponent';
import { FaChevronDown, FaChevronUp } from 'react-icons/fa';
import { FaPlus, FaMinus } from 'react-icons/fa';

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

        let graphData;
        if (check.graph.type === 'bar_grouped_by_10_percentile') {
          graphData = new Array(11).fill(0).map((_, i) => ({
            value: 0,
            failed: 0,
          }));

          summary[checkName].details.forEach(detail => {
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

        const data = labels.map((label, index) =>
          check.graph.type === 'bar_grouped_by_10_percentile'
            ? graphData[index]
            : graphData[label]
        );

        const hasPassedDetails = summary[checkName].details.some(detail => detail.status === 'passed');
        const hasFailedDetails = summary[checkName].details.some(detail => detail.status === 'failed');

        return (
          <React.Fragment key={checkName}>
            {index > 0 && <hr className="separator" />}
            <div className="check-section">
              <div className="check-header" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <h3 className="is-size-3 write" id={checkName}>{check.title}</h3>
                <a
                  className='no-print'
                  onClick={() => toggleDetails(checkName)}
                  style={{ cursor: 'pointer', color: '#3273dc' }}
                  
                >
                  {showDetails[checkName] ? '' : ''}
                  {showDetails[checkName] ? <FaChevronUp className="ml-2" /> : <FaChevronDown className="ml-2" />}
                  
                </a>
              </div>
              <p className="is-size-6 print-only">{check.description}</p>

              {showDetails[checkName] && (
                <div className='no-print check_details has-text-left has-background-light py-3 px-3 my-3'>
                    <h5 className="is-size-5 write">Check Description</h5>
                    <p className="is-size-6">{check.description}</p>
                    <h5 className="is-size-5 write mt-3">{check.service ? 'Service' : 'Command'}</h5>
                    <p><code className="is-size-7">{check.service || check.command}</code></p>
                    <h5 className="is-size-5 write mt-3">Failed when </h5>
                    <p><code className="is-size-7">result {check.fail_when} {check.fail_value}</code></p>
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
