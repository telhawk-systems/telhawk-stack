import { useState, useEffect } from 'react';
import { Layout } from '../components/Layout';
import { useScope } from '../components/ScopeProvider';
import { apiClient } from '../services/api';
import { HECToken } from '../types';

// Extract timestamp from UUIDv7 (first 48 bits are milliseconds since Unix epoch)
function getDateFromUUIDv7(uuid: string): Date {
  const hex = uuid.replace(/-/g, '').substring(0, 12);
  const ms = parseInt(hex, 16);
  return new Date(ms);
}

export function TokensPage() {
  const { scope, hasClientSelected } = useScope();
  const [tokens, setTokens] = useState<HECToken[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [tokenName, setTokenName] = useState('');
  const [newlyCreatedToken, setNewlyCreatedToken] = useState<HECToken | null>(null);
  const [copiedTokenId, setCopiedTokenId] = useState<string | null>(null);

  useEffect(() => {
    loadTokens();
  }, []);

  const loadTokens = async () => {
    try {
      setLoading(true);
      const data = await apiClient.listHECTokens();
      setTokens(data);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load HEC tokens');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateToken = async () => {
    if (!tokenName) {
      setError('Token name is required');
      return;
    }

    if (!scope.client_id) {
      setError('A client must be selected to create HEC tokens');
      return;
    }

    try {
      const token = await apiClient.createHECToken(tokenName, scope.client_id);
      setNewlyCreatedToken(token);
      await loadTokens();
      setShowCreateForm(false);
      setTokenName('');
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create HEC token');
    }
  };

  const handleRevokeToken = async (tokenId: string) => {
    if (!confirm('Are you sure you want to revoke this token? This action cannot be undone.')) return;

    try {
      await apiClient.revokeHECToken(tokenId);
      await loadTokens();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to revoke token');
    }
  };

  const copyToClipboard = (text: string, tokenId: string) => {
    navigator.clipboard.writeText(text).then(() => {
      setCopiedTokenId(tokenId);
      setTimeout(() => setCopiedTokenId(null), 2000);
    });
  };

  if (loading) {
    return (
      <Layout>
        <div className="flex items-center justify-center h-64">
          <div className="text-gray-600">Loading HEC tokens...</div>
        </div>
      </Layout>
    );
  }

  const clientSelected = hasClientSelected();

  return (
    <Layout>
      <div className="mb-6 flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">HEC Token Management</h1>
          <p className="text-gray-600 mt-1">Manage HTTP Event Collector tokens for data ingestion</p>
        </div>
        <button
          onClick={() => setShowCreateForm(true)}
          disabled={!clientSelected}
          className={`px-4 py-2 rounded-md font-medium transition-colors ${
            clientSelected
              ? 'bg-blue-600 text-white hover:bg-blue-700'
              : 'bg-gray-300 text-gray-500 cursor-not-allowed'
          }`}
          title={!clientSelected ? 'Select a client to create tokens' : undefined}
        >
          Create New Token
        </button>
      </div>

      {!clientSelected && (
        <div className="mb-4 p-4 bg-yellow-50 border border-yellow-200 rounded-lg">
          <p className="text-yellow-800">
            Select a client from the sidebar to create new HEC tokens. Existing tokens are shown below.
          </p>
        </div>
      )}

      {error && (
        <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-lg">
          <p className="text-red-800">{error}</p>
        </div>
      )}

      {newlyCreatedToken && (
        <div className="mb-6 bg-green-50 border border-green-200 rounded-lg p-6">
          <div className="flex justify-between items-start mb-2">
            <h2 className="text-xl font-semibold text-green-900">Token Created Successfully!</h2>
            <button
              onClick={() => setNewlyCreatedToken(null)}
              className="text-green-600 hover:text-green-800 font-bold text-xl"
            >
              ×
            </button>
          </div>
          <p className="text-sm text-green-700 mb-3">
            Save this token now. For security reasons, it won't be displayed again.
          </p>
          <div className="bg-white border border-green-300 rounded-md p-3 flex justify-between items-center">
            <code className="text-sm font-mono text-gray-800 break-all">{newlyCreatedToken.token}</code>
            <button
              onClick={() => copyToClipboard(newlyCreatedToken.token, newlyCreatedToken.id)}
              className="ml-4 px-3 py-1 bg-green-600 text-white rounded hover:bg-green-700 text-sm font-medium whitespace-nowrap"
            >
              {copiedTokenId === newlyCreatedToken.id ? 'Copied!' : 'Copy'}
            </button>
          </div>
        </div>
      )}

      {showCreateForm && (
        <div className="mb-6 bg-white shadow-md rounded-lg p-6">
          <h2 className="text-xl font-semibold text-gray-900 mb-4">Create New HEC Token</h2>
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-1">Client</label>
            <div className="px-3 py-2 bg-gray-100 border border-gray-300 rounded-md text-sm text-gray-700">
              {scope.client_name || 'No client selected'}
            </div>
            <p className="text-xs text-gray-500 mt-1">
              Token will be created for the currently selected client
            </p>
          </div>
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-1">Token Name</label>
            <input
              type="text"
              value={tokenName}
              onChange={(e) => setTokenName(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="Enter a descriptive name for this token"
            />
            <p className="text-xs text-gray-500 mt-1">
              Use a name that helps you identify the source or purpose of this token
            </p>
          </div>
          <div className="flex gap-2 justify-end">
            <button
              onClick={() => {
                setShowCreateForm(false);
                setTokenName('');
                setError(null);
              }}
              className="px-4 py-2 bg-gray-300 text-gray-700 rounded-md hover:bg-gray-400 font-medium transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleCreateToken}
              className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 font-medium transition-colors"
            >
              Create Token
            </button>
          </div>
        </div>
      )}

      <div className="bg-white shadow-md rounded-lg overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Name
              </th>
              {tokens.length > 0 && tokens[0].username && (
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Owner
                </th>
              )}
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Token (masked)
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Status
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Created
              </th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                Actions
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {tokens.map((token) => (
              <tr key={token.id} className="hover:bg-gray-50">
                <td className="px-6 py-4 whitespace-nowrap">
                  <div className="text-sm font-medium text-gray-900">{token.name}</div>
                </td>
                {token.username && (
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-sm text-gray-900">{token.username}</div>
                  </td>
                )}
                <td className="px-6 py-4 whitespace-nowrap">
                  <code className="text-xs font-mono text-gray-600">
                    {token.token.substring(0, 8)}...{token.token.substring(token.token.length - 8)}
                  </code>
                </td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <span
                    className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                      token.enabled
                        ? 'bg-green-100 text-green-800'
                        : 'bg-red-100 text-red-800'
                    }`}
                  >
                    {token.enabled ? 'Active' : 'Revoked'}
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                  {getDateFromUUIDv7(token.id).toLocaleDateString()} {getDateFromUUIDv7(token.id).toLocaleTimeString()}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm">
                  {token.enabled ? (
                    <button
                      onClick={() => handleRevokeToken(token.id)}
                      className="text-red-600 hover:text-red-800 font-medium"
                    >
                      Revoke
                    </button>
                  ) : (
                    <span className="text-gray-400">Revoked</span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {tokens.length === 0 && (
          <div className="text-center py-8 text-gray-500">
            No HEC tokens found. Create one to start ingesting data.
          </div>
        )}
      </div>

      <div className="mt-6 bg-blue-50 border border-blue-200 rounded-lg p-4">
        <h3 className="text-sm font-semibold text-blue-900 mb-2">About HEC Tokens</h3>
        <p className="text-sm text-blue-800">
          HEC (HTTP Event Collector) tokens are used to authenticate data ingestion requests.
          Include the token in the Authorization header as "Splunk &lt;token&gt;" when sending events
          to the /services/collector/event endpoint.
        </p>
      </div>
    </Layout>
  );
}
