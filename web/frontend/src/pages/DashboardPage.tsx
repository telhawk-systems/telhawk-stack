import React, { useState } from 'react';
import { useAuth } from '../components/AuthProvider';
import { apiClient } from '../services/api';

export function DashboardPage() {
  const { user, logout } = useAuth();
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const data = await apiClient.search(query);
      setResults(data);
    } catch (err) {
      setError('Search failed');
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
    <div style={{ padding: '20px' }}>
      <header style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '20px' }}>
        <h1>TelHawk Dashboard</h1>
        <div>
          <span style={{ marginRight: '15px' }}>
            User: {user?.user_id} | Roles: {user?.roles.join(', ')}
          </span>
          <button onClick={handleLogout}>Logout</button>
        </div>
      </header>

      <div style={{ marginBottom: '30px' }}>
        <h2>Search</h2>
        <form onSubmit={handleSearch}>
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Enter search query..."
            style={{ width: '70%', padding: '10px', marginRight: '10px' }}
            disabled={loading}
          />
          <button type="submit" disabled={loading}>
            {loading ? 'Searching...' : 'Search'}
          </button>
        </form>
        {error && <div style={{ color: 'red', marginTop: '10px' }}>{error}</div>}
      </div>

      {results && (
        <div>
          <h3>Results ({results.total} total)</h3>
          <div style={{ border: '1px solid #ccc', padding: '10px', maxHeight: '600px', overflow: 'auto' }}>
            <pre>{JSON.stringify(results, null, 2)}</pre>
          </div>
        </div>
      )}
    </div>
  );
}
