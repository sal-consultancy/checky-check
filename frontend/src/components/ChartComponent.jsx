import React, { useEffect, useRef } from 'react';
import { Chart } from 'chart.js/auto';
import 'chartjs-plugin-roughness';

const ChartComponent = ({ data, labels, title, theme, type, colors }) => {
  const chartRef = useRef(null);
  const chartInstanceRef = useRef(null);

  useEffect(() => {
    if (chartInstanceRef.current) {
      chartInstanceRef.current.destroy();
    }

    const borderColor = theme === 'dark' ? 'white' : 'black';

    const backgroundColors = data.map((value, index) => {
      return value.failed > 0 ? colors.failed[index % colors.failed.length] : colors.passed[index % colors.passed.length];
    });

    const datasets = [
      {
        label: 'Values',
        data: data.map(d => d.value),
        backgroundColor: backgroundColors,
        borderColor: borderColor,
        borderWidth: type === 'pie' ? '0.45' : '1' ,
      },
    ];

    chartInstanceRef.current = new Chart(chartRef.current, {
      type: type,
      data: {
        labels: labels,
        datasets: datasets,
      },
      options: {
        plugins: {
          roughness: {
            disabled: false,
            fillStyle: 'hachure',
            fillWeight: 0.8,
            roughness: 1.2,
            hachureGap: 3.8,
          },
          legend: {
            display: type === 'pie',
            position: 'bottom',
            labels: {
              font: {
                family: 'as-virgil',
              },
            },
          },
        },
        scales: {
          x: {
            display: type !== 'pie',
            grid: {
              display: false,
            },
            ticks: {
              font: {
                family: 'as-virgil',
              },
            },
          },
          y: {
            display: type !== 'pie',
            grid: {
              display: false,
            },
            ticks: {
              font: {
                family: 'as-virgil',
              },
            },
            beginAtZero: true,
          },
        },
      },
    });

    return () => {
      if (chartInstanceRef.current) {
        chartInstanceRef.current.destroy();
      }
    };
  }, [data, labels, theme, type, colors]);

  return (
    <div style={{ width: type === 'pie' ? '45%' : '100%' }}>
      <h3 className='write'>{title}</h3>
      <canvas ref={chartRef}></canvas>
    </div>
  );
};

export default ChartComponent;
