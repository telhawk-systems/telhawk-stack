import { useState, FormEvent } from 'react';
import { DetectionSchemaCreateRequest } from '../../types/rules';

interface CreateRuleModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

export function CreateRuleModal({ isOpen, onClose, onSuccess }: CreateRuleModalProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // View fields (user-facing)
  const [title, setTitle] = useState('');
  const [severity, setSeverity] = useState<'critical' | 'high' | 'medium' | 'low' | 'informational'>('medium');
  const [priority, setPriority] = useState('P3');
  const [descriptionTemplate, setDescriptionTemplate] = useState('');
  const [mitreTactics, setMitreTactics] = useState('');
  const [mitreTechniques, setMitreTechniques] = useState('');

  // Controller fields (detection logic)
  const [query, setQuery] = useState('');
  const [condition, setCondition] = useState('');
  const [lookback, setLookback] = useState('5m');
  const [evaluationInterval, setEvaluationInterval] = useState('1m');
  const [aggregationField, setAggregationField] = useState('');

  // Model fields (data processing)
  const [fields, setFields] = useState('');
  const [groupBy, setGroupBy] = useState('');
  const [timeWindow, setTimeWindow] = useState('5m');
  const [threshold, setThreshold] = useState('10');
  const [aggregation, setAggregation] = useState('count');

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      // Build the request payload
      const request: DetectionSchemaCreateRequest = {
        view: {
          title,
          severity,
          priority,
          description_template: descriptionTemplate || undefined,
          mitre_attack: (mitreTactics || mitreTechniques) ? {
            tactics: mitreTactics ? mitreTactics.split(',').map(t => t.trim()).filter(Boolean) : undefined,
            techniques: mitreTechniques ? mitreTechniques.split(',').map(t => t.trim()).filter(Boolean) : undefined,
          } : undefined,
        },
        controller: {
          query,
          condition: condition || undefined,
          lookback: lookback || undefined,
          evaluation_interval: evaluationInterval || undefined,
          aggregation_field: aggregationField || undefined,
        },
        model: {
          fields: fields ? fields.split(',').map(f => f.trim()).filter(Boolean) : undefined,
          group_by: groupBy ? groupBy.split(',').map(g => g.trim()).filter(Boolean) : undefined,
          time_window: timeWindow || undefined,
          threshold: threshold ? parseInt(threshold, 10) : undefined,
          aggregation: aggregation || undefined,
        },
      };

      const response = await fetch('/api/rules/schemas', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(request),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: 'Failed to create rule' }));
        throw new Error(errorData.message || `HTTP error! status: ${response.status}`);
      }

      // Success
      onSuccess();
      handleClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create detection rule');
      console.error('Error creating rule:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    // Reset form
    setTitle('');
    setSeverity('medium');
    setPriority('P3');
    setDescriptionTemplate('');
    setMitreTactics('');
    setMitreTechniques('');
    setQuery('');
    setCondition('');
    setLookback('5m');
    setEvaluationInterval('1m');
    setAggregationField('');
    setFields('');
    setGroupBy('');
    setTimeWindow('5m');
    setThreshold('10');
    setAggregation('count');
    setError(null);
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50 overflow-y-auto">
      <div className="bg-white rounded-lg shadow-xl max-w-4xl w-full my-8">
        {/* Header */}
        <div className="px-6 py-4 border-b border-gray-200 flex justify-between items-center">
          <h2 className="text-xl font-bold text-gray-900">Create Detection Rule</h2>
          <button
            onClick={handleClose}
            className="text-gray-400 hover:text-gray-600"
            disabled={loading}
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="px-6 py-4 max-h-[calc(100vh-200px)] overflow-y-auto">
          {error && (
            <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-md">
              <p className="text-sm text-red-800">{error}</p>
            </div>
          )}

          {/* View Section */}
          <div className="mb-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center">
              <span className="bg-blue-100 text-blue-800 px-2 py-1 rounded text-sm mr-2">View</span>
              Presentation & Display
            </h3>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="md:col-span-2">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Title <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  required
                  placeholder="e.g., SSH Brute Force Detection"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Severity <span className="text-red-500">*</span>
                </label>
                <select
                  value={severity}
                  onChange={(e) => setSeverity(e.target.value as any)}
                  required
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="critical">Critical</option>
                  <option value="high">High</option>
                  <option value="medium">Medium</option>
                  <option value="low">Low</option>
                  <option value="informational">Informational</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Priority
                </label>
                <select
                  value={priority}
                  onChange={(e) => setPriority(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="P1">P1 - Critical</option>
                  <option value="P2">P2 - High</option>
                  <option value="P3">P3 - Medium</option>
                  <option value="P4">P4 - Low</option>
                </select>
              </div>

              <div className="md:col-span-2">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Description Template
                </label>
                <input
                  type="text"
                  value={descriptionTemplate}
                  onChange={(e) => setDescriptionTemplate(e.target.value)}
                  placeholder="e.g., {{count}} failed attempts from {{src_endpoint.ip}}"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <p className="text-xs text-gray-500 mt-1">Use {'{{field}}'} for dynamic values</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  MITRE ATT&CK Tactics
                </label>
                <input
                  type="text"
                  value={mitreTactics}
                  onChange={(e) => setMitreTactics(e.target.value)}
                  placeholder="e.g., TA0006, TA0008"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <p className="text-xs text-gray-500 mt-1">Comma-separated tactic IDs</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  MITRE ATT&CK Techniques
                </label>
                <input
                  type="text"
                  value={mitreTechniques}
                  onChange={(e) => setMitreTechniques(e.target.value)}
                  placeholder="e.g., T1110.001, T1078"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <p className="text-xs text-gray-500 mt-1">Comma-separated technique IDs</p>
              </div>
            </div>
          </div>

          {/* Controller Section */}
          <div className="mb-6 border-t pt-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center">
              <span className="bg-purple-100 text-purple-800 px-2 py-1 rounded text-sm mr-2">Controller</span>
              Detection Logic
            </h3>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="md:col-span-2">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Query <span className="text-red-500">*</span>
                </label>
                <textarea
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  required
                  rows={3}
                  placeholder="e.g., class_uid:3002 AND activity_id:1 AND status:Failure"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono text-sm"
                />
                <p className="text-xs text-gray-500 mt-1">OpenSearch query syntax</p>
              </div>

              <div className="md:col-span-2">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Condition
                </label>
                <input
                  type="text"
                  value={condition}
                  onChange={(e) => setCondition(e.target.value)}
                  placeholder="e.g., count > 10"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <p className="text-xs text-gray-500 mt-1">Trigger condition (e.g., count &gt; threshold)</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Lookback Window
                </label>
                <input
                  type="text"
                  value={lookback}
                  onChange={(e) => setLookback(e.target.value)}
                  placeholder="5m"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <p className="text-xs text-gray-500 mt-1">Time range to query (e.g., 5m, 1h, 24h)</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Evaluation Interval
                </label>
                <input
                  type="text"
                  value={evaluationInterval}
                  onChange={(e) => setEvaluationInterval(e.target.value)}
                  placeholder="1m"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <p className="text-xs text-gray-500 mt-1">How often to check (e.g., 1m, 5m, 10m)</p>
              </div>

              <div className="md:col-span-2">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Aggregation Field
                </label>
                <input
                  type="text"
                  value={aggregationField}
                  onChange={(e) => setAggregationField(e.target.value)}
                  placeholder="e.g., src_endpoint.ip"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <p className="text-xs text-gray-500 mt-1">Field to group results by</p>
              </div>
            </div>
          </div>

          {/* Model Section */}
          <div className="mb-6 border-t pt-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center">
              <span className="bg-green-100 text-green-800 px-2 py-1 rounded text-sm mr-2">Model</span>
              Data Processing
            </h3>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="md:col-span-2">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Fields
                </label>
                <input
                  type="text"
                  value={fields}
                  onChange={(e) => setFields(e.target.value)}
                  placeholder="e.g., src_endpoint.ip, dst_endpoint.port, actor.user.name"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <p className="text-xs text-gray-500 mt-1">Comma-separated field names to extract</p>
              </div>

              <div className="md:col-span-2">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Group By
                </label>
                <input
                  type="text"
                  value={groupBy}
                  onChange={(e) => setGroupBy(e.target.value)}
                  placeholder="e.g., src_endpoint.ip"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <p className="text-xs text-gray-500 mt-1">Comma-separated fields for grouping</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Time Window
                </label>
                <input
                  type="text"
                  value={timeWindow}
                  onChange={(e) => setTimeWindow(e.target.value)}
                  placeholder="5m"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <p className="text-xs text-gray-500 mt-1">Aggregation time window</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Threshold
                </label>
                <input
                  type="number"
                  value={threshold}
                  onChange={(e) => setThreshold(e.target.value)}
                  placeholder="10"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <p className="text-xs text-gray-500 mt-1">Numeric threshold for alerting</p>
              </div>

              <div className="md:col-span-2">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Aggregation Type
                </label>
                <select
                  value={aggregation}
                  onChange={(e) => setAggregation(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="count">Count</option>
                  <option value="sum">Sum</option>
                  <option value="avg">Average</option>
                  <option value="min">Minimum</option>
                  <option value="max">Maximum</option>
                </select>
              </div>
            </div>
          </div>
        </form>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-gray-200 flex justify-end space-x-3">
          <button
            type="button"
            onClick={handleClose}
            disabled={loading}
            className="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50 disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            disabled={loading || !title || !query}
            className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {loading ? 'Creating...' : 'Create Rule'}
          </button>
        </div>
      </div>
    </div>
  );
}
