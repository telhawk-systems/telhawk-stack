import { useState } from 'react';
import { Layout } from '../components/Layout';
import { SearchConsole } from '../components/SearchConsole';
import { EventsTable } from '../components/EventsTable';
import { EventDetailModal } from '../components/EventDetailModal';
import { DashboardOverview } from '../components/dashboard/DashboardOverview';
import { apiClient } from '../services/api';
import { Query } from '../types/query';

type TabType = 'overview' | 'search';

export function DashboardPage() {
  const [activeTab, setActiveTab] = useState<TabType>('overview');
  const [results, setResults] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [saveBusy, setSaveBusy] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState('');
  const [selectedEvent, setSelectedEvent] = useState<any>(null);
  const [currentQuery, setCurrentQuery] = useState<Query | null>(null);
  const [allEvents, setAllEvents] = useState<any[]>([]);

  const handleSearch = async (query: Query) => {
    setError('');
    setLoading(true);
    setResults(null);
    setAllEvents([]);
    setCurrentQuery(query);

    try {
      const data = await apiClient.executeQuery(query);

      // Handle both response formats (results vs events)
      const events = (data as any).results || data.events || [];
      const total = (data as any).total_matches || data.total || 0;

      // Map response to match expected format
      const results = {
        results: events,
        result_count: events.length,
        total_matches: total,
        latency_ms: data.took || (data as any).latency_ms || 0,
        cursor: (data as any).search_after?.[0]?.toString() || data.cursor,
        request_id: (data as any).request_id || 'query-' + Date.now(),
      };

      setResults(results);
      setAllEvents(events);
    } catch (err) {
      setError('Search failed. Please try again.');
      console.error('Search error:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async (query: Query) => {
    const name = window.prompt('Name for this saved search?');
    if (!name) return;
    try {
      setSaveBusy(true);
      await apiClient.createSavedSearch(name, query as any);
      alert('Saved search created');
    } catch (e) {
      console.error(e);
      alert('Failed to save search');
    } finally { setSaveBusy(false); }
  };

  const handleLoadMore = async () => {
    if (!results?.cursor || loadingMore || !currentQuery) return;

    setLoadingMore(true);
    setError('');

    try {
      // Update query with cursor for next page
      const nextPageQuery: Query = {
        ...currentQuery,
        cursor: results.cursor,
      };

      const data = await apiClient.executeQuery(nextPageQuery);

      // Handle both response formats (results vs events)
      const events = (data as any).results || data.events || [];
      const total = (data as any).total_matches || data.total || 0;

      // Append new results to existing events
      const newEvents = [...allEvents, ...events];
      setAllEvents(newEvents);

      // Update results with new cursor and counts
      setResults({
        results: newEvents,
        result_count: newEvents.length,
        total_matches: total,
        latency_ms: data.took || (data as any).latency_ms || 0,
        cursor: (data as any).search_after?.[0]?.toString() || data.cursor,
        request_id: 'query-' + Date.now(),
      });
    } catch (err) {
      setError('Failed to load more results. Please try again.');
      console.error('Load more error:', err);
    } finally {
      setLoadingMore(false);
    }
  };

  return (
    <Layout>
      {/* Tab Navigation */}
      <div className="border-b border-gray-200 mb-6">
        <nav className="-mb-px flex space-x-8">
          <button
            onClick={() => setActiveTab('overview')}
            className={`${
              activeTab === 'overview'
                ? 'border-blue-500 text-blue-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            } whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm`}
          >
            Overview
          </button>
          <button
            onClick={() => setActiveTab('search')}
            className={`${
              activeTab === 'search'
                ? 'border-blue-500 text-blue-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
            } whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm`}
          >
            Search
          </button>
        </nav>
      </div>

      {/* Tab Content */}
      {activeTab === 'overview' ? (
        <DashboardOverview />
      ) : (
        <div className="space-y-6">
          {/* Search Console */}
          <div className="space-y-3">
            <SearchConsole onSearch={handleSearch} loading={loading} />
            <div>
              <button onClick={() => handleSave(currentQuery as any)} disabled={!currentQuery || saveBusy} className={`px-3 py-1 rounded ${currentQuery? 'bg-gray-800 text-white':'bg-gray-300 text-gray-500 cursor-not-allowed'}`}>
                {saveBusy? 'Saving…' : 'Save as Saved Search'}
              </button>
            </div>
          </div>

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
                events={allEvents}
                totalMatches={results.total_matches}
                onEventClick={setSelectedEvent}
                onLoadMore={results.cursor ? handleLoadMore : undefined}
                loadingMore={loadingMore}
              />
            </>
          )}
        </div>
      )}

      {/* Event Detail Modal */}
      {selectedEvent && (
        <EventDetailModal 
          event={selectedEvent} 
          onClose={() => setSelectedEvent(null)} 
        />
      )}
    </Layout>
  );
}
