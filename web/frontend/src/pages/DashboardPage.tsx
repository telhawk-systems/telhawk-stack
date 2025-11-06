import { useState } from 'react';
import { useAuth } from '../components/AuthProvider';
import { SearchConsole } from '../components/SearchConsole';
import { EventsTable } from '../components/EventsTable';
import { EventDetailModal } from '../components/EventDetailModal';
import { apiClient } from '../services/api';

export function DashboardPage() {
  const { user, logout } = useAuth();
  const [results, setResults] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [selectedEvent, setSelectedEvent] = useState<any>(null);

  const handleSearch = async (query: string, timeRange?: { start: string; end: string }) => {
    setError('');
    setLoading(true);
    setResults(null);

    try {
      // Build query with time range if provided
      let searchQuery = query;
      if (timeRange) {
        searchQuery = `${query} time:[${timeRange.start} TO ${timeRange.end}]`;
      }
      
      const data = await apiClient.search(searchQuery);
      setResults(data);
    } catch (err) {
      setError('Search failed. Please try again.');
      console.error('Search error:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleLogout = async () => {
    try {
      await logout();
    } catch (err) {
      console.error('Logout failed', err);
    }
  };

  return (
    <div className="min-h-screen bg-gray-100">
      {/* Header */}
      <header className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
          <div className="flex justify-between items-center">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">TelHawk SIEM</h1>
              <p className="text-sm text-gray-500 mt-1">OCSF-compliant Security Event Management</p>
            </div>
            <div className="flex items-center space-x-4">
              <div className="text-right">
                <p className="text-sm font-medium text-gray-900">{user?.user_id}</p>
                <p className="text-xs text-gray-500">
                  Roles: {user?.roles.join(', ')}
                </p>
              </div>
              <button
                onClick={handleLogout}
                className="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 font-medium transition-colors"
              >
                Logout
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="space-y-6">
          {/* Search Console */}
          <SearchConsole onSearch={handleSearch} loading={loading} />

          {/* Error Message */}
          {error && (
            <div className="bg-red-50 border-l-4 border-red-500 p-4 rounded-md">
              <div className="flex">
                <div className="flex-shrink-0">
                  <svg className="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
                    <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                  </svg>
                </div>
                <div className="ml-3">
                  <p className="text-sm text-red-700">{error}</p>
                </div>
              </div>
            </div>
          )}

          {/* Results */}
          {results && (
            <>
              <div className="bg-white rounded-lg shadow-md p-4">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-gray-500">Query completed in {results.latency_ms}ms</p>
                    <p className="text-lg font-semibold text-gray-900">
                      {results.result_count} events {results.total_matches && `of ${results.total_matches} total`}
                    </p>
                  </div>
                  {results.result_count > 0 && (
                    <div className="text-sm text-gray-500">
                      Request ID: {results.request_id}
                    </div>
                  )}
                </div>
              </div>

              <EventsTable 
                events={results.results || []} 
                totalMatches={results.total_matches}
                onEventClick={setSelectedEvent}
              />
            </>
          )}
        </div>
      </main>

      {/* Event Detail Modal */}
      {selectedEvent && (
        <EventDetailModal 
          event={selectedEvent} 
          onClose={() => setSelectedEvent(null)} 
        />
      )}
    </div>
  );
}
