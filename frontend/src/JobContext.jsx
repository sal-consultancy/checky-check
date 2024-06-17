import React, { createContext, useState } from 'react';

export const JobContext = createContext();

export const JobProvider = ({ children }) => {
  const [selectedJob, setSelectedJob] = useState('');

  return (
    <JobContext.Provider value={{ selectedJob, setSelectedJob }}>
      {children}
    </JobContext.Provider>
  );
};
