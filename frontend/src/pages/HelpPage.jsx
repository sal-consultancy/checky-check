// src/components/HelpPage.jsx
import React, { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import ThemeToggle from "../components/ThemeToggle";

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
        <h3 className="title is-4 mt-5">Dark theme</h3>
        <p>
          With the button below you can change the theme from dark to light, 
          and the other way around.
        </p>
        <p>
          <ThemeToggle />
        </p>
        <h3 className="title is-4 mt-5">Refreshing data</h3>
        <p>
          You can refresh the data manually using the <Link to="/run-tests">run tests</Link> link. The tests will immediately start running.
          When the tests are done, the results, the output is shown on the page.
          
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
