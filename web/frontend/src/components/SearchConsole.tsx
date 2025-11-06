import React, { useState } from 'react';

interface SearchConsoleProps {
  onSearch: (query: string, timeRange?: { start: string; end: string }) => void;
  loading: boolean;
}

export function SearchConsole({ onSearch, loading }: SearchConsoleProps) {
  const [query, setQuery] = useState('*');
  const [timeRange, setTimeRange] = useState('24h');
  const [customStart, setCustomStart] = useState('');
  const [customEnd, setCustomEnd] = useState('');

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
    <div className="bg-white rounded-lg shadow-md p-6">
      <h2 className="text-2xl font-bold mb-4 text-gray-800">Search Events</h2>
      
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label htmlFor="query" className="block text-sm font-medium text-gray-700 mb-2">
            Search Query (SPL-like syntax)
          </label>
          <input
            id="query"
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="* or search category=iam OR source=auth"
            className="w-full px-4 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            disabled={loading}
          />
          <p className="mt-1 text-xs text-gray-500">
            Examples: *, source=auth, severity_id:3, category=iam AND status=success
          </p>
        </div>

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

        <button
          type="submit"
          disabled={loading}
          className="w-full bg-blue-600 text-white px-6 py-3 rounded-md font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {loading ? 'Searching...' : 'Search'}
        </button>
      </form>
    </div>
  );
}
