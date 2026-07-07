import { useState, useEffect, useMemo } from 'react';
import type { FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api';
import type { Workflow, PVCInfo, SecretInfo } from '../api';

function cleanYAMLScalar(value: string): string {
  return value.trim().replace(/^['"]|['"]$/g, '');
}

function workflowNameFromDirectoryFile(content: string): string | null {
  const match = content.match(/^name:\s*([^#\n]+)/m);
  return match ? cleanYAMLScalar(match[1]) : null;
}

function workflowNamesFromFile(content: string): string[] {
  const lines = content.split(/\r?\n/);
  const names: string[] = [];
  let workflowsIndent: number | null = null;

  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) continue;

    const indent = line.length - line.trimStart().length;
    if (workflowsIndent === null) {
      if (trimmed === 'workflows:') workflowsIndent = indent;
      continue;
    }
    if (indent <= workflowsIndent && !trimmed.startsWith('-')) break;

    const match = line.match(/^(\s*)-\s+name:\s*([^#\n]+)/);
    if (!match) continue;
    const itemIndent = match[1].length;
    if (itemIndent === workflowsIndent + 2) {
      names.push(cleanYAMLScalar(match[2]));
    }
  }

  return names;
}

function workflowEntrypointSuggestions(workflow?: Workflow): string[] {
  if (!workflow) return [];
  const names = workflow.definition_kind === 'directory'
    ? workflow.files
        .filter(file => file.path.startsWith('workflows/') && file.path.endsWith('.yaml'))
        .map(file => workflowNameFromDirectoryFile(file.content))
        .filter((name): name is string => Boolean(name))
    : workflowNamesFromFile(workflow.yaml || workflow.files?.[0]?.content || '');
  return Array.from(new Set(names)).sort();
}

export default function TriggerRun() {
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [pvcs, setPvcs] = useState<PVCInfo[]>([]);
  const [secrets, setSecrets] = useState<SecretInfo[]>([]);
  const [selectedWorkflow, setSelectedWorkflow] = useState('');
  const [workflowEntrypoint, setWorkflowEntrypoint] = useState('');
  const [vars, setVars] = useState<{ key: string; value: string }[]>([]);
  const [selectedPVCs, setSelectedPVCs] = useState<Record<string, string>>({});
  const [selectedSecrets, setSelectedSecrets] = useState<Record<string, string>>({});
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [debug, setDebug] = useState(false);
  const [error, setError] = useState('');
  const navigate = useNavigate();
  const selectedWorkflowDef = workflows.find(w => w.name === selectedWorkflow);
  const workflowEntrypoints = useMemo(
    () => workflowEntrypointSuggestions(selectedWorkflowDef),
    [selectedWorkflowDef],
  );
  const selectedPVCCount = Object.keys(selectedPVCs).length;
  const selectedSecretCount = Object.keys(selectedSecrets).length;
  const advancedSummary = [
    pvcs.length > 0 ? `${selectedPVCCount} PVC${selectedPVCCount === 1 ? '' : 's'} selected` : '',
    secrets.length > 0 ? `${selectedSecretCount} secret${selectedSecretCount === 1 ? '' : 's'} selected` : '',
  ].filter(Boolean).join(' / ');

  useEffect(() => {
    api.listWorkflows().then(setWorkflows).catch(() => {});
    api.listPVCs().then(setPvcs).catch(() => {});
    api.listSecrets().then(setSecrets).catch(() => {});
    api.getPreferences().then((prefs) => {
      const pvcDefaults: Record<string, string> = {};
      for (const v of prefs.default_volumes) {
        if (v.name && v.mount_path) pvcDefaults[v.name] = v.mount_path;
      }
      if (Object.keys(pvcDefaults).length > 0) setSelectedPVCs(pvcDefaults);
      const secretDefaults: Record<string, string> = {};
      for (const s of prefs.default_secrets) {
        if (s.name && s.mount_path) secretDefaults[s.name] = s.mount_path;
      }
      if (Object.keys(secretDefaults).length > 0) setSelectedSecrets(secretDefaults);
    }).catch(() => {});
  }, []);

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
      if (name in next) {
        delete next[name];
      } else {
        next[name] = `/mnt/${name}`;
      }
      return next;
    });
  };

  const updateMountPath = (name: string, path: string) => {
    setSelectedPVCs(prev => ({ ...prev, [name]: path }));
  };

  const toggleSecret = (name: string) => {
    setSelectedSecrets(prev => {
      const next = { ...prev };
      if (name in next) {
        delete next[name];
      } else {
        next[name] = `/etc/secrets/${name}`;
      }
      return next;
    });
  };

  const updateSecretMountPath = (name: string, path: string) => {
    setSelectedSecrets(prev => ({ ...prev, [name]: path }));
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    if (!selectedWorkflow) {
      setError('Select a workflow');
      return;
    }
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
    try {
      const run = await api.createRun(selectedWorkflow, varsMap, debug, volsList, secretsList, workflowEntrypoint);
      navigate(`/runs/${run.run_id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to trigger run');
    }
  };

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">Trigger Run</h1>
      </div>

      <div className="trigger-form">
        {error && <div className="msg-error">{error}</div>}

        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label className="form-label">Workflow</label>
            <select
              className="form-select"
              value={selectedWorkflow}
              onChange={(e) => {
                setSelectedWorkflow(e.target.value);
                setWorkflowEntrypoint('');
              }}
              required
            >
              <option value="">Select a workflow...</option>
              {workflows.map((wf) => (
                <option key={wf.name} value={wf.name}>{wf.name}</option>
              ))}
            </select>
            {selectedWorkflowDef && (
              <div style={{ marginTop: 8, display: 'flex', gap: 8, alignItems: 'center' }}>
                <span className="source-badge source-badge-manual">
                  {selectedWorkflowDef.definition_kind === 'directory' ? 'Directory' : 'File'}
                </span>
                <span className={selectedWorkflowDef.project_id ? 'source-badge source-badge-project' : 'source-badge source-badge-manual'}>
                  {selectedWorkflowDef.project_id ? 'Project' : 'Manual'}
                </span>
                {selectedWorkflowDef.source_path && (
                  <span className="cell-mono" style={{ color: 'var(--text-muted)', fontSize: 12 }}>
                    {selectedWorkflowDef.source_path}
                  </span>
                )}
              </div>
            )}
          </div>

          <div className="form-group">
            <label className="form-label">Workflow Entrypoint</label>
            <input
              type="text"
              className="form-input"
              value={workflowEntrypoint}
              onChange={(e) => setWorkflowEntrypoint(e.target.value)}
              placeholder="Use definition entrypoint"
              list={workflowEntrypoints.length > 0 ? 'workflow-entrypoint-options' : undefined}
            />
            {workflowEntrypoints.length > 0 && (
              <datalist id="workflow-entrypoint-options">
                {workflowEntrypoints.map((name) => (
                  <option key={name} value={name} />
                ))}
              </datalist>
            )}
          </div>

          <div className="form-group">
            <div className="vars-header">
              <label className="form-label" style={{ margin: 0 }}>Variables</label>
              <button type="button" className="btn btn-ghost btn-sm" onClick={addVar}>
                + Add
              </button>
            </div>
            {vars.map((v, i) => (
              <div key={i} className="var-row">
                <input
                  type="text"
                  className="form-input"
                  placeholder="key"
                  value={v.key}
                  onChange={(e) => updateVar(i, 'key', e.target.value)}
                />
                <input
                  type="text"
                  className="form-input"
                  placeholder="value"
                  value={v.value}
                  onChange={(e) => updateVar(i, 'value', e.target.value)}
                />
                <button
                  type="button"
                  className="btn btn-danger btn-sm"
                  onClick={() => removeVar(i)}
                >
                  x
                </button>
              </div>
            ))}
          </div>

          {(pvcs.length > 0 || secrets.length > 0) && (
            <div className="form-group trigger-advanced">
              <button
                type="button"
                className="trigger-advanced-toggle"
                aria-expanded={advancedOpen}
                onClick={() => setAdvancedOpen(open => !open)}
              >
                <span className="trigger-advanced-title">
                  <span className={`trigger-advanced-caret${advancedOpen ? ' open' : ''}`}>›</span>
                  Advanced
                </span>
                {advancedSummary && <span className="trigger-advanced-summary">{advancedSummary}</span>}
              </button>

              {advancedOpen && (
                <div className="trigger-advanced-content">
                  {pvcs.length > 0 && (
                    <div className="trigger-advanced-section">
                      <label className="form-label">PVC Volumes</label>
                      <div className="pvc-list">
                        {pvcs.map((p) => {
                          const checked = p.name in selectedPVCs;
                          return (
                            <div key={p.name} className={`pvc-item${checked ? ' selected' : ''}`}>
                              <label className="pvc-item-label">
                                <input
                                  type="checkbox"
                                  checked={checked}
                                  onChange={() => togglePVC(p.name)}
                                />
                                <span className="pvc-item-name">{p.name}</span>
                                <span className={`badge badge-${p.status === 'Bound' ? 'completed' : 'pending'}`} style={{ marginLeft: 8 }}>
                                  {p.status}
                                </span>
                              </label>
                              {checked && (
                                <input
                                  type="text"
                                  className="form-input pvc-mount-input"
                                  value={selectedPVCs[p.name]}
                                  onChange={(e) => updateMountPath(p.name, e.target.value)}
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
                    <div className="trigger-advanced-section">
                      <label className="form-label">Secret Volumes</label>
                      <div className="secret-list">
                        {secrets.map((s) => {
                          const checked = s.name in selectedSecrets;
                          return (
                            <div key={s.name} className={`secret-item${checked ? ' selected' : ''}`}>
                              <label className="secret-item-label">
                                <input
                                  type="checkbox"
                                  checked={checked}
                                  onChange={() => toggleSecret(s.name)}
                                />
                                <span className="secret-item-name">{s.name}</span>
                                <span className="badge badge-pending" style={{ marginLeft: 8 }}>
                                  {s.type}
                                </span>
                              </label>
                              {checked && (
                                <input
                                  type="text"
                                  className="form-input secret-mount-input"
                                  value={selectedSecrets[s.name]}
                                  onChange={(e) => updateSecretMountPath(s.name, e.target.value)}
                                  placeholder="/etc/secrets/..."
                                />
                              )}
                            </div>
                          );
                        })}
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
          )}

          <div className="form-group">
            <label className="form-checkbox">
              <input
                type="checkbox"
                checked={debug}
                onChange={(e) => setDebug(e.target.checked)}
              />
              Enable debug mode
            </label>
          </div>

          <button type="submit" className="btn btn-primary">
            Trigger Run
          </button>
        </form>
      </div>
    </div>
  );
}
