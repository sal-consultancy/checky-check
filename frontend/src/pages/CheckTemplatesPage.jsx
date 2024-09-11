import React, { useState } from "react";
import { CopyToClipboard } from "react-copy-to-clipboard";
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';

const CheckTemplatesPage = () => {
  const [copied, setCopied] = useState(""); // Status voor gekopieerde code

  const tests = [
    {
      title: "Uptime",
      description: "Controleer of er een variabele gezet wordt in deze app",
      command: "uptime_value=$(uptime | awk '{print $3}' | tr -d ','); if echo $uptime_value | grep -qE '^[0-9]{1,2}:[0-9]{2}$'; then echo 0; else echo $uptime_value; fi",
      timeout: "5s",
      graph: {
        title: "Uptime per server",
        type: "bar_grouped_by_value"
      },
      fail_when: ">",
      fail_value: "100"
    },
    {
      title: "Disk Space",
      description: "Controleer of de beschikbare schijfruimte voldoende is",
      command: "df -h | grep /dev/sda1",
      timeout: "5s",
      graph: {
        title: "Disk Space per server",
        type: "bar_grouped_by_value"
      },
      fail_when: "<",
      fail_value: "20G"
    },
    {
       title:"Uptime",
      description:"Controleer of er een variabele gezet wordt in deze app",
      command: "sudo -k; uptime_value=$(echo lalapassword | sudo -S -u sapadm uptime 2>/dev/null | awk '{print $3}' | tr -d ','); if echo $uptime_value | grep -qE '^[0-9]{1,2}:[0-9]{2}$'; then echo 0; else echo $uptime_value; fi",
      become_user: "sapadm",
      timeout:"5s",
      graph: {
          title:"Uptime per server",
          type: "bar_grouped_by_value",
          sshow: false,
          llegend: true,
          ccolors: { "failed": [ "blue" ], "passed": [ "green" ] }
      },
      fail_when: ">",
      fail_value: "100"
  }    
  ];

  // Functie om bij te houden welke code is gekopieerd
  const handleCopy = (title) => {
    setCopied(title);
    setTimeout(() => setCopied(""), 2000); // Reset status na 2 seconden
  };

  // Functie om eventuele spaties aan het begin/einde van de code te verwijderen
  const trimCode = (code) => code.trim();

  // Stijlen voor de codeblokken
  const customStyle = {
    fontSize: '12px',
    padding: '15px',
    borderRadius: '5px',
    backgroundColor: '#2d2d2d',
    border: '1px solid #ddd',
    color: '#f8f8f2'
  };

  return (
    <div>
      <h1 className="title my-6">Check Templates Page</h1>
      <p>
        Welkom op de Check Templates Page. Hier vind je enkele voorbeelden van checks die je in je applicatie kunt gebruiken. 
        Elk voorbeeld bevat een knop om de code gemakkelijk naar je klembord te kopiëren.
      </p>

      <div className="has-text-left">
        {tests.map((test, index) => (
          <div key={index} className="mb-5">
            <h3 className="title is-4 mt-5">{test.title}</h3>

            {/* SyntaxHighlighter toont de code in een mooi format */}
            <SyntaxHighlighter language="json" style={customStyle}>
              {trimCode(JSON.stringify(test, null, 2))}
            </SyntaxHighlighter>

            {/* Copy button */}
            <CopyToClipboard text={trimCode(JSON.stringify(test, null, 2))} onCopy={() => handleCopy(test.title)}>
              <button className={`button is-dark is-small mt-2 ${copied === test.title ? "is-success" : ""}`}>
                {copied === test.title ? "Gekopieerd" : "Kopieer naar klembord"}
              </button>
            </CopyToClipboard>

            {/* Succesmelding bij gekopieerde code */}
            {copied === test.title && <p className="help is-success">Code is gekopieerd!</p>}
          </div>
        ))}
      </div>
    </div>
  );
};

export default CheckTemplatesPage;
