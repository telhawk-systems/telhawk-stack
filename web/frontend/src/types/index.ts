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

export interface HECToken {
  id: string;
  token: string;
  name: string;
  user_id: string;
  username?: string; // Only present for admin users
  enabled: boolean;
  created_at: string;
  expires_at?: string;
}

// HEC token usage statistics from Redis
export interface HECTokenStats {
  token_id: string;
  last_used_at?: string;
  last_used_ip?: string;
  total_events: number;
  events_last_hour: number;
  events_last_24h: number;
  unique_ips_today: number;
  ingest_instances?: Record<string, string>; // instance_id -> last_seen timestamp
  stats_retrieved_at: string;
}

// Scope types for multi-organization data isolation
export type ScopeType = 'platform' | 'organization' | 'client';

export interface Organization {
  id: string;
  name: string;
  slug: string;
  enabled: boolean;
}

export interface Client {
  id: string;
  name: string;
  slug: string;
  organization_id: string;
  enabled: boolean;
}

// Current viewing scope - determines what data the user sees
export interface ViewingScope {
  type: ScopeType;
  organization_id?: string;  // Set when type is 'organization' or 'client'
  client_id?: string;        // Set when type is 'client'
  // Display info (for UI)
  organization_name?: string;
  client_name?: string;
}

// User's accessible scope - what they're allowed to see
export interface UserScope {
  // Highest tier the user can access
  max_tier: ScopeType;
  // Organizations the user can access (empty = all for platform users)
  organizations: Organization[];
  // Clients the user can access (empty = all within their org scope)
  clients: Client[];
}
