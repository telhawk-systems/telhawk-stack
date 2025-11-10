/**
 * Entity modeling utilities for TelHawk
 *
 * Provides types, detection, and query building for entity-based investigation
 */

/**
 * Supported entity types for investigation
 */
export type EntityType = 'user' | 'ip' | 'hostname' | 'process' | 'file';

/**
 * Entity information extracted from events
 */
export interface Entity {
  type: EntityType;
  value: string;
  displayName: string;
}

/**
 * Icon mapping for entity types
 */
export function getEntityIcon(type: EntityType): string {
  switch (type) {
    case 'user':
      return '👤';
    case 'ip':
      return '🌐';
    case 'hostname':
      return '🖥️';
    case 'process':
      return '⚙️';
    case 'file':
      return '📄';
    default:
      return '📋';
  }
}

/**
 * Get human-readable entity type name
 */
export function getEntityTypeName(type: EntityType): string {
  switch (type) {
    case 'user':
      return 'User';
    case 'ip':
      return 'IP Address';
    case 'hostname':
      return 'Hostname';
    case 'process':
      return 'Process';
    case 'file':
      return 'File';
    default:
      return 'Entity';
  }
}

/**
 * Extract entities from an event
 * Returns all detectable entities in the event
 */
export function extractEntitiesFromEvent(event: any): Entity[] {
  const entities: Entity[] = [];
  const eventData = event.raw?.data?.event || event;

  // Extract users
  const username = eventData.user?.name || eventData.actor?.user?.name;
  if (username) {
    entities.push({
      type: 'user',
      value: username,
      displayName: username,
    });
  }

  // Extract IPs
  const srcIp = eventData.src_endpoint?.ip;
  if (srcIp) {
    entities.push({
      type: 'ip',
      value: srcIp,
      displayName: srcIp,
    });
  }

  const dstIp = eventData.dst_endpoint?.ip;
  if (dstIp && dstIp !== srcIp) {
    entities.push({
      type: 'ip',
      value: dstIp,
      displayName: dstIp,
    });
  }

  // Extract hostnames
  const hostname = eventData.device?.hostname || eventData.observables?.hostname;
  if (hostname) {
    entities.push({
      type: 'hostname',
      value: hostname,
      displayName: hostname,
    });
  }

  // Extract process names
  const processName = eventData.process?.name;
  if (processName) {
    entities.push({
      type: 'process',
      value: processName,
      displayName: processName,
    });
  }

  // Extract file paths
  const filePath = eventData.file?.path;
  if (filePath) {
    entities.push({
      type: 'file',
      value: filePath,
      displayName: filePath,
    });
  }

  return entities;
}

/**
 * Build a query filter for a specific entity
 * Uses the canonical JSON query language
 *
 * Note: Entity fields are stored at the TOP LEVEL of indexed documents
 */
export function buildEntityFilter(entity: Entity): any {
  // Build OR conditions for all possible field locations where this entity might appear
  // Note: Fields should be at TOP LEVEL per OCSF schema (actor, src_endpoint, etc.)
  const conditions: any[] = [];

  switch (entity.type) {
    case 'user':
      conditions.push(
        { field: '.actor.user.name', operator: 'eq', value: entity.value }
      );
      break;

    case 'ip':
      conditions.push(
        { field: '.src_endpoint.ip', operator: 'eq', value: entity.value },
        { field: '.dst_endpoint.ip', operator: 'eq', value: entity.value },
        { field: '.device.ip', operator: 'eq', value: entity.value }
      );
      break;

    case 'hostname':
      conditions.push(
        { field: '.device.hostname', operator: 'eq', value: entity.value },
        { field: '.src_endpoint.hostname', operator: 'eq', value: entity.value },
        { field: '.dst_endpoint.hostname', operator: 'eq', value: entity.value }
      );
      break;

    case 'process':
      conditions.push(
        { field: '.process.name', operator: 'eq', value: entity.value }
      );
      break;

    case 'file':
      conditions.push(
        { field: '.file.path', operator: 'eq', value: entity.value }
      );
      break;
  }

  // Return OR compound filter
  if (conditions.length === 1) {
    return conditions[0];
  }

  return {
    type: 'or',
    conditions: conditions,
  };
}

/**
 * Format entity display name with type badge
 */
export function formatEntityDisplay(entity: Entity): string {
  return `${getEntityIcon(entity.type)} ${entity.value}`;
}

/**
 * Get color class for entity type badge
 */
export function getEntityColorClass(type: EntityType): string {
  switch (type) {
    case 'user':
      return 'bg-blue-100 text-blue-800 border-blue-300';
    case 'ip':
      return 'bg-green-100 text-green-800 border-green-300';
    case 'hostname':
      return 'bg-purple-100 text-purple-800 border-purple-300';
    case 'process':
      return 'bg-orange-100 text-orange-800 border-orange-300';
    case 'file':
      return 'bg-pink-100 text-pink-800 border-pink-300';
    default:
      return 'bg-gray-100 text-gray-800 border-gray-300';
  }
}
