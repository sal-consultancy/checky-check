import React, { useEffect, useRef } from 'react';
import { Chart } from 'chart.js/auto';

const DEFAULT_CHART_COLORS = {
  failed: ['#a44b4b'],
  passed: ['#6aa36d'],
};

const DEFAULT_BORDER_COLORS = {
  failed: ['#8f3c3c'],
  passed: ['#4d8650'],
};

const ChartComponent = ({ data, labels, title, theme, type, colors }) => {
  const chartRef = useRef(null);
  const chartInstanceRef = useRef(null);

  useEffect(() => {
    if (chartInstanceRef.current) {
      chartInstanceRef.current.destroy();
    }

    const theme = localStorage.getItem('theme');

    const borderColor = theme === 'dark' ? 'white' : 'black';
    const palette = {
      failed: colors?.failed?.length ? colors.failed : DEFAULT_CHART_COLORS.failed,
      passed: colors?.passed?.length ? colors.passed : DEFAULT_CHART_COLORS.passed,
    };

    const backgroundColors = data.map((value, index) => {
      return value.failed > 0
        ? palette.failed[index % palette.failed.length]
        : palette.passed[index % palette.passed.length];
    });

    const borderColors = data.map((value, index) => {
      return value.failed > 0
        ? DEFAULT_BORDER_COLORS.failed[index % DEFAULT_BORDER_COLORS.failed.length]
        : DEFAULT_BORDER_COLORS.passed[index % DEFAULT_BORDER_COLORS.passed.length];
    });

    const datasets = [
      {
        label: 'Values',
        data: data.map(d => d.value),
        backgroundColor: backgroundColors,
        borderColor: type === 'pie' ? borderColor : borderColors,
        borderWidth: type === 'pie' ? '0.4' : '0.8' ,
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
          legend: {
            display: type === 'pie',
            position: 'right',
          },
        },
        scales: {
          x: {
            display: type !== 'pie',
            grid: {
              display: false,
            },
          },
          y: {
            display: type !== 'pie',
            grid: {
              display: false,
            },
            ticks: {
              stepSize: 1,
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
    <div className={`report-chart-shell ${type === 'pie' ? 'is-pie' : 'is-standard'}`}>
      <h3 className='write'>{title}</h3>
      <canvas ref={chartRef}></canvas>
    </div>
  );
};

export default ChartComponent;
