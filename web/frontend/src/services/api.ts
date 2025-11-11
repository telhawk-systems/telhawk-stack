import { User, UserDetails, LoginRequest, LoginResponse, HECToken } from '../types';
import { Query, QueryResponse } from '../types/query';
import { Alert, AlertDetails, AlertsListResponse, AlertUpdateRequest, Case, CaseDetails, CasesListResponse, CreateCaseRequest } from '../types/alerts';

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
        'Accept': 'application/vnd.api+json',
        'Content-Type': 'application/vnd.api+json',
        'X-CSRF-Token': this.csrfToken!,
      },
      credentials: 'include',
      body: JSON.stringify({
        data: {
          type: 'search',
          attributes: {
            query,
            limit: size,
            sort: { field: 'time', order: 'desc' },
            ...(aggregations && { aggregations }),
            ...(searchAfter && { search_after: searchAfter })
          }
        }
      })
    });

    if (!response.ok) {
      throw new Error('Search failed');
    }

    const json = await response.json();
    const attrs = json?.data?.attributes || {};
    return attrs;
  }

  /**
   * Execute a canonical JSON query
   * Uses the new query language from Phase 2 implementation
   */
  async executeQuery(query: Query): Promise<QueryResponse> {
    // Get CSRF token if not already set
    if (!this.csrfToken) {
      await this.getCSRFToken();
    }

    const response = await fetch(`${this.baseUrl}/query/api/v1/events/query`, {
      method: 'POST',
      headers: {
        'Accept': 'application/vnd.api+json',
        'Content-Type': 'application/vnd.api+json',
        'X-CSRF-Token': this.csrfToken!,
      },
      credentials: 'include',
      body: JSON.stringify({ data: { type: 'event-query', attributes: query } }),
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ errors: [{ title: 'Query failed' }] }));
      const title = errorData?.errors?.[0]?.title || 'Query failed';
      throw new Error(title);
    }

    const json = await response.json();
    const items = Array.isArray(json?.data) ? json.data : [];
    const results = items.map((r: any) => r.attributes || {});
    const meta = json?.meta || {};
    const total = meta.total || results.length;
    const took = meta.latency_ms || 0;
    const cursor = meta.next_cursor || undefined;
    const aggregations = meta.aggregations || undefined;
    return { events: results, total, cursor, aggregations, took } as QueryResponse;
  }

  // Saved Searches (JSON:API)
  async listSavedSearches(showAll = false, pageNumber = 1, pageSize = 20, cursor?: string): Promise<{ data: any[]; meta?: any }> {
    const params = new URLSearchParams();
    if (showAll) params.set('filter[show_all]', 'true');
    if (cursor) {
      params.set('page[cursor]', cursor);
      params.set('page[size]', String(pageSize));
    } else {
      params.set('page[number]', String(pageNumber));
      params.set('page[size]', String(pageSize));
    }
    const response = await fetch(`${this.baseUrl}/query/api/v1/saved-searches?${params.toString()}`, {
      credentials: 'include',
      headers: { 'Accept': 'application/vnd.api+json' },
    });
    if (!response.ok) throw new Error('Failed to list saved searches');
    return response.json();
  }

  async createSavedSearch(name: string, query: any, filters: any = {}, isGlobal = false): Promise<any> {
    if (!this.csrfToken) { await this.getCSRFToken(); }
    const body = { data: { type: 'saved-search', attributes: { name, query, filters, is_global: isGlobal }}};
    const response = await fetch(`${this.baseUrl}/query/api/v1/saved-searches`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/vnd.api+json', 'X-CSRF-Token': this.csrfToken! },
      credentials: 'include',
      body: JSON.stringify(body),
    });
    if (!response.ok) throw new Error('Failed to create saved search');
    return response.json();
  }

  async updateSavedSearch(id: string, attrs: any): Promise<any> {
    if (!this.csrfToken) { await this.getCSRFToken(); }
    const body = { data: { id, type: 'saved-search', attributes: attrs }};
    const response = await fetch(`${this.baseUrl}/query/api/v1/saved-searches/${id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/vnd.api+json', 'X-CSRF-Token': this.csrfToken! },
      credentials: 'include',
      body: JSON.stringify(body),
    });
    if (!response.ok) throw new Error('Failed to update saved search');
    return response.json();
  }

  async savedSearchAction(id: string, action: 'disable'|'enable'|'hide'|'unhide'|'run'): Promise<any> {
    if (!this.csrfToken) { await this.getCSRFToken(); }
    const response = await fetch(`${this.baseUrl}/query/api/v1/saved-searches/${id}/${action}`, {
      method: 'POST',
      headers: { 'X-CSRF-Token': this.csrfToken!, 'Accept': 'application/vnd.api+json' },
      credentials: 'include',
    });
    if (action === 'run') {
      if (response.status === 409) throw new Error('Search disabled');
      if (!response.ok) throw new Error('Run failed');
      const json = await response.json();
      const items = Array.isArray(json?.data) ? json.data : [];
      const events = items.map((r: any) => r.attributes || {});
      const meta = json?.meta || {};
      return { events, total: meta.total || events.length, latency_ms: meta.latency_ms || 0 };
    }
    if (!response.ok) throw new Error(`${action} failed`);
    return response.json();
  }

  async getDashboardMetrics(): Promise<any> {
    // Cached endpoint - no CSRF token needed for GET
    const response = await fetch(`${this.baseUrl}/dashboard/metrics`, {
      method: 'GET',
      credentials: 'include',
    });

    if (!response.ok) {
      if (response.status === 401) {
        throw new Error('Please log in to view dashboard metrics');
      }
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

    const data = await response.json();
    // Backward compatibility: API may omit `enabled`; default to true
    return (data as any[]).map((u) => ({ enabled: true, updated_at: '', ...u }));
  }

  // Events API (JSON:API)
  async listEvents(params?: { query?: string; sort?: string; page?: number; size?: number; cursor?: string }): Promise<{ events: any[]; total: number; nextCursor?: string; took?: number }> {
    const search = new URLSearchParams();
    if (params?.query) search.set('filter[query]', params.query);
    if (params?.sort) search.set('sort', params.sort);
    if (params?.cursor) {
      search.set('page[cursor]', params.cursor);
      if (params?.size) search.set('page[size]', String(params.size));
    } else {
      if (params?.page) search.set('page[number]', String(params.page));
      if (params?.size) search.set('page[size]', String(params.size));
    }
    const response = await fetch(`${this.baseUrl}/query/api/v1/events?${search.toString()}`, {
      method: 'GET',
      headers: { 'Accept': 'application/vnd.api+json' },
      credentials: 'include',
    });
    if (!response.ok) {
      if (response.status === 401) throw new Error('Please log in to view events');
      throw new Error('Failed to list events');
    }
    const json = await response.json();
    const items = Array.isArray(json?.data) ? json.data : [];
    const events = items.map((r: any) => r.attributes || {});
    const meta = json?.meta || {};
    return { events, total: meta.total || events.length, nextCursor: meta.next_cursor, took: meta.latency_ms };
  }

  async getUser(id: string): Promise<UserDetails> {
    const response = await fetch(`${this.baseUrl}/auth/api/v1/users/get?id=${id}`, {
      credentials: 'include',
    });

    if (!response.ok) {
      throw new Error('Failed to get user');
    }

    const u = await response.json();
    return { enabled: true, updated_at: '', ...u } as UserDetails;
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

  // Alerts API
  async listAlerts(params?: {
    page?: number;
    limit?: number;
    severity?: string;
    status?: string;
    from?: string;
    to?: string;
    detection_schema_id?: string;
    case_id?: string;
    priority?: string;
  }): Promise<AlertsListResponse> {
    const queryParams = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) {
          queryParams.append(key, String(value));
        }
      });
    }

    const response = await fetch(`${this.baseUrl}/alerting/api/v1/alerts?${queryParams}`, {
      credentials: 'include',
    });

    if (!response.ok) {
      throw new Error('Failed to list alerts');
    }

    return response.json();
  }

  async getAlert(id: string): Promise<AlertDetails> {
    const response = await fetch(`${this.baseUrl}/alerting/api/v1/alerts/${id}`, {
      credentials: 'include',
    });

    if (!response.ok) {
      throw new Error('Failed to get alert');
    }

    return response.json();
  }

  async updateAlert(id: string, updates: AlertUpdateRequest): Promise<Alert> {
    if (!this.csrfToken) {
      await this.getCSRFToken();
    }

    const response = await fetch(`${this.baseUrl}/alerting/api/v1/alerts/${id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': this.csrfToken!,
      },
      credentials: 'include',
      body: JSON.stringify(updates),
    });

    if (!response.ok) {
      throw new Error('Failed to update alert');
    }

    return response.json();
  }

  // Cases API
  async listCases(params?: {
    page?: number;
    limit?: number;
    status?: string;
    severity?: string;
    priority?: string;
    assigned_to?: string;
    from?: string;
    to?: string;
  }): Promise<CasesListResponse> {
    const queryParams = new URLSearchParams();
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) {
          queryParams.append(key, String(value));
        }
      });
    }

    const response = await fetch(`${this.baseUrl}/alerting/api/v1/cases?${queryParams}`, {
      credentials: 'include',
    });

    if (!response.ok) {
      throw new Error('Failed to list cases');
    }

    return response.json();
  }

  async getCase(id: string): Promise<CaseDetails> {
    const response = await fetch(`${this.baseUrl}/alerting/api/v1/cases/${id}`, {
      credentials: 'include',
    });

    if (!response.ok) {
      throw new Error('Failed to get case');
    }

    return response.json();
  }

  async createCase(request: CreateCaseRequest): Promise<Case> {
    if (!this.csrfToken) {
      await this.getCSRFToken();
    }

    const response = await fetch(`${this.baseUrl}/alerting/api/v1/cases`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': this.csrfToken!,
      },
      credentials: 'include',
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      throw new Error('Failed to create case');
    }

    return response.json();
  }
}

export const apiClient = new ApiClient();
