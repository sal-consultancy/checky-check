// src/components/HelpPage.jsx
import React, { useEffect, useState } from "react";
import { Link } from "react-router-dom";

const HelpPage = () => {
  const [version, setVersion] = useState("");

  useEffect(() => {
    fetch("/api/version")
      .then((response) => response.json())
      .then((data) => setVersion(data.version))
      .catch((error) => console.error("Error fetching version:", error));
  }, []);

  return (
    <div>
      <h1 className="title my-6">Help</h1>
      <div className="has-text-left">
        <h3 className="title is-4 mt-5">Introduction</h3>
        <p>
          This is the help page for CheckyCheck. Here you can find information
          on how to use the application.
        </p>
        <h3 className="title is-4 mt-5">Appearance</h3>
        <p>
          The theme selector is available in the header and supports system,
          light and dark mode.
        </p>
        <h3 className="title is-4 mt-5">Refreshing data</h3>
        <p>
          Use the Run Checks action on the <Link to="/">summary</Link> page to refresh the data manually.
          The run page shows command output while the checks are running and reloads the latest results when the run finishes.
        </p>
        <h3 className="title is-4 mt-5">Check voorbeelden</h3>
        <p>
          Op de <Link to="/templates">templates</Link> pagina zijn voorbeelden van checks te vinden. 
        </p>
        <h3 className="title is-4 mt-5">Version</h3>
        <p>
          This is application version {version}.
        </p>

      </div>
    </div>
  );
};

export default HelpPage;
