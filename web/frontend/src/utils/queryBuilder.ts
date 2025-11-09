/**
 * TelHawk Query Builder
 *
 * Converts filter chips to canonical JSON query structure.
 * This is the bridge between the UI filter bar and the backend query service.
 *
 * See: docs/QUERY_LANGUAGE_DESIGN.md
 */

import {
  Query,
  FilterExpression,
  FilterCondition,
  CompoundFilter,
  TimeRange,
  QueryOperator,
} from '../types/query';
import { Filter } from '../components/FilterBar';
import { getFieldFilter } from '../types/eventClasses';

/**
 * OCSF-aware default fields per event class
 *
 * When filtering to a single event class, automatically select relevant fields
 * to reduce noise and improve query performance (column projection).
 */
export const EVENT_CLASS_DEFAULT_FIELDS: Record<number, string[]> = {
  // Authentication (3002)
  3002: [
    '.time',
    '.severity',
    '.actor.user.name',
    '.src_endpoint.ip',
    '.status',
    '.auth_protocol.name',
  ],
  // Network Activity (4001)
  4001: [
    '.time',
    '.severity',
    '.src_endpoint.ip',
    '.src_endpoint.port',
    '.dst_endpoint.ip',
    '.dst_endpoint.port',
    '.connection_info.protocol_name',
  ],
  // Process Activity (1007)
  1007: [
    '.time',
    '.severity',
    '.process.name',
    '.process.pid',
    '.process.cmd_line',
    '.actor.user.name',
  ],
  // File Activity (4006)
  4006: [
    '.time',
    '.severity',
    '.file.path',
    '.activity_name',
    '.actor.user.name',
    '.file.size',
  ],
  // DNS Activity (4003)
  4003: [
    '.time',
    '.severity',
    '.query.hostname',
    '.query.type',
    '.rcode',
    '.answers',
  ],
  // HTTP Activity (4002)
  4002: [
    '.time',
    '.severity',
    '.http_request.http_method',
    '.http_request.url.path',
    '.http_response.code',
    '.src_endpoint.ip',
  ],
  // Detection Finding (2004)
  2004: [
    '.time',
    '.severity',
    '.finding.title',
    '.attacks[0].tactic.name',
    '.attacks[0].technique.name',
    '.risk_score',
  ],
};

/**
 * Convert a filter chip field path to OCSF jq-style path
 *
 * @param fieldId - Field filter ID (e.g., 'username', 'source_ip')
 * @returns OCSF field path with leading dot (e.g., '.actor.user.name')
 */
function toOCSFPath(fieldId: string): string {
  const fieldConfig = getFieldFilter(fieldId);
  if (!fieldConfig) {
    throw new Error(`Unknown field filter: ${fieldId}`);
  }

  // Add leading dot for jq-style OCSF paths
  return `.${fieldConfig.ocsfPath}`;
}

/**
 * Determine the appropriate operator based on field type and value
 *
 * @param fieldId - Field filter ID
 * @param value - Filter value
 * @returns Query operator
 */
function determineOperator(fieldId: string, value: string): QueryOperator {
  const fieldConfig = getFieldFilter(fieldId);
  if (!fieldConfig) {
    return 'eq'; // Default to equals
  }

  // Check for CIDR notation (IP fields with / in the value)
  if (fieldConfig.type === 'ip' && value.includes('/')) {
    return 'cidr';
  }

  // Check for wildcards (text fields with * in the value)
  if (fieldConfig.type === 'text') {
    if (value.startsWith('*') && value.endsWith('*')) {
      return 'contains';
    }
    if (value.startsWith('*')) {
      return 'endsWith';
    }
    if (value.endsWith('*')) {
      return 'startsWith';
    }
  }

  // Default to exact match
  return 'eq';
}

/**
 * Normalize filter value based on field type
 *
 * @param fieldId - Field filter ID
 * @param value - Raw filter value
 * @returns Normalized value (with wildcards removed, proper type conversion)
 */
function normalizeValue(fieldId: string, value: string): string | number {
  const fieldConfig = getFieldFilter(fieldId);
  if (!fieldConfig) {
    return value;
  }

  // Strip wildcards for contains/startsWith/endsWith operators
  let normalized = value;
  if (fieldConfig.type === 'text') {
    normalized = value.replace(/^\*|\*$/g, '');
  }

  // Convert numbers
  if (fieldConfig.type === 'number') {
    const num = parseFloat(normalized);
    return isNaN(num) ? normalized : num;
  }

  return normalized;
}

/**
 * Build a filter expression from filter chips
 *
 * Handles:
 * - Event class filters (.class_uid)
 * - Field filters with OCSF paths
 * - Multiple values for same field (OR logic)
 * - Different fields (AND logic)
 *
 * @param filters - Array of filter chips
 * @returns Filter expression (or undefined if no filters)
 */
