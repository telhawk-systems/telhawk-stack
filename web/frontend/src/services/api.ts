import { User, UserDetails, LoginRequest, LoginResponse } from '../types';

class ApiClient {
  private baseUrl = '/api';
  private csrfToken: string | null = null;

  async getCSRFToken(): Promise<string> {
    const response = await fetch(`${this.baseUrl}/auth/csrf-token`, {
      credentials: 'include',
    });

    if (!response.ok) {
      throw new Error('Failed to get CSRF token');
    }

    const data = await response.json();
    this.csrfToken = data.csrf_token;
    return data.csrf_token;
  }

  async login(credentials: LoginRequest): Promise<LoginResponse> {
    // Get CSRF token before login
    if (!this.csrfToken) {
      await this.getCSRFToken();
    }

    const response = await fetch(`${this.baseUrl}/auth/login`, {
      method: 'POST',
      headers: { 
        'Content-Type': 'application/json',
        'X-CSRF-Token': this.csrfToken!,
      },
      credentials: 'include',
      body: JSON.stringify(credentials),
    });

    if (!response.ok) {
      throw new Error('Login failed');
    }

    return response.json();
  }

  async logout(): Promise<void> {
    // Get CSRF token if not already set
    if (!this.csrfToken) {
      await this.getCSRFToken();
    }

    const response = await fetch(`${this.baseUrl}/auth/logout`, {
      method: 'POST',
      headers: {
        'X-CSRF-Token': this.csrfToken!,
      },
      credentials: 'include',
    });

    if (!response.ok) {
      throw new Error('Logout failed');
    }

    // Clear the CSRF token after logout
    this.csrfToken = null;
  }

  async getCurrentUser(): Promise<User> {
    const response = await fetch(`${this.baseUrl}/auth/me`, {
      credentials: 'include',
    });

    if (!response.ok) {
      throw new Error('Failed to fetch user');
    }

    return response.json();
  }

  async search(query: string, size = 50, aggregations?: any): Promise<any> {
    // Get CSRF token if not already set
    if (!this.csrfToken) {
      await this.getCSRFToken();
    }

    const response = await fetch(`${this.baseUrl}/query/api/v1/search`, {
      method: 'POST',
      headers: { 
        'Content-Type': 'application/json',
        'X-CSRF-Token': this.csrfToken!,
      },
      credentials: 'include',
      body: JSON.stringify({
        query,
        limit: size,
        sort: { field: 'time', order: 'desc' },
        ...(aggregations && { aggregations })
      }),
    });

    if (!response.ok) {
      throw new Error('Search failed');
    }

    return response.json();
  }

  async getDashboardMetrics(timeRange?: { start: string; end: string }): Promise<any> {
    // Get CSRF token if not already set
    if (!this.csrfToken) {
      await this.getCSRFToken();
    }

    let query = '*';
    if (timeRange) {
      query = `time:[${timeRange.start} TO ${timeRange.end}]`;
    }

    const response = await fetch(`${this.baseUrl}/query/api/v1/search`, {
      method: 'POST',
      headers: { 
        'Content-Type': 'application/json',
        'X-CSRF-Token': this.csrfToken!,
      },
      credentials: 'include',
      body: JSON.stringify({
        query,
        limit: 0,
        aggregations: {
          severity_count: {
            type: 'terms',
            field: 'severity',
            size: 10
          },
          events_by_class: {
            type: 'terms',
            field: 'class_name',
            size: 10
          },
          timeline: {
            type: 'date_histogram',
            field: 'time',
            opts: { interval: '1h' }
          },
          unique_users: {
            type: 'cardinality',
            field: 'actor.user.name'
          },
          unique_ips: {
            type: 'cardinality',
            field: 'src_endpoint.ip'
          }
        }
      }),
    });

    if (!response.ok) {
      throw new Error('Failed to fetch dashboard metrics');
    }

    return response.json();
  }

  // User Management API
  async listUsers(): Promise<UserDetails[]> {
    const response = await fetch(`${this.baseUrl}/auth/api/v1/users`, {
      credentials: 'include',
    });

    if (!response.ok) {
      throw new Error('Failed to list users');
    }

    return response.json();
  }

  async getUser(id: string): Promise<UserDetails> {
    const response = await fetch(`${this.baseUrl}/auth/api/v1/users/get?id=${id}`, {
      credentials: 'include',
    });

    if (!response.ok) {
      throw new Error('Failed to get user');
    }

    return response.json();
  }

  async updateUser(id: string, updates: Partial<UserDetails>): Promise<UserDetails> {
    // Get CSRF token if not already set
    if (!this.csrfToken) {
      await this.getCSRFToken();
    }

    const response = await fetch(`${this.baseUrl}/auth/api/v1/users/update?id=${id}`, {
      method: 'PUT',
      headers: { 
        'Content-Type': 'application/json',
        'X-CSRF-Token': this.csrfToken!,
      },
      credentials: 'include',
      body: JSON.stringify(updates),
    });

    if (!response.ok) {
      throw new Error('Failed to update user');
    }

    return response.json();
  }

  async deleteUser(id: string): Promise<void> {
    // Get CSRF token if not already set
    if (!this.csrfToken) {
      await this.getCSRFToken();
    }

    const response = await fetch(`${this.baseUrl}/auth/api/v1/users/delete?id=${id}`, {
      method: 'DELETE',
      headers: {
        'X-CSRF-Token': this.csrfToken!,
      },
      credentials: 'include',
    });

    if (!response.ok) {
      throw new Error('Failed to delete user');
    }
  }

  async resetPassword(id: string, newPassword: string): Promise<void> {
    // Get CSRF token if not already set
    if (!this.csrfToken) {
      await this.getCSRFToken();
    }

    const response = await fetch(`${this.baseUrl}/auth/api/v1/users/reset-password?id=${id}`, {
      method: 'POST',
      headers: { 
        'Content-Type': 'application/json',
        'X-CSRF-Token': this.csrfToken!,
      },
      credentials: 'include',
      body: JSON.stringify({ new_password: newPassword }),
    });

    if (!response.ok) {
      throw new Error('Failed to reset password');
    }
  }
}

export const apiClient = new ApiClient();
