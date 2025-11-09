import React, { useState, useEffect } from 'react';
import { FilterBar, Filter } from './FilterBar';
import { getFieldFilter } from '../types/eventClasses';

interface SearchConsoleProps {
  onSearch: (query: string, timeRange?: { start: string; end: string }) => void;
  loading: boolean;
}

export function SearchConsole({ onSearch, loading }: SearchConsoleProps) {
  const [query, setQuery] = useState('*');
  const [timeRange, setTimeRange] = useState('24h');
  const [customStart, setCustomStart] = useState('');
  const [customEnd, setCustomEnd] = useState('');
  const [filters, setFilters] = useState<Filter[]>([]);
  const [showAdvancedQuery, setShowAdvancedQuery] = useState(false);
  const [manualQueryMode, setManualQueryMode] = useState(false);

  // Convert filters to OpenSearch query
  const filtersToQuery = (filters: Filter[]): string => {
    if (filters.length === 0) return '*';

    const queryParts: string[] = [];

    // Group field filters by field name to handle OR logic
    const fieldFilters: { [key: string]: string[] } = {};

    filters.forEach(filter => {
      if (filter.type === 'event-class') {
        queryParts.push(`class_uid:${filter.value}`);
      } else if (filter.type === 'field' && filter.field && filter.value) {
        // Get the OCSF field path for this filter
        const fieldConfig = getFieldFilter(filter.field);
        if (fieldConfig) {
          if (!fieldFilters[filter.field]) {
            fieldFilters[filter.field] = [];
          }
          fieldFilters[filter.field].push(filter.value);
        }
      }
    });

    // Convert field filters to query parts
    // Multiple values for same field = OR logic
    // Different fields = AND logic
    Object.entries(fieldFilters).forEach(([fieldId, values]) => {
      const fieldConfig = getFieldFilter(fieldId);
      if (fieldConfig) {
        // For text and IP fields, use .keyword for exact matching in OpenSearch
        const fieldPath = (fieldConfig.type === 'text' || fieldConfig.type === 'ip')
          ? `${fieldConfig.ocsfPath}.keyword`
          : fieldConfig.ocsfPath;

        if (values.length === 1) {
          // Single value: simple query
          queryParts.push(`${fieldPath}:"${values[0]}"`);
        } else {
          // Multiple values: OR logic
          const orClauses = values.map(v => `${fieldPath}:"${v}"`).join(' OR ');
          queryParts.push(`(${orClauses})`);
        }
      }
    });

    return queryParts.length > 0 ? queryParts.join(' AND ') : '*';
  };

  // Update query when filters change (only in filter mode)
  useEffect(() => {
    if (!manualQueryMode && filters.length > 0) {
      setQuery(filtersToQuery(filters));
    } else if (!manualQueryMode && filters.length === 0) {
      setQuery('*');
    }
  }, [filters, manualQueryMode]);

  const handleFiltersChange = (newFilters: Filter[]) => {
    setFilters(newFilters);
    setManualQueryMode(false); // Switch to filter mode when using filters
  };

  const handleQueryChange = (newQuery: string) => {
    setQuery(newQuery);
    // When manually editing query, switch to manual mode and clear filters
    if (newQuery !== filtersToQuery(filters)) {
      setManualQueryMode(true);
      setFilters([]);
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    
    let timeFilter;
    if (timeRange === 'custom' && customStart && customEnd) {
      timeFilter = { start: customStart, end: customEnd };
    } else if (timeRange !== 'all') {
      const end = new Date().toISOString();
      const start = new Date();
      switch (timeRange) {
        case '15m':
          start.setMinutes(start.getMinutes() - 15);
          break;
        case '1h':
          start.setHours(start.getHours() - 1);
          break;
        case '24h':
          start.setHours(start.getHours() - 24);
          break;
        case '7d':
          start.setDate(start.getDate() - 7);
          break;
      }
      timeFilter = { start: start.toISOString(), end };
    }
    
    onSearch(query, timeFilter);
  };

  return (
    <div className="space-y-4">
      {/* Filter Bar (New Progressive Disclosure UI) */}
      <FilterBar onFiltersChange={handleFiltersChange} disabled={loading || manualQueryMode} />

      {/* Manual Query Mode Warning */}
      {manualQueryMode && (
        <div className="bg-yellow-50 border-l-4 border-yellow-400 p-4 rounded-md">
          <div className="flex items-center justify-between">
            <div className="flex items-center">
              <svg className="h-5 w-5 text-yellow-400 mr-2" viewBox="0 0 20 20" fill="currentColor">
                <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
              </svg>
              <p className="text-sm text-yellow-700">
                <span className="font-medium">Manual Query Mode:</span> Filter bar is disabled. Clear the query to use filters again.
              </p>
            </div>
            <button
              onClick={() => {
                setManualQueryMode(false);
                setQuery('*');
              }}
              className="text-sm text-yellow-700 hover:text-yellow-800 underline font-medium"
            >
              Switch to Filter Mode
            </button>
          </div>
        </div>
      )}

      {/* Time Range and Search Controls */}
      <div className="bg-white rounded-lg shadow-md p-6">
        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Advanced Query Toggle */}
          <div className="flex items-center justify-between">
            <h3 className="text-lg font-semibold text-gray-800">Search Options</h3>
            <button
              type="button"
              onClick={() => setShowAdvancedQuery(!showAdvancedQuery)}
              className="text-sm text-blue-600 hover:text-blue-700 font-medium"
            >
              {showAdvancedQuery ? 'Hide' : 'Show'} Advanced Query
            </button>
          </div>

          {/* Advanced Query Input (Collapsible) */}
          {showAdvancedQuery && (
            <div>
              <label htmlFor="query" className="block text-sm font-medium text-gray-700 mb-2">
                OpenSearch Query String (Power User Mode)
              </label>
              <input
                id="query"
                type="text"
                value={query}
                onChange={(e) => handleQueryChange(e.target.value)}
                placeholder="* or search category=iam OR source=auth"
                className="w-full px-4 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                disabled={loading}
              />
              <p className="mt-1 text-xs text-gray-500">
                Examples: *, source=auth, severity_id:3, category=iam AND status=success
              </p>
              <p className="mt-1 text-xs text-yellow-600 font-medium">
                ⚠️ Editing this query will clear all active filters and switch to manual query mode.
              </p>
            </div>
          )}

          {/* Current Query Display (when not in advanced mode) */}
          {!showAdvancedQuery && query !== '*' && (
            <div className="bg-blue-50 border-l-4 border-blue-500 p-3 rounded">
              <p className="text-xs font-medium text-blue-700 mb-1">Generated Query:</p>
              <code className="text-sm text-blue-900">{query}</code>
            </div>
          )}

          {/* Time Range Selector */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Time Range
            </label>
            <div className="grid grid-cols-6 gap-2">
              {['15m', '1h', '24h', '7d', 'all', 'custom'].map((range) => (
                <button
                  key={range}
                  type="button"
                  onClick={() => setTimeRange(range)}
                  className={`px-3 py-2 rounded-md text-sm font-medium transition-colors ${
                    timeRange === range
                      ? 'bg-blue-600 text-white'
                      : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                  }`}
                >
                  {range.toUpperCase()}
                </button>
              ))}
            </div>
          </div>

          {/* Custom Time Range */}
          {timeRange === 'custom' && (
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label htmlFor="start" className="block text-sm font-medium text-gray-700 mb-2">
                  Start Time
                </label>
                <input
                  id="start"
                  type="datetime-local"
                  value={customStart}
                  onChange={(e) => setCustomStart(e.target.value)}
                  className="w-full px-4 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label htmlFor="end" className="block text-sm font-medium text-gray-700 mb-2">
                  End Time
                </label>
                <input
                  id="end"
                  type="datetime-local"
                  value={customEnd}
                  onChange={(e) => setCustomEnd(e.target.value)}
                  className="w-full px-4 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500"
                />
              </div>
            </div>
          )}

          {/* Search Button */}
          <button
            type="submit"
            disabled={loading}
            className="w-full bg-blue-600 text-white px-6 py-3 rounded-md font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {loading ? 'Searching...' : 'Search'}
          </button>
        </form>
      </div>
    </div>
  );
}
