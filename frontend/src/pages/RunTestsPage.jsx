import React, { useState } from 'react';

const RUN_STATUS_PREFIX = '__CHECKY_CHECK_RUN_STATUS__:';

const RunTestsPage = ({ onTestsComplete, authSession }) => {
  const [loading, setLoading] = useState(false);
  const [output, setOutput] = useState('');
  const [hasRun, setHasRun] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');
  const [runStatus, setRunStatus] = useState('idle');
  const canRunChecks = authSession?.permissions?.operate ?? true;

  const runTests = async () => {
    if (!canRunChecks) {
      return;
    }

    setLoading(true);
    setHasRun(true);
    setOutput('');
    setErrorMessage('');
    setRunStatus('running');

    try {
      const response = await fetch('/api/run-tests', { method: 'POST' });
      let currentRunStatus = 'running';

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(errorText || 'The test run failed.');
      }

      const reader = response.body.getReader();
      const decoder = new TextDecoder('utf-8');
      let done = false;
      let pending = '';

      const appendChunk = (chunk) => {
        pending += chunk;
        const lines = pending.split('\n');
        pending = lines.pop() || '';

        const visibleLines = [];
        lines.forEach((line) => {
          if (line.startsWith(RUN_STATUS_PREFIX)) {
            const nextStatus = line.slice(RUN_STATUS_PREFIX.length).trim();
            currentRunStatus = nextStatus || 'running';
            setRunStatus(nextStatus || 'running');
            return;
          }
          visibleLines.push(line);
        });

        if (visibleLines.length > 0) {
          setOutput((prevOutput) => `${prevOutput}${visibleLines.join('\n')}\n`);
        }
      };

      while (!done) {
        const { value, done: readerDone } = await reader.read();
        done = readerDone;
        if (value) {
          const chunk = decoder.decode(value, { stream: true });
          appendChunk(chunk);
        }
      }

      if (pending) {
        if (pending.startsWith(RUN_STATUS_PREFIX)) {
          const nextStatus = pending.slice(RUN_STATUS_PREFIX.length).trim();
          currentRunStatus = nextStatus || 'running';
          setRunStatus(nextStatus || 'running');
        } else {
          setOutput((prevOutput) => `${prevOutput}${pending}`);
        }
      }

      if (currentRunStatus !== 'failed') {
        onTestsComplete();
      }
    } catch (error) {
      console.error('Error running tests:', error);
      setErrorMessage(error.message || 'The test run failed.');
      setRunStatus('failed');
    }

    setLoading(false);
  };

  return (
    <div>
      <h2 className="title is-4">Run Checks</h2>
      <p className="mb-4 has-text-left">
        Start a new run manually. When the run finishes, the latest results are reloaded automatically.
      </p>
      {!canRunChecks && (
        <div className="notification is-warning is-light has-text-left">
          <strong>Read-only access.</strong>
          <div className="mt-2">Manual runs are limited to operators and admins.</div>
        </div>
      )}
      <div className="buttons">
        <button className={`button is-dark ${loading ? 'is-loading' : ''}`} onClick={runTests} disabled={loading || !canRunChecks}>
          {loading ? 'Running Checks' : 'Run Checks Now'}
        </button>
      </div>
      {loading && (
        <p className="has-text-left">
          <span className="tag is-info is-light mr-2">Running</span>
          Checks are running. Output will appear below.
        </p>
      )}
      {!loading && hasRun && !errorMessage && runStatus === 'success' && (
        <p className="has-text-left">
          <span className="tag is-success is-light mr-2">Completed</span>
          Checks completed.
        </p>
      )}
      {!loading && hasRun && !errorMessage && runStatus === 'failed' && (
        <p className="has-text-left">
          <span className="tag is-danger is-light mr-2">Failed</span>
          Checks finished with errors.
        </p>
      )}
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
