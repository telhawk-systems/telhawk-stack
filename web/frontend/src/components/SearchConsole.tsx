import React, { useState, useEffect } from 'react';
import { FilterBar, Filter } from './FilterBar';
import { Query } from '../types/query';
import { buildQuery, getQuerySummary } from '../utils/queryBuilder';

interface SearchConsoleProps {
  onSearch: (query: Query) => void;
  loading: boolean;
}

export function SearchConsole({ onSearch, loading }: SearchConsoleProps) {
  const [timeRange, setTimeRange] = useState('24h');
  const [customStart, setCustomStart] = useState('');
  const [customEnd, setCustomEnd] = useState('');
  const [filters, setFilters] = useState<Filter[]>([]);
  const [showAdvancedQuery, setShowAdvancedQuery] = useState(false);
  const [jsonQuery, setJsonQuery] = useState<Query | null>(null);

  // Update JSON query when filters or time range change
  useEffect(() => {
    const customTimeRange = timeRange === 'custom' && customStart && customEnd
      ? { start: customStart, end: customEnd }
      : undefined;

    const query = buildQuery(filters, timeRange, customTimeRange, { limit: 50 });
    setJsonQuery(query);
  }, [filters, timeRange, customStart, customEnd]);

  const handleFiltersChange = (newFilters: Filter[]) => {
    setFilters(newFilters);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (jsonQuery) {
      onSearch(jsonQuery);
    }
  };

  return (
    <div className="space-y-4">
      {/* Filter Bar (New Progressive Disclosure UI) */}
      <FilterBar onFiltersChange={handleFiltersChange} disabled={loading} />

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
              {showAdvancedQuery ? 'Hide' : 'Show'} JSON Query
            </button>
          </div>

          {/* Generated JSON Query Display */}
          {jsonQuery && (
            <div className="bg-blue-50 border-l-4 border-blue-500 p-3 rounded">
              <div className="flex items-center justify-between mb-1">
                <p className="text-xs font-medium text-blue-700">Query Summary:</p>
                {showAdvancedQuery && (
                  <span className="text-xs text-blue-600">JSON Query Language (Phase 2)</span>
                )}
              </div>
              <code className="text-sm text-blue-900">{getQuerySummary(jsonQuery)}</code>

              {showAdvancedQuery && (
                <div className="mt-3">
                  <p className="text-xs font-medium text-blue-700 mb-1">Generated JSON:</p>
                  <pre className="text-xs text-blue-900 bg-white p-2 rounded border border-blue-200 overflow-x-auto">
                    {JSON.stringify(jsonQuery, null, 2)}
                  </pre>
                </div>
              )}
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
