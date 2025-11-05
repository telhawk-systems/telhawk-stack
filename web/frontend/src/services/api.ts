import { User, LoginRequest, LoginResponse } from '../types';

class ApiClient {
  private baseUrl = '/api';

  async login(credentials: LoginRequest): Promise<LoginResponse> {
    const response = await fetch(`${this.baseUrl}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify(credentials),
    });

    if (!response.ok) {
      throw new Error('Login failed');
    }

    return response.json();
  }

  async logout(): Promise<void> {
    const response = await fetch(`${this.baseUrl}/auth/logout`, {
      method: 'POST',
      credentials: 'include',
    });

    if (!response.ok) {
      throw new Error('Logout failed');
    }
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

  async search(query: string, from = 0, size = 50): Promise<any> {
    const params = new URLSearchParams({
      query,
      from: from.toString(),
      size: size.toString(),
    });

    const response = await fetch(`${this.baseUrl}/query/v1/search?${params}`, {
      credentials: 'include',
    });

    if (!response.ok) {
      throw new Error('Search failed');
    }

    return response.json();
  }
}

export const apiClient = new ApiClient();
