/**
 * TelHawk JSON Query Language Types
 *
 * These types define the canonical JSON query structure used for all search,
 * filtering, and analysis operations across TelHawk.
 *
 * See: docs/QUERY_LANGUAGE_DESIGN.md
 */

/**
 * Supported query operators
 */
export type QueryOperator =
  | 'eq'           // Equals
  | 'ne'           // Not equals
  | 'gt'           // Greater than
  | 'gte'          // Greater than or equal
  | 'lt'           // Less than
  | 'lte'          // Less than or equal
  | 'in'           // In array
  | 'contains'     // String contains
  | 'startsWith'   // String starts with
  | 'endsWith'     // String ends with
  | 'regex'        // Regular expression
  | 'exists'       // Field exists
  | 'cidr';        // IP in CIDR range

/**
 * Basic filter condition (field-operator-value)
 */
export interface FilterCondition {
  field: string;
  operator: QueryOperator;
  value: string | number | boolean | string[] | number[];
}

/**
 * Compound filter with AND/OR/NOT logic
 */
export interface CompoundFilter {
  type: 'and' | 'or' | 'not';
  conditions?: FilterExpression[];  // For 'and' and 'or'
  condition?: FilterExpression;     // For 'not'
}

/**
 * A filter expression can be either a simple condition or a compound filter
 */
export type FilterExpression = FilterCondition | CompoundFilter;

/**
 * Type guard to check if a filter is a compound filter
 */
export function isCompoundFilter(filter: FilterExpression): filter is CompoundFilter {
  return 'type' in filter && ['and', 'or', 'not'].includes(filter.type);
}

/**
 * Type guard to check if a filter is a filter condition
 */
export function isFilterCondition(filter: FilterExpression): filter is FilterCondition {
  return 'field' in filter && 'operator' in filter && 'value' in filter;
}

/**
 * Time range specification
 */
export interface TimeRange {
  // Absolute time range
  start?: string;  // ISO 8601 timestamp
  end?: string;    // ISO 8601 timestamp

  // Relative time range
  last?: string;   // e.g., "15m", "1h", "24h", "7d", "30d", "90d"
}

/**
 * Aggregation types
 */
export type AggregationType =
  | 'terms'          // Group by discrete values
  | 'date_histogram' // Group by time buckets
  | 'avg'            // Average
  | 'sum'            // Sum
  | 'min'            // Minimum
  | 'max'            // Maximum
  | 'stats'          // All metrics (count, avg, sum, min, max)
  | 'cardinality';   // Unique count

/**
 * Aggregation specification
 */
export interface Aggregation {
  type: AggregationType;
  field: string;
  name: string;
  size?: number;              // For terms aggregation
  interval?: string;          // For date_histogram (e.g., "1h", "1d")
  aggregations?: Aggregation[]; // Nested aggregations
}

/**
 * Sort specification
 */
export interface SortSpec {
  field: string;
  order: 'asc' | 'desc';
}

/**
 * Complete JSON query structure
 */
export interface Query {
  select?: string[];           // Field projection (optional)
  filter?: FilterExpression;   // WHERE clause (optional)
  timeRange?: TimeRange;       // Time filtering (optional but recommended)
  aggregations?: Aggregation[]; // GROUP BY / stats (optional)
  sort?: SortSpec[];           // ORDER BY (optional)
  limit?: number;              // Result limit (default: 100)
  offset?: number;             // Pagination offset (optional)
  cursor?: string;             // Cursor-based pagination (optional)
}

/**
 * Query response structure
 */
export interface QueryResponse<T = any> {
  events?: T[];                // Search results
  total: number;               // Total matches
  cursor?: string;             // Next page cursor
  aggregations?: Record<string, AggregationResult>; // Aggregation results
  took?: number;               // Query duration in ms
}

/**
 * Aggregation result bucket
 */
export interface AggregationBucket {
  key: string | number;
  count: number;
  [key: string]: any; // Nested aggregations or metrics
}

/**
 * Aggregation result structure
 */
export interface AggregationResult {
  buckets?: AggregationBucket[];
  value?: number;  // For metric aggregations
  [key: string]: any; // Nested aggregations
}
