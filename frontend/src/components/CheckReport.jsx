import React, { useState } from 'react';
import ChartComponent from './ChartComponent';

const CheckReport = ({ results, checks, theme }) => {
  const [expandedSections, setExpandedSections] = useState({});

  const toggleSection = section => {
    setExpandedSections(prevState => ({
      ...prevState,
      [section]: !prevState[section],
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
      {Object.keys(summary).map(checkName => {
        const check = checks[checkName];

        // Groepeer de data voor de grafiek
        const graphData = summary[checkName].details.reduce((acc, detail) => {
          acc[detail.value] = (acc[detail.value] || 0) + 1;
          return acc;
        }, {});

        const labels = Object.keys(graphData);
        const data = labels.map(label => ({ value: graphData[label], failed: summary[checkName].details.filter(d => d.value === label && d.status === 'failed').length }));

        return (
          <div key={checkName} className="check-section">
            <h3 className="is-size-3 write mt-5" id={checkName}>{check.title}</h3>
            <p>{check.description}</p>
            <div className="columns is-multiline">
              <ChartComponent
                data={data}
                labels={labels}
                title={check.title}
                theme={theme}
                type={check.graph?.type === 'pie_grouped_by_value' ? 'pie' : 'bar'}
                colors={check.graph?.colors || { failed: ['red'], passed: ['green'] }}
              />
            </div>
            <button onClick={() => toggleSection(`${checkName}-passed`)} className="button is-link is-small">
              {expandedSections[`${checkName}-passed`] ? 'Hide' : 'Show'} Passed Details
            </button>
            {expandedSections[`${checkName}-passed`] && (
              <table className="table is-striped is-bordered is-size-7">
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
            )}
            <button onClick={() => toggleSection(`${checkName}-failed`)} className="button is-link is-small mt-2">
              {expandedSections[`${checkName}-failed`] ? 'Hide' : 'Show'} Failed Details
            </button>
            {expandedSections[`${checkName}-failed`] && (
              <table className="table is-striped is-bordered is-size-7">
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
            )}
          </div>
        );
      })}
    </div>
  );
};

export default CheckReport;
