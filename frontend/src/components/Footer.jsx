import React from 'react';
import { FaGithub } from "react-icons/fa";

const Footer = ({ copyright }) => {
  return (
    <footer className="footer">
      <div className="content has-text-centered">
        <p>{copyright}</p>
        <p><a style={{ color: 'grey' }} target="_blank" rel="noreferrer" href="https://github.com/sal-consultancy/checky-check"><FaGithub /> Github</a></p>
      </div>
    </footer>
  );
};

export default Footer;
