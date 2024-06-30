import React from "react";
import { Link, useLocation } from "react-router-dom";

const PageMenu = ({ items }) => {
  const location = useLocation();

  if (!items) {
    return null; // Of een loading indicator
  }

  const isCheckSelected = Object.keys(items).some(
    item => location.hash === `#${item}`
  );

  return (
    <aside className="menu my-5">
      <p className="menu-label">Reports</p>
      <ul className="menu-list write">
        <li>
          <Link
            to="/"
            className={location.pathname === "/" ? "has-background-light" : ""}
          >
            Summary
          </Link>
        </li>
        <li>
          <Link
            to="/report"
            className={
              location.pathname === "/report" && !isCheckSelected
                ? "has-background-light"
                : ""
            }
          >
            Report
          </Link>
        </li>
      </ul>

      {location.pathname === "/report" && (
        <>
          <p className="menu-label">Checks</p>
          <nav className="menu">
            <ul className="menu-list write">
              {Object.keys(items).map(item => (
                <li key={item}>
                  <a
                    href={`#${item}`}
                    className={location.hash === `#${item}` ? "has-background-light" : ""}
                  >
                    {items[item].title}
                  </a>
                </li>
              ))}
            </ul>
          </nav>
        </>
      )}

      <p className="menu-label">CheckyCheck</p>
      <ul className="menu-list write">
        <li>
          <Link
            to="/help"
            className={location.pathname === "/help" ? "has-background-light" : ""}
          >
            Help
          </Link>
        </li>
      </ul>
    </aside>
  );
};

export default PageMenu;
