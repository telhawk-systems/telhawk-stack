import React from 'react';
import { EventType } from '../../utils/eventTypes';

interface TypeSpecificDetailsProps {
  event: any;
  type: EventType;
}

interface Field {
  label: string;
  value: any;
  type?: 'text' | 'badge' | 'code' | 'json';
  color?: string;
}

/**
 * Renders type-specific detailed view for an event
 */
export function TypeSpecificDetails({ event, type }: TypeSpecificDetailsProps) {
  // Get the actual event data - it may be at the top level or in raw.data.event
  const eventData = event.raw?.data?.event || event;

  const renderValue = (value: any, valueType: string = 'text'): React.ReactNode => {
    if (value === null || value === undefined) {
      return <span className="text-gray-400">N/A</span>;
    }

    switch (valueType) {
      case 'badge':
        return <span className="px-2 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800">{String(value)}</span>;
      case 'code':
        return <code className="px-2 py-1 bg-gray-100 rounded text-sm font-mono">{String(value)}</code>;
      case 'json':
        return <pre className="text-xs bg-gray-50 p-2 rounded overflow-x-auto">{JSON.stringify(value, null, 2)}</pre>;
      default:
        if (typeof value === 'object') {
          return <pre className="text-xs bg-gray-50 p-2 rounded overflow-x-auto">{JSON.stringify(value, null, 2)}</pre>;
        }
        return <span className="text-gray-900">{String(value)}</span>;
    }
  };

  const renderFields = (fields: Field[]) => (
    <div className="grid grid-cols-2 gap-4">
      {fields.map((field, index) => (
        <div key={index} className="border-b border-gray-200 pb-3">
          <dt className="text-sm font-medium text-gray-500 mb-1">{field.label}</dt>
          <dd className="text-sm">{renderValue(field.value, field.type)}</dd>
        </div>
      ))}
    </div>
  );

  const renderSection = (title: string, fields: Field[]) => (
    <div className="mb-6">
      <h3 className="text-lg font-semibold text-gray-800 mb-3">{title}</h3>
      {renderFields(fields)}
    </div>
  );

  switch (type) {
    case 'authentication':
      return (
        <>
          {renderSection('Authentication Details', [
            { label: 'Activity', value: eventData.activity_name || eventData.activity },
            { label: 'Status', value: eventData.status, type: 'badge' },
            { label: 'Status Detail', value: eventData.status_detail },
            { label: 'Auth Protocol', value: eventData.auth_protocol },
          ])}

          {renderSection('User Information', [
            { label: 'Username', value: eventData.user?.name || eventData.actor?.user?.name },
            { label: 'User ID', value: eventData.user?.uid || eventData.actor?.user?.uid },
            { label: 'Email', value: eventData.user?.email || eventData.actor?.user?.email },
            { label: 'Domain', value: eventData.user?.domain || eventData.actor?.user?.domain },
          ])}

          {renderSection('Source Information', [
            { label: 'Source IP', value: eventData.src_endpoint?.ip },
            { label: 'Source Port', value: eventData.src_endpoint?.port },
            { label: 'Hostname', value: eventData.src_endpoint?.hostname },
            { label: 'Location', value: eventData.src_endpoint?.location },
          ])}

          {eventData.session && renderSection('Session Details', [
            { label: 'Session ID', value: eventData.session?.uid },
            { label: 'Created', value: eventData.session?.created_time },
            { label: 'Expires', value: eventData.session?.expiration_time },
          ])}
        </>
      );

    case 'network':
      return (
        <>
          {renderSection('Network Activity', [
            { label: 'Activity', value: eventData.activity_name },
            { label: 'Protocol', value: eventData.connection_info?.protocol_name || eventData.protocol_name },
            { label: 'Direction', value: eventData.connection_info?.direction },
            { label: 'Boundary', value: eventData.connection_info?.boundary },
          ])}

          {renderSection('Source Endpoint', [
            { label: 'IP Address', value: eventData.src_endpoint?.ip, type: 'code' },
            { label: 'Port', value: eventData.src_endpoint?.port },
            { label: 'Hostname', value: eventData.src_endpoint?.hostname },
            { label: 'MAC Address', value: eventData.src_endpoint?.mac },
          ])}

          {renderSection('Destination Endpoint', [
            { label: 'IP Address', value: eventData.dst_endpoint?.ip, type: 'code' },
            { label: 'Port', value: eventData.dst_endpoint?.port },
            { label: 'Hostname', value: eventData.dst_endpoint?.hostname },
            { label: 'MAC Address', value: eventData.dst_endpoint?.mac },
          ])}

          {eventData.traffic && renderSection('Traffic Information', [
            { label: 'Bytes Sent', value: eventData.traffic?.bytes },
            { label: 'Packets Sent', value: eventData.traffic?.packets },
            { label: 'Bytes Received', value: eventData.traffic?.bytes_in },
            { label: 'Packets Received', value: eventData.traffic?.packets_in },
          ])}
        </>
      );

    case 'process':
      return (
        <>
          {renderSection('Process Details', [
            { label: 'Process Name', value: eventData.process?.name },
            { label: 'Process ID', value: eventData.process?.pid },
            { label: 'Command Line', value: eventData.process?.cmd_line, type: 'code' },
            { label: 'Executable Path', value: eventData.process?.file?.path },
          ])}

          {eventData.parent_process && renderSection('Parent Process', [
            { label: 'Parent Name', value: eventData.parent_process?.name },
            { label: 'Parent PID', value: eventData.parent_process?.pid },
            { label: 'Parent Command', value: eventData.parent_process?.cmd_line, type: 'code' },
          ])}

          {renderSection('User Context', [
            { label: 'Username', value: eventData.actor?.user?.name || eventData.user?.name },
            { label: 'User ID', value: eventData.actor?.user?.uid || eventData.user?.uid },
            { label: 'Effective UID', value: eventData.actor?.user?.uid_alt },
          ])}

          {eventData.process?.file && renderSection('Executable Information', [
            { label: 'File Path', value: eventData.process?.file?.path },
            { label: 'MD5 Hash', value: eventData.process?.file?.hashes?.find((h: any) => h.algorithm === 'MD5')?.value },
            { label: 'SHA256 Hash', value: eventData.process?.file?.hashes?.find((h: any) => h.algorithm === 'SHA-256')?.value },
            { label: 'File Size', value: eventData.process?.file?.size },
          ])}
        </>
      );

    case 'file':
      return (
        <>
          {renderSection('File Activity', [
            { label: 'Activity', value: eventData.activity_name },
            { label: 'File Path', value: eventData.file?.path, type: 'code' },
            { label: 'File Name', value: eventData.file?.name },
            { label: 'File Type', value: eventData.file?.type },
          ])}

          {eventData.file && renderSection('File Attributes', [
            { label: 'Size', value: eventData.file?.size ? `${eventData.file.size} bytes` : 'N/A' },
            { label: 'Modified Time', value: eventData.file?.modified_time },
            { label: 'Created Time', value: eventData.file?.created_time },
            { label: 'Accessed Time', value: eventData.file?.accessed_time },
          ])}

          {eventData.file?.hashes && renderSection('File Hashes', [
            { label: 'MD5', value: eventData.file.hashes?.find((h: any) => h.algorithm === 'MD5')?.value, type: 'code' },
            { label: 'SHA1', value: eventData.file.hashes?.find((h: any) => h.algorithm === 'SHA-1')?.value, type: 'code' },
            { label: 'SHA256', value: eventData.file.hashes?.find((h: any) => h.algorithm === 'SHA-256')?.value, type: 'code' },
          ])}

          {renderSection('Actor Information', [
            { label: 'Username', value: eventData.actor?.user?.name || eventData.user?.name },
            { label: 'Process', value: eventData.actor?.process?.name },
            { label: 'Process PID', value: eventData.actor?.process?.pid },
          ])}
        </>
      );

    case 'dns':
      return (
        <>
          {renderSection('DNS Query', [
            { label: 'Query Name', value: eventData.query?.hostname || eventData.query?.name, type: 'code' },
            { label: 'Query Type', value: eventData.query?.type },
            { label: 'Query Class', value: eventData.query?.class },
            { label: 'Response Code', value: eventData.rcode },
          ])}

          {eventData.answers && eventData.answers.length > 0 && renderSection('DNS Answers', [
            { label: 'Answer Count', value: eventData.answers?.length },
            { label: 'Answers', value: eventData.answers, type: 'json' },
          ])}

          {renderSection('Network Context', [
            { label: 'Source IP', value: eventData.src_endpoint?.ip },
            { label: 'DNS Server', value: eventData.dst_endpoint?.ip },
            { label: 'Protocol', value: eventData.connection_info?.protocol_name || 'UDP' },
          ])}
        </>
      );

    case 'http':
      return (
        <>
          {renderSection('HTTP Request', [
            { label: 'Method', value: eventData.http_request?.method, type: 'badge' },
            { label: 'URL', value: eventData.http_request?.url?.text || eventData.http_request?.url?.path, type: 'code' },
            { label: 'User Agent', value: eventData.http_request?.user_agent },
            { label: 'Referrer', value: eventData.http_request?.referrer },
          ])}

          {eventData.http_response && renderSection('HTTP Response', [
            { label: 'Status Code', value: eventData.http_response?.code, type: 'badge' },
            { label: 'Content Type', value: eventData.http_response?.content_type },
            { label: 'Content Length', value: eventData.http_response?.length ? `${eventData.http_response.length} bytes` : 'N/A' },
          ])}

          {renderSection('Network Information', [
            { label: 'Client IP', value: eventData.src_endpoint?.ip },
            { label: 'Server IP', value: eventData.dst_endpoint?.ip },
            { label: 'Server Port', value: eventData.dst_endpoint?.port },
            { label: 'Hostname', value: eventData.http_request?.url?.hostname },
          ])}
        </>
      );

    case 'detection':
      return (
        <>
          {renderSection('Security Finding', [
            { label: 'Title', value: eventData.finding?.title },
            { label: 'Description', value: eventData.message || eventData.finding?.desc },
            { label: 'Finding Type', value: eventData.finding?.type },
            { label: 'UID', value: eventData.finding?.uid },
          ])}

          {eventData.attacks && eventData.attacks.length > 0 && renderSection('MITRE ATT&CK', [
            { label: 'Tactic', value: eventData.attacks?.[0]?.tactic?.name },
            { label: 'Tactic ID', value: eventData.attacks?.[0]?.tactic?.uid, type: 'code' },
            { label: 'Technique', value: eventData.attacks?.[0]?.technique?.name },
            { label: 'Technique ID', value: eventData.attacks?.[0]?.technique?.uid, type: 'code' },
          ])}

          {renderSection('Detection Details', [
            { label: 'Confidence', value: eventData.confidence },
            { label: 'Risk Score', value: eventData.risk_score },
            { label: 'Impact', value: eventData.impact },
            { label: 'Analytic', value: eventData.analytic?.name },
          ])}

          {eventData.resources && renderSection('Affected Resources', [
            { label: 'Resources', value: eventData.resources, type: 'json' },
          ])}
        </>
      );

    default:
      return (
        <div className="mb-6">
          <h3 className="text-lg font-semibold text-gray-800 mb-3">Event Data</h3>
          <p className="text-sm text-gray-500 mb-3">
            No type-specific view available for this event class.
          </p>
        </div>
      );
  }
}
