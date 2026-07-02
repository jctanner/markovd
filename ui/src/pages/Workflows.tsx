import { useState, useEffect } from 'react';
import type { FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { api } from '../api';
import type { Workflow, WorkflowDefinitionFile } from '../api';

const defaultDirectoryFiles: WorkflowDefinitionFile[] = [
  { path: 'meta.yaml', content: 'entrypoint: main\nforks: 2\n' },
  { path: 'vars.yaml', content: '{}\n' },
  { path: 'rules.yaml', content: '[]\n' },
  { path: 'step_types.yaml', content: '{}\n' },
  { path: 'workflows/main.yaml', content: 'name: main\nsteps: []\n' },
];

export default function Workflows() {
  const navigate = useNavigate();
  const [workflows, setWorkflows] = useState<Workflow[]>([]);
  const [name, setName] = useState('');
  const [yaml, setYaml] = useState('');
  const [definitionKind, setDefinitionKind] = useState<'file' | 'directory'>('file');
  const [files, setFiles] = useState<WorkflowDefinitionFile[]>(defaultDirectoryFiles);
  const [selectedFile, setSelectedFile] = useState(0);
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
      if (definitionKind === 'file') {
        await api.createWorkflow(name, yaml);
      } else {
        await api.createWorkflowDefinition(name, 'directory', files);
      }
      setSuccess(`Workflow "${name}" uploaded.`);
      setName('');
      setYaml('');
      setFiles(defaultDirectoryFiles);
      setSelectedFile(0);
      setDefinitionKind('file');
      setShowUpload(false);
      loadWorkflows();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to upload workflow');
    }
  };

  const updateFile = (idx: number, field: 'path' | 'content', value: string) => {
    setFiles(prev => prev.map((f, i) => i === idx ? { ...f, [field]: value } : f));
  };

  const addFile = () => {
    setFiles(prev => [...prev, { path: `workflows/file-${prev.length}.yaml`, content: 'name: new_workflow\nsteps: []\n' }]);
    setSelectedFile(files.length);
  };

  const removeFile = (idx: number) => {
    setFiles(prev => prev.filter((_, i) => i !== idx));
    setSelectedFile(0);
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
              <th>Type</th>
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
                  <span className="source-badge source-badge-manual">
                    {wf.definition_kind === 'directory' ? 'Directory' : 'File'}
                  </span>
                </td>
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
                <td colSpan={5} className="table-empty">
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
                  <div className="tabs" style={{ marginBottom: 8 }}>
                    <button
                      type="button"
                      className={`tab-btn${definitionKind === 'file' ? ' active' : ''}`}
                      onClick={() => setDefinitionKind('file')}
                    >
                      Single file
                    </button>
                    <button
                      type="button"
                      className={`tab-btn${definitionKind === 'directory' ? ' active' : ''}`}
                      onClick={() => setDefinitionKind('directory')}
                    >
                      Directory
                    </button>
                  </div>
                  {definitionKind === 'file' ? (
                    <textarea
                      className="form-textarea"
                      value={yaml}
                      onChange={(e) => setYaml(e.target.value)}
                      placeholder="entrypoint: main&#10;workflows:&#10;  - name: main&#10;    steps: []"
                      required
                      rows={14}
                    />
                  ) : (
                    <div className="definition-editor">
                      <div className="file-tree">
                        {files.map((f, idx) => (
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
                      {files[selectedFile] && (
                        <>
                          <input
                            className="form-input"
                            value={files[selectedFile].path}
                            onChange={(e) => updateFile(selectedFile, 'path', e.target.value)}
                            placeholder="workflows/main.yaml"
                            required
                          />
                          <textarea
                            className="form-textarea"
                            value={files[selectedFile].content}
                            onChange={(e) => updateFile(selectedFile, 'content', e.target.value)}
                            rows={12}
                            required
                          />
                        </>
                      )}
                    </div>
                  )}
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
