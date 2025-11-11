import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { apiClient } from '../services/api';
import { Layout } from '../components/Layout';
import { EventsTable } from '../components/EventsTable';
import { EventDetailModal } from '../components/EventDetailModal';

export function SavedSearchesPage() {
  const [items, setItems] = useState<any[]>([]);
  const [meta, setMeta] = useState<any>({ page: { number: 1, size: 20 }, total: 0, next_cursor: undefined });
  const [showAll, setShowAll] = useState(false);
  const [loading, setLoading] = useState(false);
  const [cursorStack, setCursorStack] = useState<string[]>([]);
  const [cursor, setCursor] = useState<string | undefined>(undefined);
  const [runOpen, setRunOpen] = useState(false);
  const [runResults, setRunResults] = useState<any>(null);
  const [runEvents, setRunEvents] = useState<any[]>([]);
  const [selectedEvent, setSelectedEvent] = useState<any>(null);
  const pageNumber = meta?.page?.number || 1;
  const pageSize = meta?.page?.size || 20;

  const load = async (page = pageNumber, size = pageSize, all = showAll, cur?: string) => {
    try {
      setLoading(true);
      const res = await apiClient.listSavedSearches(all, page, size, cur);
      setItems(Array.isArray(res.data) ? res.data : []);
      setMeta(res.meta || { page: { number: page, size }, total: 0 });
    } catch (e) {
      console.error('Failed to load saved searches', e);
    } finally { setLoading(false); }
  };

  useEffect(() => { 
    // reset paging on filter change
    setCursorStack([]);
    setCursor(undefined);
    load(1, pageSize, showAll, undefined);
    // eslint-disable-next-line
  }, [showAll]);

  const doAction = async (id: string, action: 'disable'|'enable'|'hide'|'unhide') => {
    try { await apiClient.savedSearchAction(id, action); await load(); } catch (e) { console.error(e); }
  };

  const doRun = async (id: string) => {
    try {
      const res = await apiClient.savedSearchAction(id, 'run');
      const events = (res as any).events || (res as any).results || [];
      setRunEvents(events);
      setRunResults(res);
      setRunOpen(true);
    } catch (e) {
      console.error(e); alert('Run failed');
    }
  };

  const handleNext = async () => {
    if (!meta.next_cursor) return;
    // push current cursor onto stack (use empty string for first page)
    setCursorStack(prev => [...prev, cursor || '']);
    const next = meta.next_cursor as string;
    setCursor(next);
    await load(1, pageSize, showAll, next);
  };

  const handlePrev = async () => {
    if (cursorStack.length === 0) return;
    const copy = cursorStack.slice();
    const prevCursor = copy.pop();
    setCursorStack(copy);
    const prev = prevCursor || undefined;
    setCursor(prev);
    await load(1, pageSize, showAll, prev);
  };

  return (
    <Layout>
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-2xl font-bold text-gray-800">Saved Searches</h2>
        <label className="flex items-center space-x-2 text-sm">
          <input type="checkbox" checked={showAll} onChange={(e) => setShowAll(e.target.checked)} />
          <span>Show hidden</span>
        </label>
      </div>

      <div className="bg-white shadow rounded-lg overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Name</th>
              <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">State</th>
              <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Created</th>
              <th className="px-4 py-2"></th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {loading && (
              <tr><td className="px-4 py-4 text-gray-500" colSpan={4}>Loading…</td></tr>
            )}
            {!loading && items.length === 0 && (
              <tr><td className="px-4 py-4 text-gray-500" colSpan={4}>No saved searches</td></tr>
            )}
            {items.map((r: any) => {
              const id = r.id;
              const a = r.attributes || {};
              const state = a.hidden_at ? 'hidden' : (a.disabled_at ? 'disabled' : 'active');
              return (
                <tr key={id}>
                  <td className="px-4 py-2 text-gray-900"><Link className="text-blue-600 hover:underline" to={`/saved-searches/${id}`}>{a.name || id}</Link></td>
                  <td className="px-4 py-2">
                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${state==='active'?'bg-green-100 text-green-800': state==='disabled'?'bg-yellow-100 text-yellow-800':'bg-gray-200 text-gray-700'}`}>{state}</span>
                  </td>
                  <td className="px-4 py-2 text-gray-500">{String(a.created_at || '')}</td>
                  <td className="px-4 py-2 text-right space-x-2">
                    <button onClick={() => doRun(id)} className="px-3 py-1 bg-blue-600 text-white rounded hover:bg-blue-700">Run</button>
                    {state !== 'disabled' && <button onClick={() => doAction(id, 'disable')} className="px-3 py-1 bg-yellow-600 text-white rounded hover:bg-yellow-700">Disable</button>}
                    {state === 'disabled' && <button onClick={() => doAction(id, 'enable')} className="px-3 py-1 bg-green-600 text-white rounded hover:bg-green-700">Enable</button>}
                    {!a.hidden_at && <button onClick={() => doAction(id, 'hide')} className="px-3 py-1 bg-gray-600 text-white rounded hover:bg-gray-700">Hide</button>}
                    {a.hidden_at && <button onClick={() => doAction(id, 'unhide')} className="px-3 py-1 bg-gray-600 text-white rounded hover:bg-gray-700">Unhide</button>}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      <div className="flex justify-between items-center mt-4">
        <div className="text-sm text-gray-600">Total: {meta.total || 0}</div>
        <div className="space-x-2">
          <button disabled={cursorStack.length===0} onClick={handlePrev} className={`px-3 py-1 rounded ${cursorStack.length>0? 'bg-gray-200':'bg-gray-100 opacity-50'}`}>Prev</button>
          <button disabled={!meta.next_cursor} onClick={handleNext} className={`px-3 py-1 rounded ${meta.next_cursor? 'bg-gray-200':'bg-gray-100 opacity-50'}`}>Next</button>
        </div>
      </div>

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
