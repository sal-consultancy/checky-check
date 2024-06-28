import React, { useEffect, useState } from "react";
import { CopyToClipboard } from "react-copy-to-clipboard";
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';

const CheckTemplatesPage = () => {
  const [version, setVersion] = useState("");
  const [copied, setCopied] = useState("");

  const tests = [
    {
      title: "Uptime",
      code: `
{
  "check_uptime": {
    "title": "Uptime",
    "description": "Controleer of er een variabele gezet wordt in deze app",
    "command": "uptime | awk '{print $3}'",
    "timeout": "5s",
    "graph": {
      "title": "Uptime per server",
      "type": "bar_grouped_by_value",
    },
    "fail_when": ">",
    "fail_value": "100"
  }
}`
    },
    {
      title: "Disk Space",
      code: `
{
  "check_disk_space": {
    "title": "Disk Space",
    "description": "Controleer of de beschikbare schijfruimte voldoende is",
    "command": "df -h | grep /dev/sda1",
    "timeout": "5s",
    "graph": {
      "title": "Disk Space per server",
      "type": "bar_grouped_by_value",
    },
    "fail_when": "<",
    "fail_value": "20G"
  }
}`
    }
  ];

  useEffect(() => {
    fetch("/api/version")
      .then((response) => response.json())
      .then((data) => setVersion(data.version))
      .catch((error) => console.error("Error fetching version:", error));
  }, []);

  const handleCopy = (title) => {
    setCopied(title);
    setTimeout(() => setCopied(""), 2000); // Reset copied state after 2 seconds
  };

  const trimCode = (code) => code.trim();

  const customStyle = {
    fontSize: '10px', // Ensure font size is applied
    padding: '10px',
    borderRadius: '5px',
    backgroundColor: '', // Set background to dark
    border: '1px solid #ddd', // Outline the code block
    color: '#f8f8f2' // Set text color to light for readability
  };

  return (
    <div>
      <h1 className="title my-6">Check Templates Page</h1>
      <p>Welcome to the Check Templates Page. Here you can find several examples of checks that you can use in your application. Each example includes a button to easily copy the code to your clipboard.</p>
      <div className="has-text-left">
        {tests.map((test, index) => (
          <div key={index} className="mb-5">
            <h3 className="title is-4 mt-5">{test.title}</h3>
            <SyntaxHighlighter language="json" customStyle={customStyle}>
              {trimCode(test.code)}
            </SyntaxHighlighter>
            <CopyToClipboard text={trimCode(test.code)} onCopy={() => handleCopy(test.title)}>
              <button className={`button is-dark is-small mt-2 ${copied === test.title ? "is-success" : ""}`}>
                {copied === test.title ? "Copied" : "Copy to Clipboard"}
              </button>
            </CopyToClipboard>
            {copied === test.title && <p className="help is-success">Code is gekopieerd!</p>}
          </div>
        ))}
      </div>
    </div>
  );
};

export default CheckTemplatesPage;
