import { useEffect, useState } from 'react';
import { Layout } from '../components/Layout';
import { apiClient } from '../services/api';
import { EventsTable } from '../components/EventsTable';
import { EventDetailModal } from '../components/EventDetailModal';

export function EventsPage() {
  const [query, setQuery] = useState<string>('');
  const [events, setEvents] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>('');
  const [pageSize] = useState<number>(20);
  const [cursor, setCursor] = useState<string | undefined>(undefined);
  const [cursorStack, setCursorStack] = useState<string[]>([]);
  const [selectedEvent, setSelectedEvent] = useState<any>(null);

  const load = async (opts?: { next?: boolean; prev?: boolean }) => {
    try {
      setLoading(true);
      setError('');
      const res = await apiClient.listEvents({ query, size: pageSize, cursor: opts?.next ? cursor : (opts?.prev ? cursorStack[cursorStack.length-1] : undefined) });
      if (opts?.next && res.nextCursor) {
        setCursorStack(prev => [...prev, cursor || '']);
        setCursor(res.nextCursor);
      } else if (opts?.prev) {
        const copy = cursorStack.slice(0, -1);
        setCursorStack(copy);
        setCursor(copy.length ? copy[copy.length-1] : undefined);
      } else {
        setCursor(res.nextCursor);
        setCursorStack([]);
      }
      setEvents(res.events);
    } catch (e: any) {
      setError(e?.message || 'Failed to load events');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); // eslint-disable-next-line
  }, []);

  return (
    <Layout>
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-2xl font-bold text-gray-800">Events</h2>
        <div className="flex items-center gap-2">
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Query (e.g., severity:high AND class_uid:3002)"
            className="px-3 py-2 border rounded w-96"
          />
          <button onClick={() => load()} className="px-3 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">Search</button>
        </div>
      </div>

      {error && (
        <div className="bg-red-50 border-l-4 border-red-500 p-3 rounded mb-4 text-sm text-red-700">{error}</div>
      )}

      <div className="bg-white rounded shadow">
        {loading ? (
          <div className="p-6 text-gray-500">Loading…</div>
        ) : (
          <div className="p-2">
            <EventsTable
              events={events as any}
              onEventClick={(e) => setSelectedEvent(e)}
            />
          </div>
        )}
      </div>

      <div className="flex justify-between items-center mt-4">
        <div className="text-sm text-gray-600">Page size: {pageSize}</div>
        <div className="space-x-2">
          <button disabled={cursorStack.length===0} onClick={() => load({ prev: true })} className={`px-3 py-1 rounded ${cursorStack.length>0? 'bg-gray-200':'bg-gray-100 opacity-50'}`}>Prev</button>
          <button disabled={!cursor} onClick={() => load({ next: true })} className={`px-3 py-1 rounded ${cursor? 'bg-gray-200':'bg-gray-100 opacity-50'}`}>Next</button>
        </div>
      </div>

      {selectedEvent && (
        <EventDetailModal event={selectedEvent} onClose={() => setSelectedEvent(null)} />
      )}
    </Layout>
  );
}
