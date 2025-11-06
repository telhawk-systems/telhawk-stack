interface EventDetailModalProps {
  event: any;
  onClose: () => void;
}

export function EventDetailModal({ event, onClose }: EventDetailModalProps) {
  if (!event) return null;

  const renderValue = (value: any): string => {
    if (value === null || value === undefined) return 'N/A';
    if (typeof value === 'object') return JSON.stringify(value, null, 2);
    return String(value);
  };

  const mainFields = [
    { label: 'Time', key: 'time', format: (v: string) => new Date(v).toLocaleString() },
    { label: 'Category', key: 'category' },
    { label: 'Class', key: 'class' },
    { label: 'Activity', key: 'activity' },
    { label: 'Severity', key: 'severity' },
    { label: 'Status', key: 'status' },
    { label: 'Type UID', key: 'type_uid' },
    { label: 'Activity ID', key: 'activity_id' },
  ];

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-lg shadow-xl max-w-4xl w-full max-h-[90vh] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="px-6 py-4 border-b border-gray-200 bg-gray-50 flex justify-between items-center">
          <h2 className="text-xl font-bold text-gray-800">Event Details</h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 text-2xl font-bold leading-none"
          >
            ×
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">
          {/* Main Fields */}
          <div className="mb-6">
            <h3 className="text-lg font-semibold text-gray-800 mb-3">OCSF Fields</h3>
            <div className="grid grid-cols-2 gap-4">
              {mainFields.map(({ label, key, format }) => (
                <div key={key} className="border-b border-gray-200 pb-2">
                  <dt className="text-sm font-medium text-gray-500">{label}</dt>
                  <dd className="mt-1 text-sm text-gray-900">
                    {format ? format(event[key]) : renderValue(event[key])}
                  </dd>
                </div>
              ))}
            </div>
          </div>

          {/* Metadata */}
          {event.metadata && (
            <div className="mb-6">
              <h3 className="text-lg font-semibold text-gray-800 mb-3">Metadata</h3>
              <div className="bg-gray-50 rounded-md p-4">
                <pre className="text-xs text-gray-700 whitespace-pre-wrap">
                  {JSON.stringify(event.metadata, null, 2)}
                </pre>
              </div>
            </div>
          )}

          {/* Raw Data */}
          {event.raw && (
            <div className="mb-6">
              <h3 className="text-lg font-semibold text-gray-800 mb-3">Raw Event Data</h3>
              <div className="bg-gray-50 rounded-md p-4">
                <pre className="text-xs text-gray-700 whitespace-pre-wrap overflow-x-auto">
                  {JSON.stringify(event.raw.data, null, 2)}
                </pre>
              </div>
            </div>
          )}

          {/* Properties */}
          {event.properties && (
            <div className="mb-6">
              <h3 className="text-lg font-semibold text-gray-800 mb-3">Properties</h3>
              <div className="bg-gray-50 rounded-md p-4">
                <pre className="text-xs text-gray-700 whitespace-pre-wrap">
                  {JSON.stringify(event.properties, null, 2)}
                </pre>
              </div>
            </div>
          )}

          {/* Full Event JSON */}
          <div>
            <details className="group">
              <summary className="text-lg font-semibold text-gray-800 mb-3 cursor-pointer hover:text-blue-600">
                Full Event JSON (click to expand)
              </summary>
              <div className="bg-gray-50 rounded-md p-4 mt-2">
                <pre className="text-xs text-gray-700 whitespace-pre-wrap overflow-x-auto">
                  {JSON.stringify(event, null, 2)}
                </pre>
              </div>
            </details>
          </div>
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-gray-200 bg-gray-50 flex justify-end">
          <button
            onClick={onClose}
            className="px-6 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 font-medium transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
