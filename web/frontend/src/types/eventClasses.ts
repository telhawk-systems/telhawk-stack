/**
 * OCSF Event Class Definitions
 * Based on OCSF 1.1.0 schema and UX Design Philosophy
 */

export interface FieldFilterConfig {
  id: string;
  label: string;
  ocsfPath: string;
  type: 'text' | 'ip' | 'enum' | 'number';
  enumValues?: string[];
}

export interface EventClassConfig {
  classUid: number;
  name: string;
  icon: string;
  description: string;
  primaryFilters: string[];
  secondaryFilters: string[];
}

/**
 * Field filter definitions for all event classes
 * Maps filter IDs to OCSF field paths and UI metadata
 */
export const FIELD_FILTERS: Record<string, FieldFilterConfig> = {
  // Authentication (3002)
  username: { id: 'username', label: 'Username', ocsfPath: 'actor.user.name', type: 'text' },
  source_ip: { id: 'source_ip', label: 'Source IP', ocsfPath: 'src_endpoint.ip', type: 'ip' },
  status: { id: 'status', label: 'Status', ocsfPath: 'status', type: 'enum', enumValues: ['Success', 'Failure', 'Unknown'] },
  auth_protocol: { id: 'auth_protocol', label: 'Auth Protocol', ocsfPath: 'auth_protocol', type: 'enum', enumValues: ['LDAP', 'Kerberos', 'NTLM', 'SAML', 'OAuth'] },
  destination_host: { id: 'destination_host', label: 'Destination Host', ocsfPath: 'dst_endpoint.hostname', type: 'text' },
  session_id: { id: 'session_id', label: 'Session ID', ocsfPath: 'session.uid', type: 'text' },
  mfa_status: { id: 'mfa_status', label: 'MFA Status', ocsfPath: 'auth_factor', type: 'text' },

  // Network Activity (4001)
  dest_ip: { id: 'dest_ip', label: 'Destination IP', ocsfPath: 'dst_endpoint.ip', type: 'ip' },
  protocol: { id: 'protocol', label: 'Protocol', ocsfPath: 'connection_info.protocol_name', type: 'enum', enumValues: ['TCP', 'UDP', 'ICMP', 'HTTP', 'HTTPS'] },
  port: { id: 'port', label: 'Port', ocsfPath: 'dst_endpoint.port', type: 'number' },
  direction: { id: 'direction', label: 'Direction', ocsfPath: 'traffic.direction', type: 'enum', enumValues: ['Inbound', 'Outbound', 'Lateral', 'Unknown'] },
  boundary: { id: 'boundary', label: 'Boundary', ocsfPath: 'traffic.boundary', type: 'enum', enumValues: ['Internal', 'External', 'Unknown'] },
  bytes_transferred: { id: 'bytes_transferred', label: 'Bytes Transferred', ocsfPath: 'traffic.bytes', type: 'number' },

  // Process Activity (1007)
  process_name: { id: 'process_name', label: 'Process Name', ocsfPath: 'process.name', type: 'text' },
  user: { id: 'user', label: 'User', ocsfPath: 'actor.user.name', type: 'text' },
  parent_process: { id: 'parent_process', label: 'Parent Process', ocsfPath: 'parent_process.name', type: 'text' },
  command_line: { id: 'command_line', label: 'Command Line', ocsfPath: 'process.cmd_line', type: 'text' },
  pid: { id: 'pid', label: 'Process ID', ocsfPath: 'process.pid', type: 'number' },
  executable_hash: { id: 'executable_hash', label: 'Executable Hash', ocsfPath: 'process.file.hashes', type: 'text' },

  // File Activity (4006)
  file_path: { id: 'file_path', label: 'File Path', ocsfPath: 'file.path', type: 'text' },
  operation: { id: 'operation', label: 'Operation', ocsfPath: 'activity_name', type: 'text' },
  file_size: { id: 'file_size', label: 'File Size', ocsfPath: 'file.size', type: 'number' },
  file_hash: { id: 'file_hash', label: 'File Hash', ocsfPath: 'file.hashes', type: 'text' },
  modified_time: { id: 'modified_time', label: 'Modified Time', ocsfPath: 'file.modified_time', type: 'text' },

  // DNS Activity (4003)
  query_hostname: { id: 'query_hostname', label: 'Query Hostname', ocsfPath: 'query.hostname', type: 'text' },
  record_type: { id: 'record_type', label: 'Record Type', ocsfPath: 'query.type', type: 'enum', enumValues: ['A', 'AAAA', 'CNAME', 'MX', 'TXT', 'NS', 'SOA', 'PTR'] },
  response_code: { id: 'response_code', label: 'Response Code', ocsfPath: 'rcode', type: 'enum', enumValues: ['NoError', 'FormErr', 'ServFail', 'NXDomain', 'NotImp', 'Refused'] },
  dns_server: { id: 'dns_server', label: 'DNS Server', ocsfPath: 'dst_endpoint.ip', type: 'ip' },
  answer_count: { id: 'answer_count', label: 'Answer Count', ocsfPath: 'answers', type: 'number' },
  ttl: { id: 'ttl', label: 'TTL', ocsfPath: 'ttl', type: 'number' },

  // HTTP Activity (4002)
  method: { id: 'method', label: 'HTTP Method', ocsfPath: 'http_request.http_method', type: 'enum', enumValues: ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS'] },
  url_path: { id: 'url_path', label: 'URL Path', ocsfPath: 'http_request.url.path', type: 'text' },
  status_code: { id: 'status_code', label: 'Status Code', ocsfPath: 'http_response.code', type: 'enum', enumValues: ['200', '201', '204', '301', '302', '400', '401', '403', '404', '500', '502', '503'] },
  client_ip: { id: 'client_ip', label: 'Client IP', ocsfPath: 'src_endpoint.ip', type: 'ip' },
  user_agent: { id: 'user_agent', label: 'User Agent', ocsfPath: 'http_request.user_agent', type: 'text' },
  content_type: { id: 'content_type', label: 'Content Type', ocsfPath: 'http_response.content_type', type: 'text' },
  response_size: { id: 'response_size', label: 'Response Size', ocsfPath: 'http_response.length', type: 'number' },

  // Detection Finding (2004)
  tactic: { id: 'tactic', label: 'MITRE Tactic', ocsfPath: 'attacks[0].tactic.name', type: 'text' },
  technique: { id: 'technique', label: 'MITRE Technique', ocsfPath: 'attacks[0].technique.name', type: 'text' },
  severity: { id: 'severity', label: 'Severity', ocsfPath: 'severity', type: 'enum', enumValues: ['Informational', 'Low', 'Medium', 'High', 'Critical'] },
  risk_score: { id: 'risk_score', label: 'Risk Score', ocsfPath: 'risk_score', type: 'number' },
  analytic_name: { id: 'analytic_name', label: 'Analytic Name', ocsfPath: 'analytic.name', type: 'text' },
  confidence: { id: 'confidence', label: 'Confidence', ocsfPath: 'confidence', type: 'enum', enumValues: ['Low', 'Medium', 'High'] },
  affected_resources: { id: 'affected_resources', label: 'Affected Resources', ocsfPath: 'resources[0].name', type: 'text' },
};

/**
 * OCSF Event Classes with UI Configuration
 */
export const EVENT_CLASSES: Record<number, EventClassConfig> = {
  3002: {
    classUid: 3002,
    name: 'Authentication',
    icon: '🔐',
    description: 'Login attempts, MFA, logout, password changes',
    primaryFilters: ['username', 'source_ip', 'status', 'auth_protocol'],
    secondaryFilters: ['destination_host', 'session_id', 'mfa_status'],
  },
  4001: {
    classUid: 4001,
    name: 'Network Activity',
    icon: '🌐',
    description: 'TCP/UDP/ICMP connections, firewall events',
    primaryFilters: ['source_ip', 'dest_ip', 'protocol', 'port'],
    secondaryFilters: ['direction', 'boundary', 'bytes_transferred'],
  },
  1007: {
    classUid: 1007,
    name: 'Process Activity',
    icon: '⚙️',
    description: 'Process launches with command lines',
    primaryFilters: ['process_name', 'user', 'parent_process'],
    secondaryFilters: ['command_line', 'pid', 'executable_hash'],
  },
  4006: {
    classUid: 4006,
    name: 'File Activity',
    icon: '📁',
    description: 'File operations (create, read, update, delete)',
    primaryFilters: ['file_path', 'operation', 'user'],
    secondaryFilters: ['file_size', 'file_hash', 'modified_time'],
  },
  4003: {
    classUid: 4003,
    name: 'DNS Activity',
    icon: '🔍',
    description: 'DNS queries with various record types',
    primaryFilters: ['query_hostname', 'record_type', 'response_code'],
    secondaryFilters: ['dns_server', 'answer_count', 'ttl'],
  },
  4002: {
    classUid: 4002,
    name: 'HTTP Activity',
    icon: '🌐',
    description: 'HTTP requests with status codes',
    primaryFilters: ['method', 'url_path', 'status_code', 'client_ip'],
    secondaryFilters: ['user_agent', 'content_type', 'response_size'],
  },
  2004: {
    classUid: 2004,
    name: 'Detection Finding',
    icon: '🚨',
    description: 'Security alerts with MITRE ATT&CK tactics',
    primaryFilters: ['tactic', 'technique', 'severity', 'risk_score'],
    secondaryFilters: ['analytic_name', 'confidence', 'affected_resources'],
  },
};

/**
 * Get all event classes as an array sorted by name
 */
export function getAllEventClasses(): EventClassConfig[] {
  return Object.values(EVENT_CLASSES).sort((a, b) => a.name.localeCompare(b.name));
}

/**
 * Get event class by class_uid
 */
export function getEventClass(classUid: number): EventClassConfig | undefined {
  return EVENT_CLASSES[classUid];
}

/**
 * Search event classes by name (case-insensitive)
 */
export function searchEventClasses(query: string): EventClassConfig[] {
  const lowerQuery = query.toLowerCase();
  return getAllEventClasses().filter(ec =>
    ec.name.toLowerCase().includes(lowerQuery) ||
    ec.description.toLowerCase().includes(lowerQuery)
  );
}

/**
 * Get field filter configs for a given event class
 */
export function getEventClassFilters(classUid: number, includePrimary: boolean = true, includeSecondary: boolean = false): FieldFilterConfig[] {
  const eventClass = EVENT_CLASSES[classUid];
  if (!eventClass) return [];

  const filterIds: string[] = [];
  if (includePrimary) {
    filterIds.push(...eventClass.primaryFilters);
  }
  if (includeSecondary) {
    filterIds.push(...eventClass.secondaryFilters);
  }

  return filterIds
    .map(id => FIELD_FILTERS[id])
    .filter(f => f !== undefined);
}

/**
 * Get a single field filter config by ID
 */
export function getFieldFilter(filterId: string): FieldFilterConfig | undefined {
  return FIELD_FILTERS[filterId];
}
