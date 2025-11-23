const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export interface AccessLog {
  id: number;
  timestamp: string;
  gitlab_project: string;
  harbor_project: string;
  permission: string;
  robot_id?: number;
  robot_name?: string;
  expires_at?: string;
  pipeline_id?: string;
  job_id?: string;
  status: string;
  error_message?: string;
}

export interface PolicyRule {
  id: number;
  gitlab_project: string;
  harbor_projects: string[];
  allowed_permissions: string[];
  created_at: string;
  updated_at: string;
}

export interface AccessLogsResponse {
  logs: AccessLog[];
  total: number;
  page: number;
  limit: number;
}

export const api = {
  async getAccessLogs(params: {
    page?: number;
    limit?: number;
    gitlab_project?: string;
    harbor_project?: string;
    status?: string;
  }): Promise<AccessLogsResponse> {
    const queryParams = new URLSearchParams();
    if (params.page) queryParams.set('page', params.page.toString());
    if (params.limit) queryParams.set('limit', params.limit.toString());
    if (params.gitlab_project) queryParams.set('gitlab_project', params.gitlab_project);
    if (params.harbor_project) queryParams.set('harbor_project', params.harbor_project);
    if (params.status) queryParams.set('status', params.status);

    const response = await fetch(`${API_BASE_URL}/api/access-logs?${queryParams}`);
    if (!response.ok) {
      throw new Error('Failed to fetch access logs');
    }
    return response.json();
  },

  async getPolicies(): Promise<PolicyRule[]> {
    const response = await fetch(`${API_BASE_URL}/api/policies`);
    if (!response.ok) {
      throw new Error('Failed to fetch policies');
    }
    return response.json();
  },

  async createPolicy(policy: Omit<PolicyRule, 'id' | 'created_at' | 'updated_at'>): Promise<PolicyRule> {
    const response = await fetch(`${API_BASE_URL}/api/policies`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(policy),
    });
    if (!response.ok) {
      throw new Error('Failed to create policy');
    }
    return response.json();
  },

  async updatePolicy(id: number, policy: Omit<PolicyRule, 'id' | 'created_at' | 'updated_at'>): Promise<PolicyRule> {
    const response = await fetch(`${API_BASE_URL}/api/policies/${id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(policy),
    });
    if (!response.ok) {
      throw new Error('Failed to update policy');
    }
    return response.json();
  },

  async deletePolicy(id: number): Promise<void> {
    const response = await fetch(`${API_BASE_URL}/api/policies/${id}`, {
      method: 'DELETE',
    });
    if (!response.ok) {
      throw new Error('Failed to delete policy');
    }
  },
};
