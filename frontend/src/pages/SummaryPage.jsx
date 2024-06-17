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

  const allHosts = Object.keys(results).map(host => {
    const hostResults = results[host];
    return {
      host,
      results: Object.keys(hostResults).map(checkName => ({
        checkName,
        status: hostResults[checkName].status,
        value: hostResults[checkName].value,
        timestamp: hostResults[checkName].timestamp
      }))
    };
  });

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

      <h2>Alle Hosts</h2>
      <table className="table is-striped is-bordered">
        <thead>
          <tr>
            <th>Host</th>
            <th>Testnaam</th>
            <th>Status</th>
            <th>Value</th>
            <th>Timestamp</th>
          </tr>
        </thead>
        <tbody>
          {allHosts.map(({ host, results }) =>
            results.map((result, index) => (
              <tr key={`${host}-${result.checkName}-${index}`}>
                <td>{index === 0 ? host : ''}</td>
                <td>{checks[result.checkName].title}</td>
                <td>{result.status}</td>
                <td>{result.value}</td>
                <td>{result.timestamp}</td>
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
};

export default CheckReport;
