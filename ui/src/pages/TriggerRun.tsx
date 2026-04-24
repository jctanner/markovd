import { useState, useEffect } from 'react';
import type { FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api';
import type { Workflow } from '../api';

export default function TriggerRun() {
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [selectedWorkflow, setSelectedWorkflow] = useState('');
  const [vars, setVars] = useState<{ key: string; value: string }[]>([]);
  const [debug, setDebug] = useState(false);
  const [error, setError] = useState('');
  const navigate = useNavigate();

  useEffect(() => {
    api.listWorkflows().then(setWorkflows).catch(() => {});
  }, []);

  const addVar = () => setVars([...vars, { key: '', value: '' }]);
  const removeVar = (i: number) => setVars(vars.filter((_, idx) => idx !== i));
  const updateVar = (i: number, field: 'key' | 'value', val: string) => {
    const updated = [...vars];
    updated[i][field] = val;
    setVars(updated);
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
    try {
      const run = await api.createRun(selectedWorkflow, varsMap, debug);
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
              onChange={(e) => setSelectedWorkflow(e.target.value)}
              required
            >
              <option value="">Select a workflow...</option>
              {workflows.map((wf) => (
                <option key={wf.name} value={wf.name}>{wf.name}</option>
              ))}
            </select>
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
