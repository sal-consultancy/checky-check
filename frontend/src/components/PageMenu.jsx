import React from "react";
import { Link, useLocation } from "react-router-dom";

const PageMenu = () => {
  const location = useLocation();

  return (
    <nav className="header-nav no-print" aria-label="Primary">
      <ul className="header-nav-list write">
        <li>
          <Link
            to="/"
            className={location.pathname === "/" ? "is-active" : ""}
          >
            Summary
          </Link>
        </li>
        <li>
          <Link
            to="/report"
            className={location.pathname === "/report" ? "is-active" : ""}
          >
            Report
          </Link>
        </li>
        <li>
          <Link
            to="/hosts"
            className={location.pathname === "/hosts" ? "is-active" : ""}
          >
            Hosts
          </Link>
        </li>
        <li>
          <Link
            to="/history"
            className={location.pathname === "/history" ? "is-active" : ""}
          >
            History
          </Link>
        </li>
        <li>
          <Link
            to="/diagnostics"
            className={location.pathname === "/diagnostics" ? "is-active" : ""}
          >
            Diagnostics
          </Link>
        </li>
        <li>
          <Link
            to="/help"
            className={location.pathname === "/help" ? "is-active" : ""}
          >
            Help
          </Link>
        </li>
      </ul>
    </nav>
  );
};

export default PageMenu;
