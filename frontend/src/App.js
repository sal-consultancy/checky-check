import React, { useEffect, useState } from 'react';
import CheckReport from './components/CheckReport';
import './App.css';
import 'bulma/css/bulma.css';
import PageMenu from './components/PageMenu';
import Footer from './components/Footer';
import heartIcon from './images/heartpulse.svg';
import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';
import HelpPage from './pages/HelpPage';
import SummaryPage from './pages/SummaryPage';
import RunTestsPage from './pages/RunTestsPage';  // Importeer de nieuwe component
import CheckTemplatesPage from './pages/CheckTemplatesPage';  // Importeer de nieuwe component

const App = () => {
  const [results, setResults] = useState({ checks: {}, results: {}, report: {} });
  const [theme] = useState('light');

  const fetchResults = () => {
    fetch('/results')
      .then(response => response.json())
      .then(data => setResults(data))
      .catch(error => console.error('Error fetching results:', error));
  };

  useEffect(() => {
    fetchResults();
  }, []);

  const handleTestsComplete = () => {
    fetchResults();
  };



  return (
    <Router>
      <div className={`App ${theme}`}>
        <section className={`hero is-${theme}`}>
          <div className="hero-body">
            <img className='' src={heartIcon} alt="court icon" width="50"/>
            <p className="title">{results.report.title}</p>
            <p className="subtitle write">{results.report.subtitle}</p>
          </div>
        </section>
        <section>
          <div className="fixed-grid has-12-cols">
            <div className="grid">
              <div className="cell is-col-start-3 is-col-span-2 no-print">
                <PageMenu items={results.checks} />
              </div>
              <div className="cell is-col-start-5 is-col-span-4 my-5 print-adjust">
                <Routes>
                  <Route path="/report" element={<CheckReport results={results.results} checks={results.checks} theme={theme} />} />
                  <Route path="/" element={<SummaryPage results={results.results} checks={results.checks} />} />
                  <Route path="/help" element={<HelpPage />} />
                  <Route path="/templates" element={<CheckTemplatesPage />} />
                  <Route path="/run-tests" element={<RunTestsPage onTestsComplete={handleTestsComplete} />} />
                </Routes>
              </div>
            </div>
          </div>
        </section>
      </div>
      <Footer copyright={results.report?.copyright} />
    </Router>
  );
};

export default App;
