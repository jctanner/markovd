import { useState, useEffect } from 'react';
import type { FormEvent } from 'react';
import { api } from '../api';
import type { Workflow } from '../api';

export default function Workflows() {
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [name, setName] = useState('');
  const [yaml, setYaml] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [selected, setSelected] = useState<Workflow | null>(null);

  const loadWorkflows = async () => {
    try {
      setWorkflows(await api.listWorkflows());
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load workflows');
    }
  };

  useEffect(() => { loadWorkflows(); }, []);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');
    try {
      await api.createWorkflow(name, yaml);
      setSuccess(`Workflow "${name}" uploaded.`);
      setName('');
      setYaml('');
      loadWorkflows();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to upload workflow');
    }
  };

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">Workflows</h1>
      </div>

      <div className="split-layout">
        <div>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Updated</th>
                </tr>
              </thead>
              <tbody>
                {workflows.map((wf) => (
                  <tr
                    key={wf.id}
                    className={`wf-row${selected?.id === wf.id ? ' selected' : ''}`}
                    onClick={() => setSelected(wf)}
                  >
                    <td className="cell-mono">{wf.name}</td>
                    <td className="cell-mono">
                      {new Date(wf.updated_at).toLocaleString()}
                    </td>
                  </tr>
                ))}
                {workflows.length === 0 && (
                  <tr>
                    <td colSpan={2} className="table-empty">
                      No workflows uploaded yet.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>

          {selected && (
            <div style={{ marginTop: 20 }}>
              <div className="section-heading">{selected.name}</div>
              <pre className="yaml-viewer">{selected.yaml}</pre>
            </div>
          )}
        </div>

        <div className="card">
          <div className="card-title">Upload Workflow</div>
          {error && <div className="msg-error">{error}</div>}
          {success && <div className="msg-success">{success}</div>}
          <form onSubmit={handleSubmit}>
            <div className="form-group">
              <label className="form-label">Name</label>
              <input
                type="text"
                className="form-input"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="my-workflow"
                required
              />
            </div>
            <div className="form-group">
              <label className="form-label">YAML Definition</label>
              <textarea
                className="form-textarea"
                value={yaml}
                onChange={(e) => setYaml(e.target.value)}
                placeholder="name: my-workflow&#10;steps:&#10;  - name: step-1&#10;    shell: echo hello"
                required
                rows={14}
              />
            </div>
            <button type="submit" className="btn btn-primary">
              Upload
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
