export interface User {
  user_id: string;
  roles: string[];
}

export interface UserDetails extends User {
  id: string;
  username: string;
  email: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  success: boolean;
  message: string;
}

export interface SearchRequest {
  query: string;
  from?: number;
  size?: number;
  sort?: string;
}

export interface SearchResult {
  total: number;
  hits: Array<{
    _id: string;
    _source: Record<string, any>;
  }>;
}
