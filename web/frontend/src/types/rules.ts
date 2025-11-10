// Detection Schema types based on the Rules Service API

export interface DetectionSchemaModel {
  fields?: string[];
  group_by?: string[];
  time_window?: string;
  threshold?: number;
  aggregation?: string;
}

export interface DetectionSchemaView {
  title: string;
  severity: 'critical' | 'high' | 'medium' | 'low' | 'informational';
  priority?: string;
  fields_order?: string[];
  description_template?: string;
  mitre_attack?: {
    tactics?: string[];
    techniques?: string[];
  };
}

export interface DetectionSchemaController {
  query: string;
  aggregation_field?: string;
  condition?: string;
  lookback?: string;
  evaluation_interval?: string;
}

export interface DetectionSchema {
  id: string;
  version_id: string;
  version: number;
  model: DetectionSchemaModel;
  view: DetectionSchemaView;
  controller: DetectionSchemaController;
  created_by: string;
  created_at: string;
  disabled_at?: string | null;
  disabled_by?: string | null;
  hidden_at?: string | null;
  hidden_by?: string | null;
}

export interface DetectionSchemaCreateRequest {
  model: DetectionSchemaModel;
  view: DetectionSchemaView;
  controller: DetectionSchemaController;
}

export interface DetectionSchemaListResponse {
  schemas: DetectionSchema[];
  pagination: {
    page: number;
    limit: number;
    total: number;
    total_pages: number;
  };
}

export interface DetectionSchemaVersion {
  version_id: string;
  version: number;
  title: string;
  created_by: string;
  created_at: string;
  disabled_at?: string | null;
  changes?: string | null;
}

export interface DetectionSchemaVersionHistory {
  id: string;
  title: string;
  versions: DetectionSchemaVersion[];
}
