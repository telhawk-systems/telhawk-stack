import { useState, useEffect, FormEvent } from 'react';
import { DetectionSchema, DetectionSchemaCreateRequest } from '../../types/rules';

interface EditRuleModalProps {
  isOpen: boolean;
  schema: DetectionSchema | null;
  onClose: () => void;
  onSuccess: () => void;
}

export function EditRuleModal({ isOpen, schema, onClose, onSuccess }: EditRuleModalProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // View fields
  const [title, setTitle] = useState('');
  const [severity, setSeverity] = useState<'critical' | 'high' | 'medium' | 'low' | 'informational'>('medium');
  const [priority, setPriority] = useState('P3');
  const [descriptionTemplate, setDescriptionTemplate] = useState('');
  const [mitreTactics, setMitreTactics] = useState('');
  const [mitreTechniques, setMitreTechniques] = useState('');

  // Controller fields
  const [query, setQuery] = useState('');
  const [condition, setCondition] = useState('');
  const [lookback, setLookback] = useState('5m');
  const [evaluationInterval, setEvaluationInterval] = useState('1m');
  const [aggregationField, setAggregationField] = useState('');

  // Model fields
  const [fields, setFields] = useState('');
  const [groupBy, setGroupBy] = useState('');
  const [timeWindow, setTimeWindow] = useState('5m');
  const [threshold, setThreshold] = useState('10');
  const [aggregation, setAggregation] = useState('count');

  // Load schema data when modal opens
  useEffect(() => {
    if (schema && isOpen) {
      // View fields
      setTitle(schema.view.title);
      setSeverity(schema.view.severity);
      setPriority(schema.view.priority || 'P3');
      setDescriptionTemplate(schema.view.description_template || '');
      setMitreTactics(schema.view.mitre_attack?.tactics?.join(', ') || '');
      setMitreTechniques(schema.view.mitre_attack?.techniques?.join(', ') || '');

      // Controller fields
      setQuery(schema.controller.query);
      setCondition(schema.controller.condition || '');
      setLookback(schema.controller.lookback || '5m');
      setEvaluationInterval(schema.controller.evaluation_interval || '1m');
      setAggregationField(schema.controller.aggregation_field || '');

      // Model fields
      setFields(schema.model.fields?.join(', ') || '');
      setGroupBy(schema.model.group_by?.join(', ') || '');
      setTimeWindow(schema.model.time_window || '5m');
      setThreshold(schema.model.threshold?.toString() || '10');
      setAggregation(schema.model.aggregation || 'count');
    }
  }, [schema, isOpen]);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!schema) return;

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

      // PUT to create a new version of the existing rule
      const response = await fetch(`/api/rules/api/v1/schemas/${schema.id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(request),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ message: 'Failed to update rule' }));
        throw new Error(errorData.message || `HTTP error! status: ${response.status}`);
      }

      // Success
      onSuccess();
      handleClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update detection rule');
      console.error('Error updating rule:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    setError(null);
    onClose();
  };

  if (!isOpen || !schema) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50 overflow-y-auto">
      <div className="bg-white rounded-lg shadow-xl max-w-4xl w-full my-8">
        {/* Header */}
        <div className="px-6 py-4 border-b border-gray-200 flex justify-between items-center">
          <div>
            <h2 className="text-xl font-bold text-gray-900">Edit Detection Rule</h2>
            <p className="text-sm text-gray-500 mt-1">
              Creating new version (current: v{schema.version})
            </p>
          </div>
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

        {/* Form - same as CreateRuleModal */}
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
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono text-sm"
                />
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
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Lookback Window
                </label>
                <input
                  type="text"
                  value={lookback}
                  onChange={(e) => setLookback(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Evaluation Interval
                </label>
                <input
                  type="text"
                  value={evaluationInterval}
                  onChange={(e) => setEvaluationInterval(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div className="md:col-span-2">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Aggregation Field
                </label>
                <input
                  type="text"
                  value={aggregationField}
                  onChange={(e) => setAggregationField(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
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
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div className="md:col-span-2">
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Group By
                </label>
                <input
                  type="text"
                  value={groupBy}
                  onChange={(e) => setGroupBy(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Time Window
                </label>
                <input
                  type="text"
                  value={timeWindow}
                  onChange={(e) => setTimeWindow(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Threshold
                </label>
                <input
                  type="number"
                  value={threshold}
                  onChange={(e) => setThreshold(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
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
            {loading ? 'Updating...' : 'Update Rule (New Version)'}
          </button>
        </div>
      </div>
    </div>
  );
}
