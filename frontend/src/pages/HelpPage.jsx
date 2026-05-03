import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';

const HelpPage = () => {
  const [version, setVersion] = useState('');

  useEffect(() => {
    fetch('/api/version')
      .then((response) => response.json())
      .then((data) => setVersion(data.version || 'development'))
      .catch((error) => {
        console.error('Error fetching version:', error);
        setVersion('unknown');
      });
  }, []);

  return (
    <div className="help-page">
      <div className="help-header">
        <div>
          <p className="help-kicker">Support</p>
          <h1 className="title is-3 mb-2">Help</h1>
          <p className="help-copy">
            CheckyCheck toont de laatste run, actuele failures en de historie van wijzigingen in checks.
          </p>
        </div>
        <span className="tag is-light">Version {version || '-'}</span>
      </div>

      <div className="help-grid">
        <section className="help-card">
          <h2>Dagelijkse flow</h2>
          <p>
            Start op <Link to="/">Summary</Link> voor de huidige status. Open daarna <Link to="/report">Report</Link> of <Link to="/hosts">Hosts</Link> om failures verder uit te splitsen.
          </p>
          <div className="help-link-list">
            <Link to="/">Open Summary</Link>
            <Link to="/report">Open Report</Link>
            <Link to="/hosts">Open Hosts</Link>
          </div>
        </section>

        <section className="help-card">
          <h2>Checks draaien</h2>
          <p>
            Gebruik <Link to="/run-tests">Run Checks</Link> om handmatig een volledige run te starten. Operators en admins kunnen ook individuele checks opnieuw draaien vanuit het rapport.
          </p>
          <div className="help-note">
            Resultaten worden na een run opnieuw geladen. De run zelf wordt ook in history opgeslagen.
          </div>
        </section>

        <section className="help-card">
          <h2>Failures beoordelen</h2>
          <p>
            Result-tabellen tonen de actuele waarde, de failure-regel en de verwachte failure value. Gebruik die combinatie om te zien waarom een host of URL faalt.
          </p>
          <div className="help-note">
            History toont alleen relevante events: nieuwe failures, recoveries, config errors en gewijzigde failure details.
          </div>
        </section>

        <section className="help-card">
          <h2>History</h2>
          <p>
            <Link to="/history">History</Link> laat recente runs en events zien. Selecteer een run om alleen de events van die uitvoering te bekijken.
          </p>
          <div className="help-link-list">
            <Link to="/history">Open History</Link>
          </div>
        </section>

        <section className="help-card">
          <h2>Account menu</h2>
          <p>
            Rechtsboven vind je je clearance, theme selector, Help en admin-acties. Diagnostics staat daar alleen voor admins.
          </p>
          <div className="help-note">
            Bij <code>auth: none</code> draait de UI als lokale admin. Bij proxy-auth komen gebruiker, groepen en rechten uit headers.
          </div>
        </section>
      </div>
    </div>
  );
};

export default HelpPage;
