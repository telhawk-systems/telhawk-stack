// OCSF Event Class UIDs
export enum EventClassUID {
  Authentication = 3002,
  NetworkActivity = 4001,
  ProcessActivity = 1007,
  FileActivity = 4006,
  DNSActivity = 4003,
  HTTPActivity = 4002,
  DetectionFinding = 2004,
}

export type EventType =
  | 'authentication'
  | 'network'
  | 'process'
  | 'file'
  | 'dns'
  | 'http'
  | 'detection'
  | 'unknown';

/**
 * Detect event type from class_uid
 */
export function detectEventType(event: any): EventType {
  const classUid = event.class_uid || event._source?.class_uid;

  switch (classUid) {
    case EventClassUID.Authentication:
      return 'authentication';
    case EventClassUID.NetworkActivity:
      return 'network';
    case EventClassUID.ProcessActivity:
      return 'process';
    case EventClassUID.FileActivity:
      return 'file';
    case EventClassUID.DNSActivity:
      return 'dns';
    case EventClassUID.HTTPActivity:
      return 'http';
    case EventClassUID.DetectionFinding:
      return 'detection';
    default:
      return 'unknown';
  }
}

/**
 * Get friendly display name for event type
 */
export function getEventTypeName(type: EventType): string {
  const names: Record<EventType, string> = {
    authentication: 'Authentication',
    network: 'Network Activity',
    process: 'Process Activity',
    file: 'File Activity',
    dns: 'DNS Activity',
    http: 'HTTP Activity',
    detection: 'Security Detection',
    unknown: 'Unknown',
  };

  return names[type] || 'Unknown';
}

/**
 * Get icon for event type (can be used with icon libraries)
 */
export function getEventTypeIcon(type: EventType): string {
  const icons: Record<EventType, string> = {
    authentication: '🔐',
    network: '🌐',
    process: '⚙️',
    file: '📁',
    dns: '🔍',
    http: '🌐',
    detection: '🚨',
    unknown: '❓',
  };

  return icons[type] || '❓';
}

/**
 * Get color theme for event type
 */
export function getEventTypeColor(type: EventType): string {
  const colors: Record<EventType, string> = {
    authentication: 'blue',
    network: 'green',
    process: 'purple',
    file: 'yellow',
    dns: 'cyan',
    http: 'indigo',
    detection: 'red',
    unknown: 'gray',
  };

  return colors[type] || 'gray';
}
