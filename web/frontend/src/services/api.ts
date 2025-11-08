import { User, UserDetails, LoginRequest, LoginResponse, HECToken } from '../types';

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

  async search(query: string, size = 50, aggregations?: any, searchAfter?: any[]): Promise<any> {
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
        ...(aggregations && { aggregations }),
        ...(searchAfter && { search_after: searchAfter })
      }),
    });

    if (!response.ok) {
      throw new Error('Search failed');
    }

    return response.json();
  }

  async getDashboardMetrics(): Promise<any> {
    // Cached endpoint - no CSRF token needed for GET
    const response = await fetch(`${this.baseUrl}/dashboard/metrics`, {
      method: 'GET',
      credentials: 'include',
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

  async createUser(username: string, email: string, password: string, roles: string[]): Promise<UserDetails> {
    // Get CSRF token if not already set
    if (!this.csrfToken) {
      await this.getCSRFToken();
    }

    const response = await fetch(`${this.baseUrl}/auth/api/v1/auth/register`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': this.csrfToken!,
      },
      credentials: 'include',
      body: JSON.stringify({ username, email, password, roles }),
    });

    if (!response.ok) {
      throw new Error('Failed to create user');
    }

    return response.json();
  }

  // HEC Token Management API
  async createHECToken(name: string): Promise<HECToken> {
    // Get CSRF token if not already set
    if (!this.csrfToken) {
      await this.getCSRFToken();
    }

    const response = await fetch(`${this.baseUrl}/auth/api/v1/hec/tokens`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': this.csrfToken!,
      },
      credentials: 'include',
      body: JSON.stringify({ name }),
    });

    if (!response.ok) {
      throw new Error('Failed to create HEC token');
    }

    return response.json();
  }

  async listHECTokens(): Promise<HECToken[]> {
    const response = await fetch(`${this.baseUrl}/auth/api/v1/hec/tokens`, {
      credentials: 'include',
    });

    if (!response.ok) {
      throw new Error('Failed to list HEC tokens');
    }

    return response.json();
  }

  async revokeHECToken(id: string): Promise<void> {
    // Get CSRF token if not already set
    if (!this.csrfToken) {
      await this.getCSRFToken();
    }

    const response = await fetch(`${this.baseUrl}/auth/api/v1/hec/tokens/${id}/revoke`, {
      method: 'DELETE',
      headers: {
        'X-CSRF-Token': this.csrfToken!,
      },
      credentials: 'include',
    });

    if (!response.ok) {
      throw new Error('Failed to revoke HEC token');
    }
  }
}

export const apiClient = new ApiClient();