export function buildFilterExpression(filters: Filter[]): FilterExpression | undefined {
  if (filters.length === 0) {
    return undefined;
  }

  const conditions: FilterCondition[] = [];
  const fieldValueGroups: Record<string, string[]> = {};

  filters.forEach(filter => {
    if (filter.type === 'event-class') {
      // Event class filter: .class_uid = <classUid>
      conditions.push({
        field: '.class_uid',
        operator: 'eq',
        value: parseInt(filter.value, 10),
      });
    } else if (filter.type === 'field' && filter.field) {
      // Field filter: group by field for OR logic
      if (!fieldValueGroups[filter.field]) {
        fieldValueGroups[filter.field] = [];
      }
      fieldValueGroups[filter.field].push(filter.value);
    }
  });

  // Convert field value groups to filter conditions
  Object.entries(fieldValueGroups).forEach(([fieldId, values]) => {
    const ocsfPath = toOCSFPath(fieldId);

    if (values.length === 1) {
      // Single value: simple condition
      const operator = determineOperator(fieldId, values[0]);
      const value = normalizeValue(fieldId, values[0]);

      conditions.push({
        field: ocsfPath,
        operator,
        value,
      });
    } else {
      // Multiple values: OR logic
      const orConditions: FilterCondition[] = values.map(v => ({
        field: ocsfPath,
        operator: determineOperator(fieldId, v),
        value: normalizeValue(fieldId, v),
      }));

      // Check if all conditions use 'eq' operator - optimize to 'in'
      const allEq = orConditions.every(c => c.operator === 'eq');
      if (allEq) {
        const values = orConditions.map(c => c.value);
        // Type assertion: we know these are all the same type because they came from the same field
        conditions.push({
          field: ocsfPath,
          operator: 'in',
          value: values as string[] | number[],
        });
      } else {
        // Mixed operators: use OR compound filter
        const orFilter: CompoundFilter = {
          type: 'or',
          conditions: orConditions,
        };
        conditions.push(orFilter as any); // Type assertion needed due to union type
      }
    }
  });

  // Return single condition or AND compound filter
  if (conditions.length === 0) {
    return undefined;
  } else if (conditions.length === 1) {
    return conditions[0];
  } else {
    return {
      type: 'and',
      conditions,
    };
  }
}

/**
 * Build time range from UI time range selector
 *
 * @param timeRangeValue - Selected time range ('15m', '1h', '24h', '7d', 'custom', 'all')
 * @param customRange - Custom time range (if timeRangeValue is 'custom')
 * @returns TimeRange object (or undefined for 'all')
 */
export function buildTimeRange(
  timeRangeValue: string,
  customRange?: { start: string; end: string }
): TimeRange | undefined {
  if (timeRangeValue === 'all') {
    return undefined; // No time filter
  }

  if (timeRangeValue === 'custom' && customRange) {
    return {
      start: customRange.start,
      end: customRange.end,
    };
  }

  // Relative time range
  if (['15m', '1h', '24h', '7d'].includes(timeRangeValue)) {
    return {
      last: timeRangeValue,
    };
  }

  return undefined;
}

/**
 * Get default select fields for a query based on event class filter
 *
 * If a single event class is filtered, returns OCSF-aware default fields.
 * Otherwise, returns undefined (all fields).
 *
 * @param filters - Array of filter chips
 * @returns Array of OCSF field paths (or undefined for all fields)
 */
export function buildSelectFields(filters: Filter[]): string[] | undefined {
  const eventClassFilter = filters.find(f => f.type === 'event-class');
  if (!eventClassFilter) {
    return undefined; // No event class filter, return all fields
  }

  const classUid = parseInt(eventClassFilter.value, 10);
  return EVENT_CLASS_DEFAULT_FIELDS[classUid] || undefined;
}

/**
 * Build a complete JSON query from filter chips and search options
 *
 * This is the main entry point for converting UI state to JSON query.
 *
 * @param filters - Array of filter chips
 * @param timeRange - Selected time range
 * @param customTimeRange - Custom time range (if applicable)
 * @param options - Additional query options
 * @returns Complete JSON query object
 */
export function buildQuery(
  filters: Filter[],
  timeRange: string = '24h',
  customTimeRange?: { start: string; end: string },
  options?: {
    limit?: number;
    offset?: number;
    cursor?: string;
  }
): Query {
  const query: Query = {
    filter: buildFilterExpression(filters),
    timeRange: buildTimeRange(timeRange, customTimeRange),
    select: buildSelectFields(filters),
    sort: [{ field: '.time', order: 'desc' }], // Default sort by time descending
  };

  // Add pagination options
  if (options?.limit) {
    query.limit = options.limit;
  }
  if (options?.offset) {
    query.offset = options.offset;
  }
  if (options?.cursor) {
    query.cursor = options.cursor;
  }

  return query;
}

/**
 * Helper: Check if a query has any active filters
 */
export function hasFilters(query: Query): boolean {
  return query.filter !== undefined;
}

/**
 * Helper: Get human-readable summary of query
 */
export function getQuerySummary(query: Query): string {
  const parts: string[] = [];

  if (query.filter) {
    parts.push('Filtered');
  }

  if (query.timeRange?.last) {
    parts.push(`Last ${query.timeRange.last}`);
  } else if (query.timeRange?.start && query.timeRange?.end) {
    parts.push('Custom time range');
  }

  if (query.select) {
    parts.push(`${query.select.length} fields`);
  }

  return parts.length > 0 ? parts.join(' • ') : 'All events';
}
