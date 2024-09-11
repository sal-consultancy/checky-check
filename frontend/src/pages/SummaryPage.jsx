import React, { useRef, useEffect } from 'react';
import { Chart } from 'chart.js/auto';
import 'chartjs-plugin-roughness';

const CheckReport = ({ results, checks, theme }) => {
  const summary = Object.keys(checks).reduce((acc, checkName) => {
    acc[checkName] = { passed: 0, failed: 0, details: [] };

    Object.keys(results).forEach(host => {
      if (results[host][checkName]) {
        const result = results[host][checkName];
        if (result.status === 'passed') {
          acc[checkName].passed += 1;
        } else {
          acc[checkName].failed += 1;
        }
        acc[checkName].details.push({ host, ...result });
      }
    });

    return acc;
  }, {});

  //const allHosts = Object.keys(results).map(host => {
   Object.keys(results).map(host => {
      const hostResults = results[host];
    return {
      host,
      results: Object.keys(hostResults).map(checkName => ({
        checkName,
        status: hostResults[checkName].status,
        value: hostResults[checkName].value,
        timestamp: hostResults[checkName].timestamp
      }))
    };
  });

  const passedCount = Object.keys(summary).reduce((acc, checkName) => acc + summary[checkName].passed, 0);
  const failedCount = Object.keys(summary).reduce((acc, checkName) => acc + summary[checkName].failed, 0);

  const pieChartRef = useRef(null);
  const barChartRef = useRef(null);

  useEffect(() => {
    const theme = localStorage.getItem('theme');
    const borderColor = theme === 'dark' ? 'white' : 'black';

    const pieChart = new Chart(pieChartRef.current, {
      type: 'pie',
      data: {
        labels: ['Passed', 'Failed'],
        datasets: [
          {
            label: '# of Tests',
            data: [passedCount, failedCount],
            backgroundColor: ['green', 'red'],
            borderColor: borderColor,
            borderWidth: 0.4
          }
        ]
      },
      options: {
        plugins: {
          roughness: {
            disabled: false,
            fillStyle: 'hachure',
            fillWeight: 0.8,
            roughness: 1.2,
            hachureGap: 3.8
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
        }
      }
    });

    return () => {
      pieChart.destroy();
    };
  }, [passedCount, failedCount]);

  useEffect(() => {
    if (failedCount === 0) return;

    const theme = localStorage.getItem('theme');
    const borderColor = theme === 'dark' ? 'white' : 'black';

    const checksWithFailures = Object.keys(summary).filter(checkName => summary[checkName].failed > 0);
    const failedChecks = checksWithFailures.map(checkName => ({
      checkName,
      passed: summary[checkName].passed,
      failed: summary[checkName].failed
    }));

    const barChart = new Chart(barChartRef.current, {
      type: 'bar',
      data: {
        labels: failedChecks.map(item => checks[item.checkName].title),
        datasets: [
          {
            label: '# of Failed Tests',
            data: failedChecks.map(item => item.failed),
            backgroundColor: 'red',
            borderColor: borderColor,
            borderWidth: 0.8
          },
          {
            label: '# of Passed Tests',
            data: failedChecks.map(item => item.passed),
            backgroundColor: 'green',
            borderColor: borderColor,
            borderWidth: 0.8
          }
        ]
      },
      options: {
        plugins: {
          roughness: {
            disabled: false,
            fillStyle: 'hachure',
            fillWeight: 0.8,
            roughness: 1.2,
            hachureGap: 3.8
          },
          legend: {
            display: true,
            position: 'bottom',
            labels: {
              font: {
                family: 'as-virgil'
              }
            }
          },
          tooltip: {
            callbacks: {
              label: function (context) {
                return `${context.dataset.label}: ${context.raw}`;
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
            beginAtZero: true,
            grid: {
              display: false
            },
            ticks: {
              stepSize: 1,
              callback: function (value) {
                if (Number.isInteger(value)) {
                  return value;
                }
              },
              font: {
                family: 'as-virgil'
              }
            }
          }
        }
      }
    });

    return () => {
      barChart.destroy();
    };
  }, [summary, checks, failedCount]);

  return (
    <div className="check-report">
      {failedCount > 0 ? (
        <>
          <h5 className='is-size-5 write py-5'>Failed Tests Overview</h5>
          <div className="bar-chart-container" style={{ width: '100%', margin: '0 auto' }}>
            <canvas ref={barChartRef}></canvas>
          </div>
          <hr className="separator" />
          <h5 className='is-size-5 write py-5'>Passed vs Failed Checks</h5>
          <div className="pie-chart-container" style={{ width: '50%', margin: '0 auto' }}>
            <canvas ref={pieChartRef}></canvas>
          </div>
        </>
      ) : (
        <div style={{ textAlign: 'center', margin: '20px 0' }}>
          <h5 className='is-size-5 write py-5'>All checks passed successfully!</h5>
          <h3 className='is-size-2 write py-5'> 🎉 🙌</h3>
        </div>
      )}
    </div>
  );
};

export default CheckReport;
