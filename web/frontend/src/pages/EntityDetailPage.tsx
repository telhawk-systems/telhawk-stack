import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { apiClient } from '../services/api';
import { EntityType, getEntityIcon, getEntityTypeName, getEntityColorClass, buildEntityFilter } from '../utils/entityUtils';
import { Query, QueryResponse } from '../types/query';
import { EventsTable } from '../components/EventsTable';
import { EventDetailModal } from '../components/EventDetailModal';

interface EntityStats {
  eventCount: number;
  firstSeen: string | null;
  lastSeen: string | null;
}

export function EntityDetailPage() {
  const { entityType, entityValue } = useParams<{ entityType: string; entityValue: string }>();
  const navigate = useNavigate();
  const [events, setEvents] = useState<any[]>([]);
  const [stats, setStats] = useState<EntityStats>({ eventCount: 0, firstSeen: null, lastSeen: null });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedEvent, setSelectedEvent] = useState<any | null>(null);
  const [cursor, setCursor] = useState<string | undefined>(undefined);
  const [loadingMore, setLoadingMore] = useState(false);

  // Decode the entity value from URL
  const decodedValue = entityValue ? decodeURIComponent(entityValue) : '';

  // Validate entity type
  const validEntityTypes: EntityType[] = ['user', 'ip', 'hostname', 'process', 'file'];
  const isValidType = entityType && validEntityTypes.includes(entityType as EntityType);

  useEffect(() => {
    if (!isValidType || !decodedValue) {
      setError('Invalid entity type or value');
      setLoading(false);
      return;
    }

    loadEntityData();
  }, [entityType, entityValue]);

  const loadEntityData = async (loadMore = false) => {
    try {
      if (loadMore) {
        setLoadingMore(true);
      } else {
        setLoading(true);
        setError(null);
      }

      // Build the entity filter using the query language
      const entityFilter = buildEntityFilter({
        type: entityType as EntityType,
        value: decodedValue,
        displayName: decodedValue,
      });

      // Debug logging
      console.log('Entity filter:', JSON.stringify(entityFilter, null, 2));

      // Build the query
      const query: Query = {
        timeRange: {
          last: '30d', // Last 30 days by default
        },
        sort: [
          {
            field: '.time',
            order: 'desc',
          },
        ],
        limit: 50,
        ...(loadMore && cursor && { cursor }),
      };

      // Only add filter if it exists
      if (entityFilter) {
        query.filter = entityFilter;
      }

      console.log('Entity query:', JSON.stringify(query, null, 2));

      // Execute the query
      const response: QueryResponse = await apiClient.executeQuery(query);

      console.log('Entity query RAW response:', response);
      console.log('Response keys:', Object.keys(response));
      console.log('Response.results?', (response as any).results?.length);
      console.log('Response.events?', response.events?.length);
      console.log('Response.total_matches?', (response as any).total_matches);
      console.log('Response.total?', response.total);

      // Handle both response formats (old search API vs new query API)
      const responseEvents = (response as any).results || response.events || [];
      const total = (response as any).total_matches || response.total || 0;
      const nextCursor = (response as any).search_after?.[0]?.toString() || response.cursor;

      console.log('FINAL extracted data:', {
        responseEventsLength: responseEvents.length,
        total: total,
        nextCursor: nextCursor,
        firstEventTime: responseEvents[0]?.time
      });

      if (loadMore) {
        setEvents((prev) => [...prev, ...responseEvents]);
      } else {
        setEvents(responseEvents);

        // Calculate stats from the results
        const eventCount = total;
        const eventList = responseEvents;

        let firstSeen: string | null = null;
        let lastSeen: string | null = null;

        if (eventList.length > 0) {
          // Events are sorted by time desc, so last in list is first seen
          lastSeen = eventList[0].time;
          firstSeen = eventList[eventList.length - 1].time;

          // If we have fewer events than total, we need to estimate first seen
          if (eventList.length < eventCount) {
            // Query for oldest event (ascending sort)
            const oldestQuery: Query = {
              timeRange: {
                last: '30d',
              },
              sort: [
                {
                  field: '.time',
                  order: 'asc',
                },
              ],
              limit: 1,
            };

            if (entityFilter) {
              oldestQuery.filter = entityFilter;
            }

            const oldestResponse: QueryResponse = await apiClient.executeQuery(oldestQuery);
            const oldestEvents = (oldestResponse as any).results || oldestResponse.events || [];
            if (oldestEvents.length > 0) {
              firstSeen = oldestEvents[0].time;
            }
          }
        }

        setStats({
          eventCount,
          firstSeen,
          lastSeen,
        });
      }

      setCursor(nextCursor);
    } catch (err) {
      console.error('Failed to load entity data:', err);
      setError(err instanceof Error ? err.message : 'Failed to load entity data');
    } finally {
      setLoading(false);
      setLoadingMore(false);
    }
  };

  const handleLoadMore = () => {
    if (cursor && !loadingMore) {
      loadEntityData(true);
    }
  };

  const handleEventClick = (event: any) => {
    setSelectedEvent(event);
  };

  const handleCloseModal = () => {
    setSelectedEvent(null);
  };

  if (!isValidType) {
    return (
      <div className="min-h-screen bg-gray-50 p-6">
        <div className="max-w-7xl mx-auto">
          <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded-lg">
            <p className="font-bold">Invalid Entity Type</p>
            <p>Supported types: user, ip, hostname, process, file</p>
            <button
              onClick={() => navigate('/')}
              className="mt-2 text-blue-600 hover:text-blue-800 underline"
            >
              Return to Dashboard
            </button>
          </div>
        </div>
      </div>
    );
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 p-6">
        <div className="max-w-7xl mx-auto">
          <div className="bg-white rounded-lg shadow-md p-8 text-center">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
            <p className="text-gray-600 mt-4">Loading entity data...</p>
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen bg-gray-50 p-6">
        <div className="max-w-7xl mx-auto">
          <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded-lg">
            <p className="font-bold">Error Loading Entity</p>
            <p>{error}</p>
            <button
              onClick={() => navigate('/')}
              className="mt-2 text-blue-600 hover:text-blue-800 underline"
            >
              Return to Dashboard
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow-sm border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <button
                onClick={() => navigate('/')}
                className="text-gray-600 hover:text-gray-900"
                title="Back to Dashboard"
              >
                ← Back
              </button>
              <div className="flex items-center space-x-3">
                <span className="text-4xl">{getEntityIcon(entityType as EntityType)}</span>
                <div>
                  <h1 className="text-2xl font-bold text-gray-900">Entity Investigation</h1>
                  <p className="text-sm text-gray-600">
                    {getEntityTypeName(entityType as EntityType)}: {decodedValue}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="max-w-7xl mx-auto px-6 py-6">
        {/* Entity Stats */}
        <div className="bg-white rounded-lg shadow-md p-6 mb-6">
          <h2 className="text-lg font-semibold text-gray-800 mb-4">Entity Statistics</h2>
          <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
            {/* Entity Type Badge */}
            <div className="flex flex-col">
              <span className="text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">
                Entity Type
              </span>
              <span className={`px-3 py-2 rounded-full text-sm font-medium border inline-block ${getEntityColorClass(entityType as EntityType)}`}>
                {getEntityIcon(entityType as EntityType)} {getEntityTypeName(entityType as EntityType)}
              </span>
            </div>

            {/* Event Count */}
            <div className="flex flex-col">
              <span className="text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">
                Total Events
              </span>
              <span className="text-2xl font-bold text-gray-900">
                {stats.eventCount.toLocaleString()}
              </span>
            </div>

            {/* First Seen */}
            <div className="flex flex-col">
              <span className="text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">
                First Seen
              </span>
              <span className="text-sm text-gray-900">
                {stats.firstSeen ? new Date(stats.firstSeen).toLocaleString() : 'N/A'}
              </span>
            </div>

            {/* Last Seen */}
            <div className="flex flex-col">
              <span className="text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">
                Last Seen
              </span>
              <span className="text-sm text-gray-900">
                {stats.lastSeen ? new Date(stats.lastSeen).toLocaleString() : 'N/A'}
              </span>
            </div>
          </div>
        </div>

        {/* Entity Timeline */}
        <div className="mb-6">
          <h2 className="text-lg font-semibold text-gray-800 mb-4 bg-white rounded-lg shadow-md px-6 py-4">
            Event Timeline
            <span className="text-sm font-normal text-gray-600 ml-2">
              (Last 30 days, sorted by most recent)
            </span>
          </h2>
          {events.length > 0 ? (
            <EventsTable
              events={events}
              totalMatches={stats.eventCount}
              onEventClick={handleEventClick}
              onLoadMore={cursor ? handleLoadMore : undefined}
              loadingMore={loadingMore}
            />
          ) : (
            <div className="bg-white rounded-lg shadow-md p-8 text-center">
              <p className="text-gray-500 text-lg">No events found for this entity</p>
              <p className="text-gray-400 text-sm mt-2">Try adjusting the time range or check if data is being ingested</p>
            </div>
          )}
        </div>
      </div>

      {/* Event Details Modal */}
      {selectedEvent && (
        <EventDetailModal event={selectedEvent} onClose={handleCloseModal} />
      )}
    </div>
  );
}
