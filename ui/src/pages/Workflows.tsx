import { useState, useEffect } from 'react';
import type { FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api';
import type { Workflow } from '../api';

export default function Workflows() {
  const navigate = useNavigate();
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [name, setName] = useState('');
  const [yaml, setYaml] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [showUpload, setShowUpload] = useState(false);

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
      setShowUpload(false);
      loadWorkflows();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to upload workflow');
    }
  };

  const handleDelete = async (wf: Workflow) => {
    if (!window.confirm(`Delete workflow "${wf.name}"? This cannot be undone.`)) return;
    setError('');
    try {
      await api.deleteWorkflow(wf.name);
      await loadWorkflows();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete workflow');
    }
  };

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">Workflows</h1>
        <button className="btn btn-primary btn-sm" onClick={() => { setShowUpload(true); setError(''); setSuccess(''); }}>
          Upload Workflow
        </button>
      </div>

      {error && !showUpload && <div className="msg-error">{error}</div>}
      {success && !showUpload && <div className="msg-success">{success}</div>}

      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Source</th>
              <th>Updated</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {workflows.map((wf) => (
              <tr
                key={wf.id}
                className="wf-row"
                onClick={() => navigate(`/workflows/${encodeURIComponent(wf.name)}`)}
              >
                <td className="cell-mono">{wf.name}</td>
                <td>
                  {wf.project_id
                    ? <span className="source-badge source-badge-project">Project</span>
                    : <span className="source-badge source-badge-manual">Manual</span>
                  }
                </td>
                <td className="cell-mono">
                  {new Date(wf.updated_at).toLocaleString()}
                </td>
                <td onClick={(e) => e.stopPropagation()}>
                  <button
                    className="btn btn-sm btn-danger"
                    onClick={() => handleDelete(wf)}
                  >
                    Delete
                  </button>
                </td>
              </tr>
            ))}
            {workflows.length === 0 && (
              <tr>
                <td colSpan={4} className="table-empty">
                  No workflows uploaded yet.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      {showUpload && (
        <div className="modal-backdrop" onClick={() => setShowUpload(false)}>
          <div className="modal-card wf-upload-modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <span className="modal-title">Upload Workflow</span>
              <button className="modal-close" onClick={() => setShowUpload(false)}>&times;</button>
            </div>
            <div className="modal-body">
              {error && <div className="msg-error">{error}</div>}
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
      )}
    </div>
  );
}
