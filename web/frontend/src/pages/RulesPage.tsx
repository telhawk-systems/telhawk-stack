import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Layout } from '../components/Layout';
import { DetectionSchema } from '../types/rules';
import { CreateRuleModal } from '../components/rules/CreateRuleModal';
import { EditRuleModal } from '../components/rules/EditRuleModal';

export function RulesPage() {
  const navigate = useNavigate();
  const [schemas, setSchemas] = useState<DetectionSchema[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [severityFilter, setSeverityFilter] = useState<string>('');
  const [titleFilter, setTitleFilter] = useState<string>('');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingSchema, setEditingSchema] = useState<DetectionSchema | null>(null);

  const fetchSchemas = async () => {
    try {
      setLoading(true);
      setError(null);

      const params = new URLSearchParams({
        page: currentPage.toString(),
        limit: '20',
      });

      if (severityFilter) params.append('severity', severityFilter);
      if (titleFilter) params.append('title', titleFilter);

      const response = await fetch(`/api/rules/schemas?${params}`);

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const json = await response.json();
      // Parse JSON:API format
      const schemas = (json.data || []).map((resource: any) => {
        if (!resource.attributes.view) {
          console.error('Schema missing view:', resource);
        }
        return {
          id: resource.id,
          ...resource.attributes,
        };
      });
      setSchemas(schemas);
      setTotalPages(json.meta?.pagination?.total_pages || 1);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch detection schemas');
      console.error('Error fetching schemas:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchSchemas();
  }, [currentPage, severityFilter, titleFilter]);

  const handleDisableSchema = async (id: string, disabled: boolean) => {
    try {
      const endpoint = disabled ? 'enable' : 'disable';
      const response = await fetch(`/api/rules/schemas/${id}/${endpoint}`, {
        method: 'PUT',
      });

      if (!response.ok) {
        throw new Error(`Failed to ${endpoint} schema`);
      }

      await fetchSchemas();
    } catch (err) {
      console.error('Error updating schema:', err);
      alert(`Failed to update schema: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  };

  const handleDeleteSchema = async (id: string) => {
    if (!confirm('Are you sure you want to delete this detection schema? This cannot be undone.')) {
      return;
    }

    try {
      const response = await fetch(`/api/rules/schemas/${id}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        throw new Error('Failed to delete schema');
      }

      await fetchSchemas();
    } catch (err) {
      console.error('Error deleting schema:', err);
      alert(`Failed to delete schema: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
  };

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical': return 'bg-purple-100 text-purple-800 border-purple-200';
      case 'high': return 'bg-red-100 text-red-800 border-red-200';
      case 'medium': return 'bg-yellow-100 text-yellow-800 border-yellow-200';
      case 'low': return 'bg-blue-100 text-blue-800 border-blue-200';
      case 'informational': return 'bg-gray-100 text-gray-800 border-gray-200';
      default: return 'bg-gray-100 text-gray-800 border-gray-200';
    }
  };

  return (
    <Layout>
      <div className="space-y-6">
        {/* Header */}
        <div className="bg-white shadow rounded-lg p-6">
          <div className="flex justify-between items-center">
            <div>
              <h1 className="text-2xl font-bold text-gray-900">Detection Rules</h1>
              <p className="text-sm text-gray-600 mt-1">
                Manage detection schemas for automated threat detection
              </p>
            </div>
            <button
              className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 font-medium transition-colors"
              onClick={() => setShowCreateModal(true)}
            >
              Create Rule
            </button>
          </div>

          {/* Filters */}
          <div className="mt-6 grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Search by Title
              </label>
              <input
                type="text"
                value={titleFilter}
                onChange={(e) => {
                  setTitleFilter(e.target.value);
                  setCurrentPage(1);
                }}
                placeholder="Filter by title..."
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Severity
              </label>
              <select
                value={severityFilter}
                onChange={(e) => {
                  setSeverityFilter(e.target.value);
                  setCurrentPage(1);
                }}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">All Severities</option>
                <option value="critical">Critical</option>
                <option value="high">High</option>
                <option value="medium">Medium</option>
                <option value="low">Low</option>
                <option value="informational">Informational</option>
              </select>
            </div>
            <div className="flex items-end">
              <button
                onClick={() => {
                  setTitleFilter('');
                  setSeverityFilter('');
                  setCurrentPage(1);
                }}
                className="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300 transition-colors"
              >
                Clear Filters
              </button>
            </div>
          </div>
        </div>

        {/* Rules List */}
        <div className="bg-white shadow rounded-lg overflow-hidden">
          {loading ? (
            <div className="p-12 text-center">
              <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
              <p className="mt-4 text-gray-600">Loading detection rules...</p>
            </div>
          ) : error ? (
            <div className="p-8 text-center">
              <div className="text-red-600 mb-2">
                <svg className="inline-block w-12 h-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
              <p className="text-lg font-semibold text-gray-900">Error loading rules</p>
              <p className="text-sm text-gray-600 mt-1">{error}</p>
              <button
                onClick={fetchSchemas}
                className="mt-4 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
              >
                Retry
              </button>
            </div>
          ) : schemas.length === 0 ? (
            <div className="p-12 text-center">
              <svg className="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
              <h3 className="mt-2 text-sm font-medium text-gray-900">No detection rules</h3>
              <p className="mt-1 text-sm text-gray-500">
                Get started by creating your first detection rule.
              </p>
              <div className="mt-6">
                <button
                  onClick={() => setShowCreateModal(true)}
                  className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
                >
                  Create Rule
                </button>
              </div>
            </div>
          ) : (
            <>
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Rule
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Severity
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Status
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Version
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
                  {schemas.map((schema) => (
                    <tr key={schema.version_id} className="hover:bg-gray-50">
                      <td className="px-6 py-4">
                        <div>
                          <div className="text-sm font-medium text-gray-900">
                            {schema.view?.title || 'Untitled Rule'}
                          </div>
                          {schema.view?.description_template && (
                            <div className="text-xs text-gray-500 mt-1">
                              {schema.view.description_template}
                            </div>
                          )}
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className={`px-2 py-1 text-xs font-medium rounded-full border ${getSeverityColor(schema.view.severity)}`}>
                          {schema.view.severity}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className={`px-2 py-1 text-xs font-medium rounded-full ${
                          schema.disabled_at
                            ? 'bg-gray-100 text-gray-600'
                            : 'bg-green-100 text-green-800'
                        }`}>
                          {schema.disabled_at ? 'Disabled' : 'Active'}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        v{schema.version}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {new Date(schema.created_at).toLocaleDateString()}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium space-x-2">
                        <button
                          onClick={() => navigate(`/rules/${schema.id}`)}
                          className="text-blue-600 hover:text-blue-900"
                        >
                          View
                        </button>
                        <button
                          onClick={() => setEditingSchema(schema)}
                          className="text-indigo-600 hover:text-indigo-900"
                        >
                          Edit
                        </button>
                        <button
                          onClick={() => handleDisableSchema(schema.id, !!schema.disabled_at)}
                          className={schema.disabled_at ? 'text-green-600 hover:text-green-900' : 'text-yellow-600 hover:text-yellow-900'}
                        >
                          {schema.disabled_at ? 'Enable' : 'Disable'}
                        </button>
                        <button
                          onClick={() => handleDeleteSchema(schema.id)}
                          className="text-red-600 hover:text-red-900"
                        >
                          Delete
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>

              {/* Pagination */}
              {totalPages > 1 && (
                <div className="bg-gray-50 px-6 py-4 flex items-center justify-between border-t border-gray-200">
                  <div className="text-sm text-gray-700">
                    Page {currentPage} of {totalPages}
                  </div>
                  <div className="flex space-x-2">
                    <button
                      onClick={() => setCurrentPage(Math.max(1, currentPage - 1))}
                      disabled={currentPage === 1}
                      className="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      Previous
                    </button>
                    <button
                      onClick={() => setCurrentPage(Math.min(totalPages, currentPage + 1))}
                      disabled={currentPage === totalPages}
                      className="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      Next
                    </button>
                  </div>
                </div>
              )}
            </>
          )}
        </div>
      </div>

      {/* Create Rule Modal */}
      <CreateRuleModal
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        onSuccess={fetchSchemas}
      />

      {/* Edit Rule Modal */}
      <EditRuleModal
        isOpen={!!editingSchema}
        schema={editingSchema}
        onClose={() => setEditingSchema(null)}
        onSuccess={fetchSchemas}
      />
    </Layout>
  );
}
