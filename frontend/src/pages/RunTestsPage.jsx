import React, { useState, useEffect } from 'react';

const RunTestsPage = ({ onTestsComplete }) => {
  const [loading, setLoading] = useState(false); // Houdt bij of de tests bezig zijn
  const [output, setOutput] = useState(''); // Houdt de uitvoer van de tests bij

  const runTests = async () => {
    setLoading(true);  // Zet de laadstatus op 'true'
    setOutput('');     // Reset de uitvoer

    try {
      const response = await fetch('/run-tests', { method: 'POST' }); // Stuur het verzoek naar de server
      const reader = response.body.getReader(); // Lees de response stream
      const decoder = new TextDecoder('utf-8');
      let done = false;

      while (!done) {
        const { value, done: readerDone } = await reader.read(); // Lees stukjes data
        done = readerDone;
        const chunk = decoder.decode(value, { stream: true }); // Decodeer de data
        setOutput(prevOutput => prevOutput + chunk); // Voeg de data toe aan de output
      }

      onTestsComplete(); // Roep de callback aan om resultaten opnieuw op te halen

    } catch (error) {
      console.error('Error running tests:', error); // Log fouten
    }

    setLoading(false); // Zet de laadstatus terug op 'false'
  };

  useEffect(() => {
    runTests(); // Voer de tests uit wanneer de component wordt geladen
  }, []); // De lege dependency array zorgt ervoor dat de tests slechts één keer worden uitgevoerd

  return (
    <div>
      <h2>Running Tests</h2>
      {loading ? <p>Loading...</p> : <p>Tests completed.</p>} {/* Toon de status van het laden */}
      <pre>{output}</pre> {/* Toon de uitvoer van de tests */}
    </div>
  );
};

export default RunTestsPage;
