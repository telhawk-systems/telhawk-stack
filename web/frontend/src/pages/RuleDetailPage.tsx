import { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { Layout } from '../components/Layout';
import { DetectionSchema, DetectionSchemaVersionHistory } from '../types/rules';

export function RuleDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [schema, setSchema] = useState<DetectionSchema | null>(null);
  const [versionHistory, setVersionHistory] = useState<DetectionSchemaVersionHistory | null>(null);
  const [selectedVersionId, setSelectedVersionId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'overview' | 'model' | 'view' | 'controller' | 'versions'>('overview');

  useEffect(() => {
    if (id) {
      fetchSchemaDetails();
      fetchVersionHistory();
    }
  }, [id]);

  useEffect(() => {
    if (selectedVersionId && selectedVersionId !== schema?.version_id) {
      fetchSpecificVersion(selectedVersionId);
    } else if (selectedVersionId === null && id) {
      fetchSchemaDetails();
    }
  }, [selectedVersionId]);

  const fetchSchemaDetails = async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await fetch(`/api/rules/schemas/${id}`);

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const json = await response.json();
      // Parse JSON:API format
      const data = { id: json.data.id, ...json.data.attributes };
      setSchema(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch schema details');
      console.error('Error fetching schema:', err);
    } finally {
      setLoading(false);
    }
  };

  const fetchSpecificVersion = async (versionId: string) => {
    try {
      setLoading(true);
      setError(null);
      const response = await fetch(`/api/rules/schemas/${versionId}`);

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const json = await response.json();
      // Parse JSON:API format
      const data = { id: json.data.id, ...json.data.attributes };
      setSchema(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch version details');
      console.error('Error fetching version:', err);
    } finally {
      setLoading(false);
    }
  };

  const fetchVersionHistory = async () => {
    try {
      const response = await fetch(`/api/rules/schemas/${id}/versions`);

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const json = await response.json();
      // Parse JSON:API format
      const versions = (json.data || []).map((resource: any) => ({
        version_id: resource.id,
        ...resource.attributes,
      }));
      setVersionHistory({ id: id!, title: '', versions });
    } catch (err) {
      console.error('Error fetching version history:', err);
    }
  };

  const handleDisableSchema = async () => {
    if (!schema) return;

    try {
      const endpoint = schema.disabled_at ? 'enable' : 'disable';
      const response = await fetch(`/api/rules/schemas/${schema.id}/${endpoint}`, {
        method: 'PUT',
      });

      if (!response.ok) {
        throw new Error(`Failed to ${endpoint} schema`);
      }

      await fetchSchemaDetails();
      await fetchVersionHistory();
    } catch (err) {
      console.error('Error updating schema:', err);
      alert(`Failed to update schema: ${err instanceof Error ? err.message : 'Unknown error'}`);
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

  if (loading && !schema) {
    return (
      <Layout>
        <div className="p-12 text-center">
          <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          <p className="mt-4 text-gray-600">Loading rule details...</p>
        </div>
      </Layout>
    );
  }

  if (error) {
    return (
      <Layout>
        <div className="p-8 text-center">
          <div className="text-red-600 mb-2">
            <svg className="inline-block w-12 h-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>
          <p className="text-lg font-semibold text-gray-900">Error loading rule</p>
          <p className="text-sm text-gray-600 mt-1">{error}</p>
          <button
            onClick={() => navigate('/rules')}
            className="mt-4 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
          >
            Back to Rules
          </button>
        </div>
      </Layout>
    );
  }

  if (!schema) {
    return (
      <Layout>
        <div className="p-8 text-center">
          <p className="text-gray-600">Rule not found</p>
          <button
            onClick={() => navigate('/rules')}
            className="mt-4 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
          >
            Back to Rules
          </button>
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="space-y-6">
        {/* Header */}
        <div className="bg-white shadow rounded-lg p-6">
          <div className="flex items-start justify-between">
            <div className="flex-1">
              <div className="flex items-center space-x-2 mb-2">
                <Link
                  to="/rules"
                  className="text-sm text-blue-600 hover:text-blue-800 flex items-center"
                >
                  <svg className="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
                  </svg>
                  Back to Rules
                </Link>
              </div>
              <h1 className="text-2xl font-bold text-gray-900">{schema.view.title}</h1>
              <div className="flex items-center space-x-4 mt-2">
                <span className={`px-2 py-1 text-xs font-medium rounded-full border ${getSeverityColor(schema.view.severity)}`}>
                  {schema.view.severity}
                </span>
                {schema.view.priority && (
                  <span className="px-2 py-1 text-xs font-medium rounded-full bg-blue-100 text-blue-800">
                    {schema.view.priority}
                  </span>
                )}
                <span className={`px-2 py-1 text-xs font-medium rounded-full ${
                  schema.disabled_at
                    ? 'bg-gray-100 text-gray-600'
                    : 'bg-green-100 text-green-800'
                }`}>
                  {schema.disabled_at ? 'Disabled' : 'Active'}
                </span>
                <span className="text-sm text-gray-500">
                  Version {schema.version}
                </span>
                {selectedVersionId && selectedVersionId !== versionHistory?.versions[0]?.version_id && (
                  <span className="px-2 py-1 text-xs font-medium rounded-full bg-yellow-100 text-yellow-800">
                    Viewing Historical Version
                  </span>
                )}
              </div>
            </div>
            <div className="flex space-x-2">
              {!selectedVersionId && (
                <button
                  onClick={handleDisableSchema}
                  className={`px-4 py-2 rounded-md font-medium ${
                    schema.disabled_at
                      ? 'bg-green-600 text-white hover:bg-green-700'
                      : 'bg-yellow-600 text-white hover:bg-yellow-700'
                  }`}
                >
                  {schema.disabled_at ? 'Enable' : 'Disable'}
                </button>
              )}
            </div>
          </div>

          {/* Metadata */}
          <div className="mt-6 grid grid-cols-1 md:grid-cols-3 gap-4 pt-6 border-t border-gray-200">
            <div>
              <p className="text-xs text-gray-500">Created</p>
              <p className="text-sm font-medium text-gray-900">
                {new Date(schema.created_at).toLocaleString()}
              </p>
            </div>
            {schema.disabled_at && (
              <div>
                <p className="text-xs text-gray-500">Disabled</p>
                <p className="text-sm font-medium text-gray-900">
                  {new Date(schema.disabled_at).toLocaleString()}
                </p>
              </div>
            )}
            <div>
              <p className="text-xs text-gray-500">Rule ID</p>
              <p className="text-sm font-mono text-gray-900">{schema.id}</p>
            </div>
          </div>
        </div>

        {/* Tabs */}
        <div className="bg-white shadow rounded-lg">
          <div className="border-b border-gray-200">
            <nav className="flex -mb-px">
              {[
                { id: 'overview', label: 'Overview' },
                { id: 'model', label: 'Model' },
                { id: 'view', label: 'View' },
                { id: 'controller', label: 'Controller' },
                { id: 'versions', label: 'Version History' },
              ].map((tab) => (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id as typeof activeTab)}
                  className={`px-6 py-3 text-sm font-medium border-b-2 ${
                    activeTab === tab.id
                      ? 'border-blue-500 text-blue-600'
                      : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                  }`}
                >
                  {tab.label}
                </button>
              ))}
            </nav>
          </div>

          <div className="p-6">
            {activeTab === 'overview' && (
              <div className="space-y-6">
                <div>
                  <h3 className="text-lg font-semibold text-gray-900 mb-2">Description</h3>
                  <p className="text-gray-700">
                    {schema.view.description_template || 'No description provided'}
                  </p>
                </div>

                {schema.view.mitre_attack && (
                  <div>
                    <h3 className="text-lg font-semibold text-gray-900 mb-2">MITRE ATT&CK</h3>
                    <div className="grid grid-cols-2 gap-4">
                      {schema.view.mitre_attack.tactics && (
                        <div>
                          <p className="text-sm font-medium text-gray-700 mb-1">Tactics</p>
                          <div className="flex flex-wrap gap-2">
                            {schema.view.mitre_attack.tactics.map((tactic) => (
                              <span key={tactic} className="px-2 py-1 text-xs bg-purple-100 text-purple-800 rounded">
                                {tactic}
                              </span>
                            ))}
                          </div>
                        </div>
                      )}
                      {schema.view.mitre_attack.techniques && (
                        <div>
                          <p className="text-sm font-medium text-gray-700 mb-1">Techniques</p>
                          <div className="flex flex-wrap gap-2">
                            {schema.view.mitre_attack.techniques.map((technique) => (
                              <span key={technique} className="px-2 py-1 text-xs bg-indigo-100 text-indigo-800 rounded">
                                {technique}
                              </span>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  </div>
                )}

                <div>
                  <h3 className="text-lg font-semibold text-gray-900 mb-2">Detection Query</h3>
                  <pre className="bg-gray-50 p-4 rounded-md overflow-x-auto text-sm font-mono">
                    {schema.controller.query}
                  </pre>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <h3 className="text-sm font-semibold text-gray-700 mb-1">Lookback Window</h3>
                    <p className="text-sm text-gray-900">{schema.controller.lookback || 'N/A'}</p>
                  </div>
                  <div>
                    <h3 className="text-sm font-semibold text-gray-700 mb-1">Evaluation Interval</h3>
                    <p className="text-sm text-gray-900">{schema.controller.evaluation_interval || 'N/A'}</p>
                  </div>
                </div>
              </div>
            )}

            {activeTab === 'model' && (
              <div>
                <h3 className="text-lg font-semibold text-gray-900 mb-4">Data Model Configuration</h3>
                <pre className="bg-gray-50 p-4 rounded-md overflow-x-auto text-sm">
                  {JSON.stringify(schema.model, null, 2)}
                </pre>
              </div>
            )}

            {activeTab === 'view' && (
              <div>
                <h3 className="text-lg font-semibold text-gray-900 mb-4">View Configuration</h3>
                <pre className="bg-gray-50 p-4 rounded-md overflow-x-auto text-sm">
                  {JSON.stringify(schema.view, null, 2)}
                </pre>
              </div>
            )}

            {activeTab === 'controller' && (
              <div>
                <h3 className="text-lg font-semibold text-gray-900 mb-4">Controller Configuration</h3>
                <pre className="bg-gray-50 p-4 rounded-md overflow-x-auto text-sm">
                  {JSON.stringify(schema.controller, null, 2)}
                </pre>
              </div>
            )}

            {activeTab === 'versions' && (
              <div>
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-lg font-semibold text-gray-900">Version History</h3>
                  {selectedVersionId && (
                    <button
                      onClick={() => setSelectedVersionId(null)}
                      className="text-sm text-blue-600 hover:text-blue-800"
                    >
                      View Latest Version
                    </button>
                  )}
                </div>
                {versionHistory && versionHistory.versions.length > 0 ? (
                  <div className="space-y-4">
                    {versionHistory.versions.map((version) => (
                      <div
                        key={version.version_id}
                        className={`border rounded-lg p-4 ${
                          version.version_id === schema.version_id
                            ? 'border-blue-500 bg-blue-50'
                            : 'border-gray-200 hover:border-gray-300'
                        }`}
                      >
                        <div className="flex items-start justify-between">
                          <div className="flex-1">
                            <div className="flex items-center space-x-2 mb-1">
                              <h4 className="text-sm font-semibold text-gray-900">
                                Version {version.version}
                              </h4>
                              {version.version_id === schema.version_id && (
                                <span className="px-2 py-0.5 text-xs font-medium bg-blue-600 text-white rounded">
                                  Current
                                </span>
                              )}
                              {version.disabled_at && (
                                <span className="px-2 py-0.5 text-xs font-medium bg-gray-400 text-white rounded">
                                  Disabled
                                </span>
                              )}
                            </div>
                            <p className="text-sm text-gray-700 font-medium mb-1">{version.title}</p>
                            <p className="text-xs text-gray-500">
                              Created {new Date(version.created_at).toLocaleString()}
                            </p>
                            {version.disabled_at && (
                              <p className="text-xs text-gray-500">
                                Disabled {new Date(version.disabled_at).toLocaleString()}
                              </p>
                            )}
                            {version.changes && (
                              <p className="text-sm text-gray-600 mt-2 italic">
                                Changes: {version.changes}
                              </p>
                            )}
                          </div>
                          <button
                            onClick={() => setSelectedVersionId(version.version_id)}
                            disabled={version.version_id === schema.version_id}
                            className={`text-sm px-3 py-1 rounded ${
                              version.version_id === schema.version_id
                                ? 'bg-gray-100 text-gray-400 cursor-not-allowed'
                                : 'bg-blue-600 text-white hover:bg-blue-700'
                            }`}
                          >
                            {version.version_id === schema.version_id ? 'Viewing' : 'View'}
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-sm text-gray-500">No version history available</p>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    </Layout>
  );
}
