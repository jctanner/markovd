import { useState, useEffect } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { api } from '../api';
import type { Workflow, DiagramResponse, WorkflowDefinitionFile } from '../api';
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
  const [editFiles, setEditFiles] = useState<WorkflowDefinitionFile[]>([]);
  const [selectedFile, setSelectedFile] = useState(0);
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
    setEditFiles(wf.files?.length ? wf.files : [{ path: 'workflow.yaml', content: wf.yaml }]);
    setSelectedFile(0);
    setEditing(true);
    setError('');
    setSuccess('');
  };

  const saveEdit = async () => {
    if (!wf) return;
    setError('');
    try {
      const updated = wf.definition_kind === 'directory'
        ? await api.updateWorkflowDefinition(wf.name, 'directory', editFiles)
        : await api.updateWorkflow(wf.name, editYaml);
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

  const updateFile = (idx: number, field: 'path' | 'content', value: string) => {
    setEditFiles(prev => prev.map((f, i) => i === idx ? { ...f, [field]: value } : f));
  };

  const addFile = () => {
    setEditFiles(prev => [...prev, { path: `workflows/file-${prev.length}.yaml`, content: 'name: new_workflow\nsteps: []\n' }]);
    setSelectedFile(editFiles.length);
  };

  const removeFile = (idx: number) => {
    setEditFiles(prev => prev.filter((_, i) => i !== idx));
    setSelectedFile(0);
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
      <div style={{ fontSize: 12, color: 'var(--text-muted)', marginBottom: 12 }}>
        Type: {wf.definition_kind === 'directory' ? 'Directory' : 'File'}
        {wf.files?.length ? ` · ${wf.files.length} file${wf.files.length === 1 ? '' : 's'}` : ''}
      </div>

      <div style={{ marginBottom: 16 }}>
        <button
          className="diagram-toggle"
          onClick={() => setYamlOpen(!yamlOpen)}
        >
          <span className="diagram-toggle-chevron">{yamlOpen ? '▾' : '▸'}</span>
          Definition
        </button>
        {yamlOpen && !editing && wf.definition_kind !== 'directory' && (
          <pre className="yaml-viewer" style={{ marginTop: 8 }}>{wf.yaml}</pre>
        )}
        {yamlOpen && !editing && wf.definition_kind === 'directory' && (
          <div className="definition-editor" style={{ marginTop: 8 }}>
            <div className="file-tree">
              {wf.files.map((f, idx) => (
                <div key={`${f.path}-${idx}`} className={`file-tree-item${selectedFile === idx ? ' imported' : ''}`}>
                  <button type="button" className="btn btn-ghost btn-sm" onClick={() => setSelectedFile(idx)}>
                    {f.path}
                  </button>
                </div>
              ))}
            </div>
            {wf.files[selectedFile] && (
              <pre className="yaml-viewer">{wf.files[selectedFile].content}</pre>
            )}
          </div>
        )}
        {yamlOpen && editing && (
          <div style={{ marginTop: 8 }}>
            {wf.definition_kind !== 'directory' ? (
              <textarea
                className="form-textarea"
                value={editYaml}
                onChange={(e) => setEditYaml(e.target.value)}
                rows={20}
              />
            ) : (
              <div className="definition-editor">
                <div className="file-tree">
                  {editFiles.map((f, idx) => (
                    <div key={`${f.path}-${idx}`} className={`file-tree-item${selectedFile === idx ? ' imported' : ''}`}>
                      <button type="button" className="btn btn-ghost btn-sm" onClick={() => setSelectedFile(idx)}>
                        {f.path || '(unnamed)'}
                      </button>
                      <button type="button" className="btn btn-danger btn-sm" onClick={() => removeFile(idx)}>
                        x
                      </button>
                    </div>
                  ))}
                </div>
                <button type="button" className="btn btn-ghost btn-sm" onClick={addFile} style={{ marginBottom: 8 }}>
                  Add file
                </button>
                {editFiles[selectedFile] && (
                  <>
                    <input
                      className="form-input"
                      value={editFiles[selectedFile].path}
                      onChange={(e) => updateFile(selectedFile, 'path', e.target.value)}
                    />
                    <textarea
                      className="form-textarea"
                      value={editFiles[selectedFile].content}
                      onChange={(e) => updateFile(selectedFile, 'content', e.target.value)}
                      rows={16}
                    />
                  </>
                )}
              </div>
            )}
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
