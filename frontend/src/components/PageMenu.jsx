import React from "react";
import { Link, useLocation } from "react-router-dom";

const PageMenu = ({ items }) => {
  const location = useLocation();

  if (!items) {
    return null; // Of een loading indicator
  }

  return (
    <aside className="menu my-5 ">
      <p className="menu-label">Reports</p>
      <ul className="menu-list write">
        <li>
        <Link to="/">Report</Link>
        <Link to="/summary">Summary</Link>
        </li>
      </ul>
      <p className="menu-label">Checks</p>

      <nav className="menu">
      <ul className="menu-list write">
        {Object.keys(items).map(item => (
          <li key={item}>
            <a href={`/#${item}`}>{items[item].title}</a>
          </li>
        ))}
      </ul>
    </nav>
      <p className="menu-label">CheckyCheck</p>
      <ul className="menu-list write">
        <li>
        <Link
            to="/help"
            className={`navbar-item ${
              location.pathname === "/help" ? "has-background-light" : ""
            }`}
          >
            Help
          </Link>
        </li>
      </ul>
    </aside>
  );
};

export default PageMenu;
