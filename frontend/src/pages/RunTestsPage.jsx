import React, { useState, useEffect } from 'react';

const RunTestsPage = ({ onTestsComplete }) => {
  const [loading, setLoading] = useState(false);
  const [output, setOutput] = useState('');

  const runTests = async () => {
    setLoading(true);
    setOutput('');

    try {
      const response = await fetch('/run-tests', { method: 'POST' });
      const reader = response.body.getReader();
      const decoder = new TextDecoder('utf-8');
      let done = false;

      while (!done) {
        const { value, done: readerDone } = await reader.read();
        done = readerDone;
        const chunk = decoder.decode(value, { stream: true });
        setOutput(prevOutput => prevOutput + chunk);
      }

      onTestsComplete(); // Roep de callback aan om de resultaten opnieuw op te halen

    } catch (error) {
      console.error('Error running tests:', error);
    }

    setLoading(false);
  };

  useEffect(() => {
    runTests();
  }, []);

  return (
    <div>
      <h2>Running Tests</h2>
      {loading ? <p>Loading...</p> : <p>Tests completed.</p>}
      <pre>{output}</pre>
    </div>
  );
};

export default RunTestsPage;
