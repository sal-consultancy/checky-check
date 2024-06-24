import React from 'react';

const Footer = ({ copyright }) => {
  return (
    <footer className="footer">
      <div className="content has-text-centered">
        <p>{copyright}</p>
      </div>
    </footer>
  );
};

export default Footer;
