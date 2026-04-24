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
  created_at: string;
  updated_at: string;
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

  createRun(workflowName: string, vars: Record<string, string>) {
    return request<Run>('/runs', {
      method: 'POST',
      body: JSON.stringify({ workflow_name: workflowName, vars }),
    });
  },

  listWorkflows() {
    return request<Workflow[]>('/workflows');
  },

  getWorkflow(name: string) {
    return request<Workflow>(`/workflows/${name}`);
  },

  createWorkflow(name: string, yaml: string) {
    return request<Workflow>('/workflows', {
      method: 'POST',
      body: JSON.stringify({ name, yaml }),
    });
  },
};
