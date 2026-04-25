import { useState, useEffect } from 'react';
import { api } from '../api';
import type { Run, PVCInfo, SecretInfo } from '../api';

interface Props {
  run: Run | null;
  onClose: () => void;
  onConfirm: (
    vars: Record<string, string>,
    volumes: { name: string; pvc: string; mount_path: string }[],
    secretVolumes: { name: string; secret: string; mount_path: string }[],
  ) => void;
}

interface VarRow {
  key: string;
  value: string;
}

function parseVolumes(json: string): Record<string, string> {
  try {
    const arr = JSON.parse(json || '[]');
    if (!Array.isArray(arr)) return {};
    const map: Record<string, string> = {};
    for (const v of arr) {
      if (v.name && v.mount_path) map[v.name] = v.mount_path;
    }
    return map;
  } catch {
    return {};
  }
}

export default function RerunModal({ run, onClose, onConfirm }: Props) {
  const [vars, setVars] = useState<VarRow[]>([]);
  const [pvcs, setPvcs] = useState<PVCInfo[]>([]);
  const [secrets, setSecrets] = useState<SecretInfo[]>([]);
  const [selectedPVCs, setSelectedPVCs] = useState<Record<string, string>>({});
  const [selectedSecrets, setSelectedSecrets] = useState<Record<string, string>>({});

  useEffect(() => {
    if (!run) return;
    try {
      const parsed = JSON.parse(run.vars_json || '{}');
      const entries = Object.entries(parsed);
      setVars(entries.map(([key, value]) => ({ key, value: String(value) })));
    } catch {
      setVars([]);
    }
    setSelectedPVCs(parseVolumes(run.volumes_json));
    setSelectedSecrets(parseVolumes(run.secret_volumes_json));
    api.listPVCs().then(setPvcs).catch(() => {});
    api.listSecrets().then(setSecrets).catch(() => {});
  }, [run]);

  if (!run) return null;

  const addVar = () => setVars([...vars, { key: '', value: '' }]);
  const removeVar = (i: number) => setVars(vars.filter((_, idx) => idx !== i));
  const updateVar = (i: number, field: 'key' | 'value', val: string) => {
    const updated = [...vars];
    updated[i][field] = val;
    setVars(updated);
  };

  const togglePVC = (name: string) => {
    setSelectedPVCs(prev => {
      const next = { ...prev };
      if (name in next) delete next[name];
      else next[name] = `/mnt/${name}`;
      return next;
    });
  };

  const toggleSecret = (name: string) => {
    setSelectedSecrets(prev => {
      const next = { ...prev };
      if (name in next) delete next[name];
      else next[name] = `/etc/secrets/${name}`;
      return next;
    });
  };

  const handleConfirm = () => {
    const varsMap: Record<string, string> = {};
    for (const v of vars) {
      if (v.key) varsMap[v.key] = v.value;
    }
    const volsList = Object.entries(selectedPVCs)
      .filter(([, path]) => path)
      .map(([name, path]) => ({ name, pvc: name, mount_path: path }));
    const secretsList = Object.entries(selectedSecrets)
      .filter(([, path]) => path)
      .map(([name, path]) => ({ name, secret: name, mount_path: path }));
    onConfirm(varsMap, volsList, secretsList);
  };

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal-card" onClick={e => e.stopPropagation()} style={{ maxWidth: 560 }}>
        <div className="modal-header">
          <div>
            <div className="modal-title">Re-run</div>
            <div className="rerun-workflow-name">{run.workflow_name}</div>
          </div>
          <button className="modal-close" onClick={onClose}>&times;</button>
        </div>
        <div className="modal-body">
          <div className="rerun-section">
            <div className="rerun-section-header">
              <span className="rerun-section-label">Variables</span>
              <button type="button" className="btn btn-ghost btn-sm" onClick={addVar}>+ Add</button>
            </div>
            {vars.map((v, i) => (
              <div key={i} className="var-row">
                <input
                  type="text"
                  className="form-input"
                  placeholder="key"
                  value={v.key}
                  onChange={e => updateVar(i, 'key', e.target.value)}
                />
                <input
                  type="text"
                  className="form-input"
                  placeholder="value"
                  value={v.value}
                  onChange={e => updateVar(i, 'value', e.target.value)}
                />
                <button type="button" className="btn btn-danger btn-sm" onClick={() => removeVar(i)}>x</button>
              </div>
            ))}
            {vars.length === 0 && <div className="rerun-empty">No variables</div>}
          </div>

          {pvcs.length > 0 && (
            <div className="rerun-section">
              <div className="rerun-section-label">PVC Volumes</div>
              <div className="pvc-list">
                {pvcs.map(p => {
                  const checked = p.name in selectedPVCs;
                  return (
                    <div key={p.name} className={`pvc-item${checked ? ' selected' : ''}`}>
                      <label className="pvc-item-label">
                        <input type="checkbox" checked={checked} onChange={() => togglePVC(p.name)} />
                        <span className="pvc-item-name">{p.name}</span>
                      </label>
                      {checked && (
                        <input
                          type="text"
                          className="form-input pvc-mount-input"
                          value={selectedPVCs[p.name]}
                          onChange={e => setSelectedPVCs(prev => ({ ...prev, [p.name]: e.target.value }))}
                          placeholder="/mnt/..."
                        />
                      )}
                    </div>
                  );
                })}
              </div>
            </div>
          )}

          {secrets.length > 0 && (
            <div className="rerun-section">
              <div className="rerun-section-label">Secret Volumes</div>
              <div className="secret-list">
                {secrets.map(s => {
                  const checked = s.name in selectedSecrets;
                  return (
                    <div key={s.name} className={`secret-item${checked ? ' selected' : ''}`}>
                      <label className="secret-item-label">
                        <input type="checkbox" checked={checked} onChange={() => toggleSecret(s.name)} />
                        <span className="secret-item-name">{s.name}</span>
                      </label>
                      {checked && (
                        <input
                          type="text"
                          className="form-input secret-mount-input"
                          value={selectedSecrets[s.name]}
                          onChange={e => setSelectedSecrets(prev => ({ ...prev, [s.name]: e.target.value }))}
                          placeholder="/etc/secrets/..."
                        />
                      )}
                    </div>
                  );
                })}
              </div>
            </div>
          )}

          <div className="rerun-actions">
            <button className="btn btn-ghost" onClick={onClose}>Cancel</button>
            <button className="btn btn-primary" onClick={handleConfirm}>Re-run</button>
          </div>
        </div>
      </div>
    </div>
  );
}
