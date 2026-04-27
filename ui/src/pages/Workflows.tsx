import { useState, useEffect } from 'react';
import type { FormEvent } from 'react';
import { api } from '../api';
import type { Workflow } from '../api';
import MermaidDiagram from '../components/MermaidDiagram';

export default function Workflows() {
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [name, setName] = useState('');
  const [yaml, setYaml] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [selected, setSelected] = useState<Workflow | null>(null);
  const [editing, setEditing] = useState(false);
  const [editYaml, setEditYaml] = useState('');
  const [diagram, setDiagram] = useState('');
  const [diagramOpen, setDiagramOpen] = useState(true);

  const loadWorkflows = async () => {
    try {
      setWorkflows(await api.listWorkflows());
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load workflows');
    }
  };

  useEffect(() => { loadWorkflows(); }, []);

  useEffect(() => {
    if (!selected) { setDiagram(''); return; }
    api.getWorkflowDiagram(selected.name)
      .then((d) => setDiagram(d.mermaid))
      .catch(() => setDiagram(''));
  }, [selected?.name]);

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

  const handleDelete = async (wf: Workflow) => {
    if (!window.confirm(`Delete workflow "${wf.name}"? This cannot be undone.`)) return;
    setError('');
    try {
      await api.deleteWorkflow(wf.name);
      if (selected?.id === wf.id) {
        setSelected(null);
        setEditing(false);
      }
      await loadWorkflows();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete workflow');
    }
  };

  const startEdit = () => {
    if (!selected || selected.project_id) return;
    setEditYaml(selected.yaml);
    setEditing(true);
    setError('');
    setSuccess('');
  };

  const cancelEdit = () => {
    setEditing(false);
  };

  const saveEdit = async () => {
    if (!selected) return;
    setError('');
    try {
      const updated = await api.updateWorkflow(selected.name, editYaml);
      setSelected(updated);
      setEditing(false);
      setSuccess(`Workflow "${selected.name}" updated.`);
      await loadWorkflows();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update workflow');
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
                  <th>Source</th>
                  <th>Updated</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {workflows.map((wf) => (
                  <tr
                    key={wf.id}
                    className={`wf-row${selected?.id === wf.id ? ' selected' : ''}`}
                    onClick={() => { setSelected(wf); setEditing(false); }}
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
                      {!wf.project_id && (
                        <button
                          className="btn btn-sm btn-ghost"
                          onClick={() => { setSelected(wf); startEdit(); }}
                        >
                          Edit
                        </button>
                      )}
                      {' '}
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

          {selected && !editing && (
            <div style={{ marginTop: 20 }}>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <div className="section-heading" style={{ margin: 0 }}>{selected.name}</div>
                {!selected.project_id && (
                  <button className="btn btn-sm btn-ghost" onClick={startEdit}>Edit</button>
                )}
              </div>
              {selected.source_path && (
                <div style={{ fontSize: 12, color: 'var(--text-muted)', marginBottom: 8 }}>
                  Source: {selected.source_path}
                </div>
              )}
              <pre className="yaml-viewer">{selected.yaml}</pre>

              {diagram && (
                <div style={{ marginTop: 16 }}>
                  <button
                    className="diagram-toggle"
                    onClick={() => setDiagramOpen(!diagramOpen)}
                  >
                    <span className="diagram-toggle-chevron">{diagramOpen ? '▾' : '▸'}</span>
                    Diagram
                  </button>
                  {diagramOpen && (
                    <div className="diagram-container">
                      <MermaidDiagram code={diagram} />
                    </div>
                  )}
                </div>
              )}
            </div>
          )}

          {selected && editing && (
            <div style={{ marginTop: 20 }}>
              <div className="section-heading">{selected.name}</div>
              <textarea
                className="form-textarea"
                value={editYaml}
                onChange={(e) => setEditYaml(e.target.value)}
                rows={18}
              />
              <div style={{ marginTop: 8, display: 'flex', gap: 8 }}>
                <button className="btn btn-primary btn-sm" onClick={saveEdit}>Save</button>
                <button className="btn btn-ghost btn-sm" onClick={cancelEdit}>Cancel</button>
              </div>
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
