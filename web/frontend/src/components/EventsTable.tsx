import { useNavigate } from 'react-router-dom';
import { detectEventType, getEventTypeName, getEventTypeIcon } from '../utils/eventTypes';
import { EntityType, getEntityIcon } from '../utils/entityUtils';

interface Event {
  time: string;
  category: string;
  class: string;
  severity: string;
  activity: string;
  [key: string]: any;
}

interface EventsTableProps {
  events: Event[];
  totalMatches?: number;
  onEventClick: (event: Event) => void;
  onLoadMore?: () => void;
  loadingMore?: boolean;
}

export function EventsTable({ events, totalMatches, onEventClick, onLoadMore, loadingMore }: EventsTableProps) {
  const navigate = useNavigate();

  if (!events || events.length === 0) {
    return (
      <div className="bg-white rounded-lg shadow-md p-8 text-center">
        <p className="text-gray-500 text-lg">No events found</p>
        <p className="text-gray-400 text-sm mt-2">Try adjusting your search query or time range</p>
      </div>
    );
  }

  const getSeverityColor = (severity: string) => {
    switch (severity?.toLowerCase()) {
      case 'critical':
        return 'bg-red-100 text-red-800 border-red-300';
      case 'high':
        return 'bg-orange-100 text-orange-800 border-orange-300';
      case 'medium':
        return 'bg-yellow-100 text-yellow-800 border-yellow-300';
      case 'low':
        return 'bg-blue-100 text-blue-800 border-blue-300';
      case 'informational':
        return 'bg-gray-100 text-gray-800 border-gray-300';
      default:
        return 'bg-gray-100 text-gray-600 border-gray-200';
    }
  };

  const renderValue = (value: any): string => {
    if (value === null || value === undefined) return 'N/A';
    if (typeof value === 'object') return JSON.stringify(value);
    return String(value);
  };

  /**
   * Render an entity value as a clickable link
   */
  const renderEntityLink = (value: string, type: EntityType) => {
    if (!value || value === 'N/A') return value;

    const handleClick = (event: React.MouseEvent) => {
      event.stopPropagation(); // Prevent event card click
      navigate(`/entity/${type}/${encodeURIComponent(value)}`);
    };

    return (
      <button
        onClick={handleClick}
        className="text-blue-600 hover:text-blue-800 hover:underline font-medium text-left truncate"
        title={`View all events for ${type}: ${value}`}
      >
        {getEntityIcon(type)} {value}
      </button>
    );
  };

  const renderEventCard = (event: Event, index: number) => {
    const eventType = detectEventType(event);
    const typeName = getEventTypeName(eventType);
    const typeIcon = getEventTypeIcon(eventType);
    const eventData = event.raw?.data?.event || event;

    // Type-specific fields to display
    let typeSpecificFields: Array<{ label: string; value: any; entityType?: EntityType }> = [];

    switch (eventType) {
      case 'authentication':
        typeSpecificFields = [
          { label: 'Username', value: eventData.user?.name || eventData.actor?.user?.name, entityType: 'user' },
          { label: 'Source IP', value: eventData.src_endpoint?.ip, entityType: 'ip' },
          { label: 'Status', value: eventData.status },
          { label: 'Auth Protocol', value: eventData.auth_protocol },
        ];
        break;
      case 'network':
        typeSpecificFields = [
          { label: 'Source IP', value: eventData.src_endpoint?.ip, entityType: 'ip' },
          { label: 'Destination IP', value: eventData.dst_endpoint?.ip, entityType: 'ip' },
          { label: 'Protocol', value: eventData.connection_info?.protocol_name },
          { label: 'Direction', value: eventData.connection_info?.direction },
        ];
        break;
      case 'process':
        typeSpecificFields = [
          { label: 'Process', value: eventData.process?.name, entityType: 'process' },
          { label: 'PID', value: eventData.process?.pid },
          { label: 'Command', value: eventData.process?.cmd_line },
          { label: 'User', value: eventData.actor?.user?.name || eventData.user?.name, entityType: 'user' },
        ];
        break;
      case 'file':
        typeSpecificFields = [
          { label: 'File Path', value: eventData.file?.path, entityType: 'file' },
          { label: 'Operation', value: eventData.activity_name },
          { label: 'User', value: eventData.actor?.user?.name || eventData.user?.name, entityType: 'user' },
          { label: 'Size', value: eventData.file?.size ? `${eventData.file.size} bytes` : 'N/A' },
        ];
        break;
      case 'dns':
        typeSpecificFields = [
          { label: 'Query', value: eventData.query?.hostname, entityType: 'hostname' },
          { label: 'Type', value: eventData.query?.type },
          { label: 'Answer', value: eventData.answers?.[0]?.rdata },
          { label: 'Source IP', value: eventData.src_endpoint?.ip, entityType: 'ip' },
        ];
        break;
      case 'http':
        typeSpecificFields = [
          { label: 'Method', value: eventData.http_request?.method },
          { label: 'URL', value: eventData.http_request?.url?.path || eventData.http_request?.url?.text },
          { label: 'Status Code', value: eventData.http_response?.code },
          { label: 'Client IP', value: eventData.src_endpoint?.ip, entityType: 'ip' },
        ];
        break;
      case 'detection':
        typeSpecificFields = [
          { label: 'Finding', value: eventData.finding?.title || eventData.message },
          { label: 'Tactic', value: eventData.attacks?.[0]?.tactic?.name },
          { label: 'Technique', value: eventData.attacks?.[0]?.technique?.name },
          { label: 'Risk Score', value: eventData.risk_score },
        ];
        break;
      default:
        // Splunk-like fallback view for unknown event types or raw events
        typeSpecificFields = [
          { label: 'Source Type', value: event.properties?.source_type || eventData.sourcetype },
          { label: 'Source', value: event.properties?.source || eventData.source },
          { label: 'Host', value: eventData.device?.hostname || eventData.device?.ip || eventData.observables?.hostname, entityType: eventData.device?.hostname ? 'hostname' : eventData.device?.ip ? 'ip' : undefined },
          { label: 'Message', value: eventData.message || eventData.description || 'No message' },
        ];
    }

    return (
      <div
        key={index}
        className="bg-white border border-gray-200 rounded-lg p-4 hover:shadow-md transition-shadow cursor-pointer mb-3"
        onClick={() => onEventClick(event)}
      >
        {/* Header Row */}
        <div className="flex items-center justify-between mb-3 pb-3 border-b border-gray-200">
          <div className="flex items-center space-x-3">
            <span className="text-2xl">{typeIcon}</span>
            <div>
              <div className="flex items-center space-x-2">
                <span className="font-semibold text-gray-900">{typeName}</span>
                <span className={`px-2 py-1 rounded-full text-xs font-medium border ${getSeverityColor(event.severity)}`}>
                  {event.severity || 'Unknown'}
                </span>
              </div>
              <div className="text-sm text-gray-500 mt-1">
                {new Date(event.time).toLocaleString()}
              </div>
            </div>
          </div>
          <button
            onClick={(e) => {
              e.stopPropagation();
              onEventClick(event);
            }}
            className="text-blue-600 hover:text-blue-800 font-medium text-sm"
          >
            View Details →
          </button>
        </div>

        {/* Type-Specific Fields Grid */}
        <div className="grid grid-cols-2 gap-3">
          {typeSpecificFields.map((field, i) => (
            <div key={i} className="flex flex-col">
              <span className="text-xs font-medium text-gray-500 uppercase tracking-wider">
                {field.label}
              </span>
              <span className="text-sm text-gray-900 mt-1 truncate" title={renderValue(field.value)}>
                {field.entityType && field.value && field.value !== 'N/A'
                  ? renderEntityLink(renderValue(field.value), field.entityType)
                  : renderValue(field.value)}
              </span>
            </div>
          ))}
        </div>
      </div>
    );
  };

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="bg-white rounded-lg shadow-md px-6 py-4">
        <h3 className="text-lg font-semibold text-gray-800">
          Events {totalMatches && `(${events.length} of ${totalMatches})`}
        </h3>
      </div>

      {/* Event Cards List */}
      <div className="space-y-3">
        {events.map((event, index) => renderEventCard(event, index))}
      </div>

      {/* Pagination Controls */}
      {onLoadMore && (
        <div className="bg-white rounded-lg shadow-md px-6 py-4">
          <div className="flex items-center justify-between">
            <p className="text-sm text-gray-600">
              Showing {events.length} {totalMatches && `of ${totalMatches} total events`}
            </p>
            <button
              onClick={onLoadMore}
              disabled={loadingMore}
              className="px-4 py-2 bg-blue-600 text-white rounded-md font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {loadingMore ? 'Loading...' : 'Load More'}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
