import React, { useEffect, useRef } from 'react';

export default function ActionDialog({ action, onClose, onConfirm, onRerun, busy, feedback }) {
  const dialog = useRef(null);
  useEffect(() => {
    const previous = document.activeElement;
    dialog.current?.showModal();
    return () => previous?.focus();
  }, []);
  return (
    <dialog ref={dialog} className="action-dialog" aria-labelledby="action-dialog-title" onCancel={(event) => { event.preventDefault(); if (!busy) onClose(); }}>
      <div className="modal-card" style={{ margin: 0, width: '100%' }}>
        <header className="modal-card-head">
          <h2 className="modal-card-title" id="action-dialog-title">{action.phase === 'confirm' ? 'Confirm action' : action.phase === 'running' ? 'Action running' : 'Action result'}</h2>
          <button className="delete" aria-label="Close" disabled={busy} onClick={onClose} />
        </header>
        <section className="modal-card-body">
          <p><strong>{action.label}</strong> — {action.host}</p>
          {action.phase === 'confirm' ? <p className="mt-3">{action.confirmation}</p> : <>
            <p className="mt-3" role="status">{action.phase === 'running' ? 'The action is running. Please wait.' : action.status === 'completed' ? 'Action completed.' : 'The action could not be completed.'}</p>
            {action.output && <pre className="mt-3" style={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere' }}>{action.output}</pre>}
            {action.phase === 'result' && <p className="mt-3">Run the check again to refresh the result for this host.</p>}
          </>}
          {feedback && <div className="notification is-light mt-3" role="status">{feedback}</div>}
        </section>
        <footer className="modal-card-foot">
          {action.phase === 'confirm' && <button className="button is-warning" onClick={onConfirm}>Run action</button>}
          {action.phase === 'result' && <button className={`button is-primary ${busy ? 'is-loading' : ''}`} disabled={busy} onClick={onRerun}>Re-run check</button>}
          <button className="button" disabled={busy} onClick={onClose}>{action.phase === 'confirm' ? 'Cancel' : 'Close'}</button>
        </footer>
      </div>
    </dialog>
  );
}
