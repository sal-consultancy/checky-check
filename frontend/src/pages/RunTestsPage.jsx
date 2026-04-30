import React, { useState } from 'react';

const RunTestsPage = ({ onTestsComplete }) => {
  const [loading, setLoading] = useState(false);
  const [output, setOutput] = useState('');
  const [hasRun, setHasRun] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');

  const runTests = async () => {
    setLoading(true);
    setHasRun(true);
    setOutput('');
    setErrorMessage('');

    try {
      const response = await fetch('/run-tests', { method: 'POST' });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(errorText || 'The test run failed.');
      }

      const reader = response.body.getReader();
      const decoder = new TextDecoder('utf-8');
      let done = false;

      while (!done) {
        const { value, done: readerDone } = await reader.read();
        done = readerDone;
        if (value) {
          const chunk = decoder.decode(value, { stream: true });
          setOutput(prevOutput => prevOutput + chunk);
        }
      }

      onTestsComplete();
    } catch (error) {
      console.error('Error running tests:', error);
      setErrorMessage(error.message || 'The test run failed.');
    }

    setLoading(false);
  };

  return (
    <div>
      <h2 className="title is-4">Run Checks</h2>
      <p className="mb-4 has-text-left">
        Start a new run manually. When the run finishes, the latest results are reloaded automatically.
      </p>
      <div className="buttons">
        <button className={`button is-dark ${loading ? 'is-loading' : ''}`} onClick={runTests} disabled={loading}>
          {loading ? 'Running Checks' : 'Run Checks Now'}
        </button>
      </div>
      {loading && <p className="has-text-left">Checks are running. Output will appear below.</p>}
      {!loading && hasRun && !errorMessage && <p className="has-text-left">Checks completed.</p>}
      {errorMessage && (
        <div className="notification is-danger is-light has-text-left">
          <strong>Run failed.</strong>
          <div className="mt-2">{errorMessage}</div>
        </div>
      )}
      <pre className="has-text-left">{output}</pre>
    </div>
  );
};

export default RunTestsPage;
