import React, { useEffect, useState, useRef } from 'react';
import { Chart } from 'chart.js/auto';

const ChartComponent = () => {
  const [chartData, setChartData] = useState(null);
  const chartRef = useRef(null); // Create a ref for the canvas
  const chartInstanceRef = useRef(null); // Create a ref for the Chart instance

  useEffect(() => {
    fetch('/api/data')
      .then(response => response.json())
      .then(data => setChartData(data))
      .catch(error => console.error('Error fetching data:', error));
  }, []);

  useEffect(() => {
    if (chartData) {
      // If there is an existing Chart instance, destroy it before creating a new one
      if (chartInstanceRef.current) {
        chartInstanceRef.current.destroy();
      }

      const ctx = chartRef.current.getContext('2d');
      chartInstanceRef.current = new Chart(ctx, {
        type: 'bar',
        data: {
          labels: chartData.Labels.split(', '),
          datasets: [
            {
              label: 'Success',
              data: chartData.SuccessData.split(', ').map(Number),
              backgroundColor: 'rgba(75, 192, 192, 0.2)',
              borderColor: 'rgba(75, 192, 192, 1)',
              borderWidth: 1
            },
            {
              label: 'Failure',
              data: chartData.FailureData.split(', ').map(Number),
              backgroundColor: 'rgba(255, 99, 132, 0.2)',
              borderColor: 'rgba(255, 99, 132, 1)',
              borderWidth: 1
            },
            {
              label: 'Aborted',
              data: chartData.AbortedData.split(', ').map(Number),
              backgroundColor: 'rgba(255, 206, 86, 0.2)',
              borderColor: 'rgba(255, 206, 86, 1)',
              borderWidth: 1
            }
          ]
        },
        options: {
          scales: {
            y: {
              beginAtZero: true
            }
          }
        }
      });
    }
  }, [chartData]);

  return (
    <div>
      <canvas id="myChart" ref={chartRef}></canvas>
    </div>
  );
};

export default ChartComponent;
