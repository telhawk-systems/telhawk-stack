import { EventType } from '../../utils/eventTypes';

interface TypeSpecificColumnsProps {
  event: any;
  type: EventType;
}

/**
 * Renders type-specific columns for an event in the table
 */
export function TypeSpecificColumns({ event, type }: TypeSpecificColumnsProps) {
  // Get the actual event data - it may be at the top level or in raw.data.event
  const eventData = event.raw?.data?.event || event;

  const renderValue = (value: any): string => {
    if (value === null || value === undefined) return 'N/A';
    if (typeof value === 'object') return JSON.stringify(value);
    return String(value);
  };

  switch (type) {
    case 'authentication':
      return (
        <>
          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
            {renderValue(eventData.user?.name || eventData.actor?.user?.name)}
          </td>
          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
            {renderValue(eventData.src_endpoint?.ip)}
          </td>
          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
            <span className={`px-2 py-1 rounded-full text-xs font-medium ${
              eventData.status?.toLowerCase() === 'success' || eventData.status_id === 1
                ? 'bg-green-100 text-green-800'
                : 'bg-red-100 text-red-800'
            }`}>
              {eventData.status || (eventData.status_id === 1 ? 'Success' : 'Failure')}
            </span>
          </td>
        </>
      );

    case 'network':
      return (
        <>
          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
            {renderValue(eventData.src_endpoint?.ip)}:{eventData.src_endpoint?.port || '?'}
          </td>
          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
            {renderValue(eventData.dst_endpoint?.ip)}:{eventData.dst_endpoint?.port || '?'}
          </td>
          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
            {renderValue(eventData.connection_info?.protocol_name || eventData.protocol_name)}
          </td>
        </>
      );

    case 'process':
      return (
        <>
          <td className="px-6 py-4 text-sm text-gray-900 max-w-xs truncate" title={eventData.process?.cmd_line}>
            {renderValue(eventData.process?.name || eventData.process?.cmd_line)}
          </td>
          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
            {renderValue(eventData.process?.pid)}
          </td>
          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
            {renderValue(eventData.actor?.user?.name || eventData.user?.name)}
          </td>
        </>
      );

    case 'file':
      return (
        <>
          <td className="px-6 py-4 text-sm text-gray-900 max-w-xs truncate" title={eventData.file?.path}>
            {renderValue(eventData.file?.path || eventData.file?.name)}
          </td>
          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
            {renderValue(eventData.activity_name || eventData.type_name)}
          </td>
          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
            {renderValue(eventData.actor?.user?.name || eventData.user?.name)}
          </td>
        </>
      );

    case 'dns':
      return (
        <>
          <td className="px-6 py-4 text-sm text-gray-900 max-w-xs truncate">
            {renderValue(eventData.query?.hostname || eventData.query?.name)}
          </td>
          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
            {renderValue(eventData.query?.type || eventData.query?.class)}
          </td>
          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
            {renderValue(eventData.answers?.[0]?.rdata || eventData.rcode)}
          </td>
        </>
      );

    case 'http':
      return (
        <>
          <td className="px-6 py-4 text-sm text-gray-900 max-w-xs truncate" title={eventData.http_request?.url?.text}>
            {renderValue(eventData.http_request?.method || 'GET')} {renderValue(eventData.http_request?.url?.path || eventData.http_request?.url?.text)}
          </td>
          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
            <span className={`px-2 py-1 rounded-full text-xs font-medium ${
              (eventData.http_response?.code >= 200 && eventData.http_response?.code < 300)
                ? 'bg-green-100 text-green-800'
                : (eventData.http_response?.code >= 400)
                ? 'bg-red-100 text-red-800'
                : 'bg-yellow-100 text-yellow-800'
            }`}>
              {eventData.http_response?.code || 'N/A'}
            </span>
          </td>
          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
            {renderValue(eventData.dst_endpoint?.ip)}
          </td>
        </>
      );

    case 'detection':
      return (
        <>
          <td className="px-6 py-4 text-sm text-gray-900 max-w-xs truncate" title={eventData.finding?.title}>
            {renderValue(eventData.finding?.title || eventData.message)}
          </td>
          <td className="px-6 py-4 text-sm text-gray-900 max-w-xs truncate">
            {renderValue(eventData.attacks?.[0]?.tactic?.name || eventData.tactic)}
          </td>
          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
            {renderValue(eventData.attacks?.[0]?.technique?.name || eventData.technique)}
          </td>
        </>
      );

    default:
      // Generic columns for unknown event types
      return (
        <>
          <td className="px-6 py-4 text-sm text-gray-900">
            {renderValue(eventData.message || eventData.description)}
          </td>
          <td className="px-6 py-4 text-sm text-gray-900">
            {renderValue(eventData.source || eventData.product?.name)}
          </td>
          <td className="px-6 py-4 text-sm text-gray-900">
            {renderValue(eventData.type_name)}
          </td>
        </>
      );
  }
}

/**
 * Get column headers for a specific event type
 */
export function getTypeSpecificHeaders(type: EventType): string[] {
  switch (type) {
    case 'authentication':
      return ['Username', 'Source IP', 'Status'];
    case 'network':
      return ['Source', 'Destination', 'Protocol'];
    case 'process':
      return ['Process', 'PID', 'User'];
    case 'file':
      return ['File Path', 'Operation', 'User'];
    case 'dns':
      return ['Query', 'Type', 'Answer'];
    case 'http':
      return ['Request', 'Status', 'Destination'];
    case 'detection':
      return ['Finding', 'Tactic', 'Technique'];
    default:
      return ['Message', 'Source', 'Type'];
  }
}
