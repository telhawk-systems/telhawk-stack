import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { apiClient } from '../services/api';
import { Layout } from '../components/Layout';
import { EventsTable } from '../components/EventsTable';
import { EventDetailModal } from '../components/EventDetailModal';

export function SavedSearchDetailPage() {
  const { id } = useParams();
  const [item, setItem] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [runOpen, setRunOpen] = useState(false);
  const [runResults, setRunResults] = useState<any>(null);
  const [runEvents, setRunEvents] = useState<any[]>([]);
  const [selectedEvent, setSelectedEvent] = useState<any>(null);

  const load = async () => {
    if (!id) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/query/api/v1/saved-searches/${id}`, { headers: { 'Accept': 'application/vnd.api+json' }, credentials: 'include' });
      if (res.ok) {
        const data = await res.json();
        setItem(data.data);
      }
    } finally { setLoading(false); }
  };

  useEffect(() => { load(); // eslint-disable-next-line
  }, [id]);

  if (!id) return null;

  const a = item?.attributes || {};
  const state = a.hidden_at ? 'hidden' : (a.disabled_at ? 'disabled' : 'active');

  return (
    <Layout>
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-2xl font-bold text-gray-800">Saved Search: {a.name || id}</h2>
        <div className="space-x-2">
          <button onClick={() => apiClient.savedSearchAction(id!, 'run').then((r: any)=>{ setRunResults(r); setRunEvents((r as any).events || (r as any).results || []); setRunOpen(true);}).catch(console.error)} className="px-3 py-1 bg-blue-600 text-white rounded">Run</button>
          {state !== 'disabled' && <button onClick={() => apiClient.savedSearchAction(id!, 'disable').then(load)} className="px-3 py-1 bg-yellow-600 text-white rounded">Disable</button>}
          {state === 'disabled' && <button onClick={() => apiClient.savedSearchAction(id!, 'enable').then(load)} className="px-3 py-1 bg-green-600 text-white rounded">Enable</button>}
          {!a.hidden_at && <button onClick={() => apiClient.savedSearchAction(id!, 'hide').then(load)} className="px-3 py-1 bg-gray-600 text-white rounded">Hide</button>}
          {a.hidden_at && <button onClick={() => apiClient.savedSearchAction(id!, 'unhide').then(load)} className="px-3 py-1 bg-gray-600 text-white rounded">Unhide</button>}
        </div>
      </div>
      {loading && <div className="text-gray-500">Loading…</div>}
      {!loading && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="bg-white rounded shadow p-4">
            <h3 className="font-semibold mb-2">Details</h3>
            <dl className="text-sm text-gray-700 space-y-1">
              <div><dt className="inline text-gray-500">ID:</dt> <dd className="inline">{id}</dd></div>
              <div><dt className="inline text-gray-500">Version:</dt> <dd className="inline">{a.version_id}</dd></div>
              <div><dt className="inline text-gray-500">State:</dt> <dd className="inline">{state}</dd></div>
              <div><dt className="inline text-gray-500">Created:</dt> <dd className="inline">{String(a.created_at || '')}</dd></div>
            </dl>
          </div>
          <div className="bg-white rounded shadow p-4">
            <h3 className="font-semibold mb-2">Query (JSON)</h3>
            <pre className="text-xs bg-gray-50 p-3 rounded overflow-x-auto">{JSON.stringify(a.query || {}, null, 2)}</pre>
          </div>
        </div>
      )}
      {runOpen && (
        <div className="fixed inset-0 bg-black/40 flex items-center justify-center z-50">
          <div className="bg-white rounded shadow-xl w-11/12 md:w-3/4 lg:w-2/3 max-h-[80vh] overflow-auto">
            <div className="p-4 border-b flex justify-between items-center">
              <h3 className="font-semibold">Run Results</h3>
              <button onClick={() => setRunOpen(false)} className="text-gray-600 hover:text-gray-900">✕</button>
            </div>
            <div className="p-4">
              <EventsTable
                events={runEvents as any}
                totalMatches={(runResults && (runResults.total_matches || runResults.total)) || undefined}
                onEventClick={(e) => setSelectedEvent(e)}
              />
            </div>
          </div>
        </div>
      )}
      {selectedEvent && (
        <EventDetailModal
          event={selectedEvent}
          onClose={() => setSelectedEvent(null)}
        />
      )}
    </Layout>
  );
}
