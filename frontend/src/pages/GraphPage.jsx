import React, { useContext } from 'react';
import ChartComponent from '../components/ChartComponent';
import { JobContext } from '../JobContext';

const GraphPage = () => {
  const { selectedJob } = useContext(JobContext);

  return (
    <ChartComponent selectedJob={selectedJob} />
  );
};

export default GraphPage;
