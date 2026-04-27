import { useState, useEffect } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { api } from '../api';
import type { Workflow, DiagramResponse } from '../api';
import WorkflowStructureGraph from '../components/WorkflowStructureGraph';

export default function WorkflowDetail() {
  const { name } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const [wf, setWf] = useState<Workflow | null>(null);
  const [diagram, setDiagram] = useState<DiagramResponse | null>(null);
  const [error, setError] = useState('');
  const [yamlOpen, setYamlOpen] = useState(false);
  const [editing, setEditing] = useState(false);
  const [editYaml, setEditYaml] = useState('');
  const [success, setSuccess] = useState('');

  useEffect(() => {
    if (!name) return;
    api.getWorkflow(name)
      .then(setWf)
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load workflow'));
    api.getWorkflowDiagram(name)
      .then(setDiagram)
      .catch(() => setDiagram(null));
  }, [name]);

  const handleDelete = async () => {
    if (!wf) return;
    if (!window.confirm(`Delete workflow "${wf.name}"? This cannot be undone.`)) return;
    try {
      await api.deleteWorkflow(wf.name);
      navigate('/workflows');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete workflow');
    }
  };

  const startEdit = () => {
    if (!wf || wf.project_id) return;
    setEditYaml(wf.yaml);
    setEditing(true);
    setError('');
    setSuccess('');
  };

  const saveEdit = async () => {
    if (!wf) return;
    setError('');
    try {
      const updated = await api.updateWorkflow(wf.name, editYaml);
      setWf(updated);
      setEditing(false);
      setSuccess(`Workflow "${wf.name}" updated.`);
      api.getWorkflowDiagram(wf.name)
        .then(setDiagram)
        .catch(() => setDiagram(null));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update workflow');
    }
  };

  if (error && !wf) return <div className="msg-error">{error}</div>;
  if (!wf) return <div className="loading-state">Loading workflow...</div>;

  return (
    <div>
      <div className="breadcrumb">
        <Link to="/workflows">Workflows</Link>
        <span className="breadcrumb-sep">/</span>
        <span>{wf.name}</span>
      </div>

      <div className="page-header">
        <h1 className="page-title">{wf.name}</h1>
        <div style={{ display: 'flex', gap: 8 }}>
          {!wf.project_id && !editing && (
            <button className="btn btn-ghost btn-sm" onClick={startEdit}>Edit</button>
          )}
          <button className="btn btn-danger btn-sm" onClick={handleDelete}>Delete</button>
        </div>
      </div>

      {error && <div className="msg-error">{error}</div>}
      {success && <div className="msg-success">{success}</div>}

      {wf.source_path && (
        <div style={{ fontSize: 12, color: 'var(--text-muted)', marginBottom: 12 }}>
          Source: {wf.source_path}
        </div>
      )}

      <div style={{ marginBottom: 16 }}>
        <button
          className="diagram-toggle"
          onClick={() => setYamlOpen(!yamlOpen)}
        >
          <span className="diagram-toggle-chevron">{yamlOpen ? '▾' : '▸'}</span>
          YAML Definition
        </button>
        {yamlOpen && !editing && (
          <pre className="yaml-viewer" style={{ marginTop: 8 }}>{wf.yaml}</pre>
        )}
        {yamlOpen && editing && (
          <div style={{ marginTop: 8 }}>
            <textarea
              className="form-textarea"
              value={editYaml}
              onChange={(e) => setEditYaml(e.target.value)}
              rows={20}
            />
            <div style={{ marginTop: 8, display: 'flex', gap: 8 }}>
              <button className="btn btn-primary btn-sm" onClick={saveEdit}>Save</button>
              <button className="btn btn-ghost btn-sm" onClick={() => setEditing(false)}>Cancel</button>
            </div>
          </div>
        )}
      </div>

      {diagram && diagram.nodes.length > 0 && (
        <WorkflowStructureGraph nodes={diagram.nodes} edges={diagram.edges} />
      )}
    </div>
  );
}
