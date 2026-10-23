import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import ActionDialog from './ActionDialog';

beforeAll(() => { HTMLDialogElement.prototype.showModal = jest.fn(); });
const action = { phase: 'confirm', host: 'test-host', label: 'Patch', confirmation: 'Install updates?' };
test('confirmation requires explicit action and supports cancellation', () => {
  const confirm = jest.fn(), close = jest.fn();
  render(<ActionDialog action={action} onConfirm={confirm} onClose={close} />);
  expect(confirm).not.toHaveBeenCalled();
  fireEvent.click(screen.getByText('Run action'));
  expect(confirm).toHaveBeenCalledTimes(1);
  fireEvent.click(screen.getByText('Cancel'));
  expect(close).toHaveBeenCalledTimes(1);
});
test('result offers a check rerun and displays feedback', () => {
  const rerun = jest.fn();
  render(<ActionDialog action={{ ...action, phase: 'result', status: 'completed', output: 'Done' }} onRerun={rerun} feedback="Check refreshed" />);
  fireEvent.click(screen.getByText('Re-run check'));
  expect(rerun).toHaveBeenCalledTimes(1);
  expect(screen.getByText('Check refreshed')).toBeInTheDocument();
});
test('running action prevents closing', () => {
  render(<ActionDialog action={{ ...action, phase: 'running' }} busy />);
  expect(screen.getByText('Close')).toBeDisabled();
});
