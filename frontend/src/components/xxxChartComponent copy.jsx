import React, { useEffect, useRef } from 'react';
import { Chart } from 'chart.js/auto';
import 'chartjs-plugin-roughness';

const ChartComponent = ({ data, labels, title, theme, type }) => {
  const chartRef = useRef(null);
  const chartInstanceRef = useRef(null);

  useEffect(() => {
    if (chartInstanceRef.current) {
      chartInstanceRef.current.destroy();
    }

    const borderColor = theme === 'dark' ? 'white' : 'black';
    const colors = {
      SUCCESS: 'green',
      FAILURE: 'red',
      ABORTED: 'gray',
      DURATION: 'blue'
    };

    chartInstanceRef.current = new Chart(chartRef.current, {
      type: type,
      data: {
        labels: labels,
        datasets: data.map(dataset => ({
          label: dataset.label,
          data: dataset.values,
          backgroundColor: colors[dataset.label.toUpperCase()],
          borderColor: borderColor,
          borderWidth: 1,
        }))
      },
      options: {
        plugins: {
          roughness: {
            disabled: false,
            fillStyle: 'hachure',
            fillWeight: 0.8,
            roughness: 1.2,
            hachureGap: 2.8
          },
          legend: {
            display: true,
            position: 'bottom',
            labels: {
              font: {
                family: 'as-virgil'
              }
            }
          }
        },
        scales: {
          x: {
            grid: {
              display: false
            },
            ticks: {
              font: {
                family: 'as-virgil'
              }
            }
          },
          y: {
            grid: {
              display: false
            },
            ticks: {
              font: {
                family: 'as-virgil'
              }
            },
            beginAtZero: true
          }
        }
      }
    });

    return () => {
      if (chartInstanceRef.current) {
        chartInstanceRef.current.destroy();
      }
    };
  }, [data, labels, theme, type]);

  return (
    <div>
      <h3>{title}</h3>
      <canvas ref={chartRef}></canvas>
    </div>
  );
};

export default ChartComponent;
