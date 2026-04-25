const BASE = '/api/v1';

function getToken(): string | null {
  return localStorage.getItem('token');
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string> || {}),
  };
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const res = await fetch(`${BASE}${path}`, { ...options, headers });

  if (res.status === 401) {
    localStorage.removeItem('token');
    window.location.href = '/login';
    throw new Error('Unauthorized');
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || `Request failed: ${res.status}`);
  }

  return res.json();
}

export interface User {
  id: number;
  username: string;
}

export interface Workflow {
  id: number;
  name: string;
  yaml: string;
  uploaded_by: number;
  project_id?: number;
  source_path?: string;
  created_at: string;
  updated_at: string;
}

export interface Project {
  id: number;
  name: string;
  url: string;
  branch: string;
  last_synced_at: string | null;
  sync_status: string;
  sync_error: string;
  created_by: number;
  created_at: string;
  updated_at: string;
}

export interface ProjectFile {
  path: string;
  imported: boolean;
}

export interface Run {
  id: number;
  run_id: string;
  workflow_name: string;
  status: string;
  vars_json: string;
  started_at: string | null;
  completed_at: string | null;
  created_at: string;
}

export interface Step {
  id: number;
  run_id: string;
  fork_id: string;
  workflow_name: string;
  step_name: string;
  step_type: string;
  status: string;
  output_json: string;
  error: string;
  started_at: string | null;
  completed_at: string | null;
}

export interface RunDetail extends Run {
  steps: Step[];
}

export const api = {
  login(username: string, password: string) {
    return request<{ token: string }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    });
  },

  register(username: string, password: string) {
    return request<{ token: string }>('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    });
  },

  listRuns() {
    return request<Run[]>('/runs');
  },

  getRun(runID: string) {
    return request<RunDetail>(`/runs/${runID}`);
  },

  createRun(workflowName: string, vars: Record<string, string>, debug = false) {
    return request<Run>('/runs', {
      method: 'POST',
      body: JSON.stringify({ workflow_name: workflowName, vars, debug }),
    });
  },

  listWorkflows() {
    return request<Workflow[]>('/workflows');
  },

  getWorkflow(name: string) {
    return request<Workflow>(`/workflows/${name}`);
  },

  cancelRun(runID: string) {
    return request<Run>(`/runs/${runID}/cancel`, { method: 'POST' });
  },

  deleteRun(runID: string) {
    return fetch(`${BASE}/runs/${runID}`, {
      method: 'DELETE',
      headers: {
        'Authorization': `Bearer ${getToken()}`,
      },
    }).then((res) => {
      if (!res.ok) throw new Error(`Delete failed: ${res.status}`);
    });
  },

  createWorkflow(name: string, yaml: string) {
    return request<Workflow>('/workflows', {
      method: 'POST',
      body: JSON.stringify({ name, yaml }),
    });
  },

  updateWorkflow(name: string, yaml: string) {
    return request<Workflow>(`/workflows/${name}`, {
      method: 'PUT',
      body: JSON.stringify({ yaml }),
    });
  },

  deleteWorkflow(name: string) {
    return fetch(`${BASE}/workflows/${name}`, {
      method: 'DELETE',
      headers: {
        'Authorization': `Bearer ${getToken()}`,
      },
    }).then((res) => {
      if (!res.ok) throw new Error(`Delete failed: ${res.status}`);
    });
  },

  listProjects() {
    return request<Project[]>('/projects');
  },

  createProject(name: string, url: string, branch: string) {
    return request<Project>('/projects', {
      method: 'POST',
      body: JSON.stringify({ name, url, branch }),
    });
  },

  getProject(id: number) {
    return request<Project>(`/projects/${id}`);
  },

  deleteProject(id: number) {
    return fetch(`${BASE}/projects/${id}`, {
      method: 'DELETE',
      headers: {
        'Authorization': `Bearer ${getToken()}`,
      },
    }).then((res) => {
      if (!res.ok) throw new Error(`Delete failed: ${res.status}`);
    });
  },

  syncProject(id: number) {
    return request<Project>(`/projects/${id}/sync`, { method: 'POST' });
  },

  listProjectFiles(id: number) {
    return request<ProjectFile[]>(`/projects/${id}/files`);
  },

  importProjectFiles(id: number, files: string[]) {
    return request<{ name: string; path: string; error?: string }[]>(`/projects/${id}/import`, {
      method: 'POST',
      body: JSON.stringify({ files }),
    });
  },
};
