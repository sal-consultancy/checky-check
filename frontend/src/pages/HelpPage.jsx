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
        <p>
          This is the help page for CheckyCheck. Here you can find information
          on how to use the application.
        </p>
        This is application version {version}.
        <p>
        <ThemeToggle />
        </p>
        <h3 className="title-3">Refreshing data</h3>
        <p>
        You can refresh the data manually using the <code>/run-test</code> url. The tests will immediately start running.
        When the tests are done, the results, the output is shown on the page.
        <Link to="/run-tests">Run tests</Link>
        </p>
      </div>
    </div>
  );
};

export default HelpPage;
