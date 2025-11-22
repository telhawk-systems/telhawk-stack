import { useState, useEffect } from 'react';
import { apiClient } from '../../services/api';
import { useAuth } from '../AuthProvider';
import { MetricCard } from './MetricCard';
import { SeverityChart } from './SeverityChart';
import { TimelineChart } from './TimelineChart';
import { EventClassChart } from './EventClassChart';
import { TimeRangeSelector } from '../common/TimeRangeSelector';

interface TimeRangeConfig {
  label: string;
  value: string;
  start: string;
  end: string;
}

const TIME_RANGES: TimeRangeConfig[] = [
  {
    label: 'Last 15 minutes',
    value: '15m',
    start: new Date(Date.now() - 15 * 60 * 1000).toISOString(),
    end: new Date().toISOString()
  },
  {
    label: 'Last hour',
    value: '1h',
    start: new Date(Date.now() - 60 * 60 * 1000).toISOString(),
    end: new Date().toISOString()
  },
  {
    label: 'Last 24 hours',
    value: '24h',
    start: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
    end: new Date().toISOString()
  },
  {
    label: 'Last 7 days',
    value: '7d',
    start: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
    end: new Date().toISOString()
  },
];

export function DashboardOverview() {
  const { user } = useAuth();
  const [timeRange, setTimeRange] = useState<TimeRangeConfig>(TIME_RANGES[2]); // 24h default
  const [metrics, setMetrics] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [autoRefresh, setAutoRefresh] = useState(false);

  const loadMetrics = async () => {
    if (!user) return;

    try {
      setError('');
      const data = await apiClient.getDashboardMetrics();
      setMetrics(data);
    } catch (err) {
      setError('Failed to load dashboard metrics');
      console.error('Dashboard error:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (user) {
      loadMetrics();
    }
  }, [timeRange, user]);

  useEffect(() => {
    if (autoRefresh && user) {
      const interval = setInterval(loadMetrics, 30000); // Refresh every 30s
      return () => clearInterval(interval);
    }
  }, [autoRefresh, timeRange, user]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-50 border-l-4 border-red-500 p-4 rounded-md">
        <p className="text-sm text-red-700">{error}</p>
      </div>
    );
  }

  const severityData = metrics?.aggregations?.severity_count?.buckets || [];
  const classData = metrics?.aggregations?.events_by_class?.buckets || [];
  const timelineData = metrics?.aggregations?.timeline?.buckets || [];
  const uniqueUsers = metrics?.aggregations?.unique_users?.value || 0;
  const uniqueIPs = metrics?.aggregations?.unique_ips?.value || 0;
  const totalEvents = metrics?.total_matches || 0;

  const handleTimeRangeChange = (value: string) => {
    const selected = TIME_RANGES.find(r => r.value === value);
    if (selected) setTimeRange(selected);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">Security Overview</h1>
          <p className="text-sm text-gray-500 mt-1">
            Last updated: {new Date().toLocaleTimeString()}
          </p>
        </div>
        <div className="flex items-center gap-3">
          {/* Time Range Selector */}
          <TimeRangeSelector
            value={timeRange.value}
            onChange={handleTimeRangeChange}
          />

          {/* Auto Refresh Toggle */}
          <label className="flex items-center gap-2 px-3 py-2 bg-white border border-gray-200 rounded-md cursor-pointer hover:bg-gray-50 transition-colors">
            <input
              type="checkbox"
              checked={autoRefresh}
              onChange={(e) => setAutoRefresh(e.target.checked)}
              className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="text-sm text-gray-700">Auto-refresh</span>
          </label>

          {/* Manual Refresh Button */}
          <button
            onClick={loadMetrics}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700 transition-colors"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            Refresh
          </button>
        </div>
      </div>

      {/* Metric Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <MetricCard
          title="Total Events"
          value={totalEvents.toLocaleString()}
          subtitle={timeRange.label.toLowerCase()}
          icon={
            <svg className="w-6 h-6 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
            </svg>
          }
        />
        
        <MetricCard
          title="Critical Events"
          value={severityData.find((s: any) => s.key === '1')?.doc_count || 0}
          subtitle="severity level 1"
          icon={
            <svg className="w-6 h-6 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          }
        />

        <MetricCard
          title="Unique Users"
          value={uniqueUsers.toLocaleString()}
          subtitle="distinct actors"
          icon={
            <svg className="w-6 h-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
            </svg>
          }
        />

        <MetricCard
          title="Unique IPs"
          value={uniqueIPs.toLocaleString()}
          subtitle="source addresses"
          icon={
            <svg className="w-6 h-6 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
            </svg>
          }
        />
      </div>

      {/* Charts Row 1 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {severityData.length > 0 && <SeverityChart data={severityData} />}
        {classData.length > 0 && <EventClassChart data={classData} />}
      </div>

      {/* Timeline Chart */}
      {timelineData.length > 0 && (
        <TimelineChart data={timelineData} />
      )}
    </div>
  );
}
