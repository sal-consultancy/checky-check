import React, { useEffect, useState } from 'react';
import CheckReport from './components/CheckReport';
import './App.css';
import 'bulma/css/bulma.css';
import PageMenu from './components/PageMenu';
import heartIcon from './images/heartpulse.svg';
import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';
import HelpPage from './pages/HelpPage';
import SummaryPage from './pages/SummaryPage';


const App = () => {
  const [results, setResults] = useState({ checks: {}, results: {} });
  const [theme, setTheme] = useState('light');

  useEffect(() => {
    fetch('/results')
      .then(response => response.json())
      .then(data => setResults(data))
      .catch(error => console.error('Error fetching results:', error));
  }, []);

  return (
    <Router>
    <div className="App">
          <section className="hero is-light">
            <div className="hero-body">
              <img className='' src={heartIcon} alt="court icon" width="50"/>
              <p className="title">CheckyCheck</p>
              <p className="subtitle write">Ad hoc monitoring</p>
            </div>
          </section>
          <section>
            <div className="fixed-grid has-12-cols">
              <div className="grid">
                <div className="cell is-col-start-3 is-col-span-2">
                  <PageMenu items={results.checks}/>
                </div>
                <div className="cell is-col-start-5 is-col-span-5 my-5">
                  <Routes>
                    <Route path="/" element={<CheckReport results={results.results} checks={results.checks} theme={theme}/>} />
                    <Route path="/summary" element={<SummaryPage results={results.results} checks={results.checks} />} />
                    <Route path="/help" element={<HelpPage />} />
                  </Routes>
                </div>
              </div>
            </div>
          </section>

    </div>
    </Router>
  );
};

export default App;
