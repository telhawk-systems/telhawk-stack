export interface Alert {
  alert_id: string;
  detection_schema_id: string;
  detection_schema_version_id: string;
  detection_schema_title: string;
  case_id?: string;
  title: string;
  description: string;
  severity: 'critical' | 'high' | 'medium' | 'low' | 'informational';
  priority: 'P1' | 'P2' | 'P3' | 'P4';
  status: 'open' | 'investigating' | 'resolved' | 'false_positive';
  triggered_at: string;
  event_count: number;
  fields: Record<string, any>;
}

export interface AlertDetails extends Alert {
  matched_events: string[];
  mitre_attack?: {
    tactics: string[];
    techniques: string[];
  };
  detection_schema: {
    model: any;
    view: any;
    controller: any;
  };
  metadata: {
    evaluation_duration_ms: number;
    matched_query: string;
  };
}

export interface Case {
  id: string;
  title: string;
  description: string;
  severity: 'critical' | 'high' | 'medium' | 'low';
  priority: 'P1' | 'P2' | 'P3' | 'P4';
  status: 'open' | 'investigating' | 'resolved' | 'closed' | 'false_positive';
  created_by: string;
  created_at: string;
  assigned_to?: string;
  resolved_at?: string;
  closed_at?: string;
  alert_count: number;
  latest_alert_at?: string;
}

export interface CaseDetails extends Case {
  alerts: Array<{
    alert_id: string;
    title: string;
    severity: string;
    triggered_at: string;
    added_at: string;
  }>;
  timeline: Array<{
    timestamp: string;
    event: string;
    actor: string;
    details: string;
  }>;
  metadata?: Record<string, any>;
}

export interface AlertsListResponse {
  alerts: Alert[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    total_pages: number;
  };
}

export interface CasesListResponse {
  cases: Case[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    total_pages: number;
  };
}

export interface AlertUpdateRequest {
  status?: 'open' | 'investigating' | 'resolved' | 'false_positive';
  assigned_to?: string;
  notes?: string;
}

export interface CreateCaseRequest {
  title: string;
  description: string;
  severity: 'critical' | 'high' | 'medium' | 'low';
  priority: 'P1' | 'P2' | 'P3' | 'P4';
  assigned_to?: string;
  alert_ids?: string[];
}
