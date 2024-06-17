import React from 'react';

const CheckReport = ({ results, checks, theme }) => {
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
      <h2>Samenvatting</h2>
      <table className="table is-striped is-bordered">
        <thead>
          <tr>
            <th>Testnaam</th>
            <th>Aantal Passed</th>
            <th>Aantal Failed</th>
          </tr>
        </thead>
        <tbody>
          {Object.keys(summary).map(checkName => (
            <tr key={checkName}>
              <td>{checks[checkName].title}</td>
              <td>{summary[checkName].passed}</td>
              <td>{summary[checkName].failed}</td>
            </tr>
          ))}
        </tbody>
      </table>


    </div>
  );
};

export default CheckReport;
