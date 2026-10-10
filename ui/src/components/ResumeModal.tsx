import { useState } from 'react';
import type { Run } from '../api';

interface Props {
  run: Run | null;
  onClose: () => void;
  onConfirm: (vars: Record<string, string>) => void;
}

interface VarRow {
  key: string;
  value: string;
}

// Resume a paused or failed run from its saved state. A run paused at a gate needs at least one
// var: the gate is evaluated again with it (for example reviewed=true).
export default function ResumeModal({ run, onClose, onConfirm }: Props) {
  const [vars, setVars] = useState<VarRow[]>([{ key: '', value: '' }]);

  if (!run) return null;
  const paused = run.status === 'paused';
  const filled = vars.filter(v => v.key.trim() !== '');

  const updateVar = (i: number, field: 'key' | 'value', val: string) => {
    const updated = [...vars];
    updated[i] = { ...updated[i], [field]: val };
    setVars(updated);
  };

  const handleConfirm = () => {
    const out: Record<string, string> = {};
    for (const v of filled) out[v.key.trim()] = v.value;
    onConfirm(out);
  };

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal-card" onClick={e => e.stopPropagation()} style={{ maxWidth: 560 }}>
        <div className="modal-header">
          <div className="modal-title">Resume {run.run_id}</div>
          <button className="modal-close" onClick={onClose}>&times;</button>
        </div>
        <div className="modal-body">
          <p className="rerun-empty">
            {paused
              ? 'The run is paused at a gate. Set at least one variable; the gate is evaluated again with it.'
              : 'The run failed. It resumes from the failed step; completed steps are kept. Variables are optional.'}
          </p>
          <div className="rerun-section">
            <div className="rerun-section-header">
              <span className="rerun-section-label">Variables</span>
              <button type="button" className="btn btn-ghost btn-sm" onClick={() => setVars([...vars, { key: '', value: '' }])}>+ Add</button>
            </div>
            {vars.map((v, i) => (
              <div key={i} className="var-row">
                <input type="text" className="form-input" placeholder="key" value={v.key}
                  onChange={e => updateVar(i, 'key', e.target.value)} />
                <input type="text" className="form-input" placeholder="value" value={v.value}
                  onChange={e => updateVar(i, 'value', e.target.value)} />
                <button type="button" className="btn btn-danger btn-sm" onClick={() => setVars(vars.filter((_, idx) => idx !== i))}>x</button>
              </div>
            ))}
          </div>
          <div className="rerun-actions">
            <button className="btn btn-ghost" onClick={onClose}>Cancel</button>
            <button className="btn btn-primary" onClick={handleConfirm} disabled={paused && filled.length === 0}>Resume</button>
          </div>
        </div>
      </div>
    </div>
  );
}
