import { useState, useEffect } from 'react';
import type { FormEvent } from 'react';
import { api } from '../api';
import type { Project, ProjectFile } from '../api';

const badgeClass = (status: string) => {
  switch (status) {
    case 'synced': return 'badge badge-completed';
    case 'syncing': return 'badge badge-running';
    case 'error': return 'badge badge-failed';
    default: return 'badge badge-pending';
  }
};

export default function Projects() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [name, setName] = useState('');
  const [url, setUrl] = useState('');
  const [branch, setBranch] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [selected, setSelected] = useState<Project | null>(null);
  const [files, setFiles] = useState<ProjectFile[]>([]);
  const [selectedFiles, setSelectedFiles] = useState<Set<string>>(new Set());
  const [syncing, setSyncing] = useState<number | null>(null);
  const [importing, setImporting] = useState(false);

  const loadProjects = async () => {
    try {
      setProjects(await api.listProjects());
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load projects');
    }
  };

  const loadFiles = async (projectId: number) => {
    try {
      const fileList = await api.listProjectFiles(projectId);
      setFiles(fileList);
      setSelectedFiles(new Set());
    } catch {
      setFiles([]);
    }
  };

  useEffect(() => { loadProjects(); }, []);

  useEffect(() => {
    if (selected && selected.sync_status === 'synced') {
      loadFiles(selected.id);
    } else {
      setFiles([]);
      setSelectedFiles(new Set());
    }
  }, [selected]);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');
    try {
      await api.createProject(name, url, branch || 'main');
      setSuccess(`Project "${name}" created.`);
      setName('');
      setUrl('');
      setBranch('');
      loadProjects();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create project');
    }
  };

  const handleSync = async (project: Project) => {
    setError('');
    setSuccess('');
    setSyncing(project.id);
    try {
      const updated = await api.syncProject(project.id);
      if (updated.sync_status === 'error') {
        setError(`Sync failed: ${updated.sync_error}`);
      } else {
        setSuccess(`Project "${project.name}" synced.`);
      }
      await loadProjects();
      if (selected?.id === project.id) {
        setSelected(updated);
        if (updated.sync_status === 'synced') {
          loadFiles(project.id);
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Sync failed');
    } finally {
      setSyncing(null);
    }
  };

  const handleDelete = async (project: Project) => {
    if (!window.confirm(`Delete project "${project.name}"? Imported workflows will become editable.`)) return;
    setError('');
    try {
      await api.deleteProject(project.id);
      if (selected?.id === project.id) {
        setSelected(null);
        setFiles([]);
      }
      await loadProjects();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete project');
    }
  };

  const toggleFile = (path: string) => {
    setSelectedFiles(prev => {
      const next = new Set(prev);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });
  };

  const handleImport = async () => {
    if (!selected || selectedFiles.size === 0) return;
    setError('');
    setSuccess('');
    setImporting(true);
    try {
      const results = await api.importProjectFiles(selected.id, Array.from(selectedFiles));
      const errors = results.filter(r => r.error);
      if (errors.length > 0) {
        setError(`Failed to import: ${errors.map(e => e.path).join(', ')}`);
      } else {
        setSuccess(`Imported ${results.length} workflow(s).`);
      }
      setSelectedFiles(new Set());
      loadFiles(selected.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Import failed');
    } finally {
      setImporting(false);
    }
  };

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">Projects</h1>
      </div>

      <div className="split-layout">
        <div>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Name</th>
                  <th>URL</th>
                  <th>Branch</th>
                  <th>Status</th>
                  <th>Last Synced</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {projects.map((p) => (
                  <tr
                    key={p.id}
                    className={`wf-row${selected?.id === p.id ? ' selected' : ''}`}
                    onClick={() => setSelected(p)}
                  >
                    <td className="cell-mono">{p.name}</td>
                    <td className="cell-mono" style={{ maxWidth: 240, overflow: 'hidden', textOverflow: 'ellipsis' }}>{p.url}</td>
                    <td className="cell-mono">{p.branch}</td>
                    <td><span className={badgeClass(p.sync_status)}>{p.sync_status}</span></td>
                    <td className="cell-mono">
                      {p.last_synced_at ? new Date(p.last_synced_at).toLocaleString() : '—'}
                    </td>
                    <td onClick={(e) => e.stopPropagation()}>
                      <button
                        className="btn btn-sm btn-ghost"
                        onClick={() => handleSync(p)}
                        disabled={syncing === p.id}
                      >
                        {syncing === p.id ? 'Syncing...' : 'Sync'}
                      </button>
                      {' '}
                      <button
                        className="btn btn-sm btn-danger"
                        onClick={() => handleDelete(p)}
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                ))}
                {projects.length === 0 && (
                  <tr>
                    <td colSpan={6} className="table-empty">
                      No projects added yet.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>

          {selected && selected.sync_status === 'synced' && (
            <div style={{ marginTop: 20 }}>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <div className="section-heading" style={{ margin: 0 }}>
                  {selected.name} — Workflow Files
                </div>
                <button
                  className="btn btn-primary btn-sm"
                  onClick={handleImport}
                  disabled={selectedFiles.size === 0 || importing}
                >
                  {importing ? 'Importing...' : `Import Selected (${selectedFiles.size})`}
                </button>
              </div>
              {files.length === 0 ? (
                <p style={{ color: 'var(--text-muted)', marginTop: 12 }}>No YAML files found in repository.</p>
              ) : (
                <ul className="file-tree">
                  {files.map((f) => (
                    <li key={f.path} className={`file-tree-item${f.imported ? ' imported' : ''}`}>
                      <input
                        type="checkbox"
                        checked={f.imported || selectedFiles.has(f.path)}
                        disabled={f.imported}
                        onChange={() => toggleFile(f.path)}
                      />
                      <span>{f.path}</span>
                      {f.imported && <span className="source-badge source-badge-project">imported</span>}
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )}

          {selected && selected.sync_status === 'error' && (
            <div style={{ marginTop: 20 }}>
              <div className="msg-error">{selected.sync_error}</div>
            </div>
          )}

          {selected && selected.sync_status === 'idle' && (
            <div style={{ marginTop: 20, color: 'var(--text-muted)' }}>
              Click <strong>Sync</strong> to clone the repository and browse workflow files.
            </div>
          )}
        </div>

        <div className="card">
          <div className="card-title">Add Project</div>
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
                placeholder="my-project"
                required
              />
            </div>
            <div className="form-group">
              <label className="form-label">Git URL</label>
              <input
                type="text"
                className="form-input"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                placeholder="https://github.com/org/repo.git"
                required
              />
            </div>
            <div className="form-group">
              <label className="form-label">Branch</label>
              <input
                type="text"
                className="form-input"
                value={branch}
                onChange={(e) => setBranch(e.target.value)}
                placeholder="main"
              />
            </div>
            <button type="submit" className="btn btn-primary">
              Add Project
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
