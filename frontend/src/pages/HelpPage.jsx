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
      </div>
    </div>
  );
};

export default HelpPage;
