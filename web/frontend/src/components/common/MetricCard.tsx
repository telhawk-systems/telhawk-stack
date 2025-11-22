interface MetricCardProps {
  label: string;
  value: string | number;
  trend?: {
    value: number;
    label?: string;
  };
  icon?: React.ReactNode;
  onClick?: () => void;
}

const TrendUpIcon = () => (
  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M5 10l7-7m0 0l7 7m-7-7v18" />
  </svg>
);

const TrendDownIcon = () => (
  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M19 14l-7 7m0 0l-7-7m7 7V3" />
  </svg>
);

export function MetricCard({ label, value, trend, icon, onClick }: MetricCardProps) {
  const formattedValue = typeof value === 'number'
    ? value.toLocaleString()
    : value;

  const isPositiveTrend = trend && trend.value > 0;
  const isNegativeTrend = trend && trend.value < 0;
  const trendValue = trend ? Math.abs(trend.value) : 0;

  return (
    <div
      className={`bg-white rounded-lg p-5 shadow-sm border border-surface-border transition-shadow ${
        onClick ? 'cursor-pointer hover:shadow-md' : ''
      }`}
      onClick={onClick}
    >
      <div className="flex items-start justify-between">
        <span className="text-xs font-medium text-gray-500 uppercase tracking-wide">
          {label}
        </span>
        {icon && (
          <span className="text-gray-400">{icon}</span>
        )}
      </div>

      <div className="mt-2">
        <span className="text-3xl font-bold text-gray-900">
          {formattedValue}
        </span>
      </div>

      {trend && (
        <div className="mt-2 flex items-center gap-1">
          <span
            className={`inline-flex items-center gap-0.5 text-xs font-medium ${
              isPositiveTrend
                ? 'text-green-600'
                : isNegativeTrend
                ? 'text-red-600'
                : 'text-gray-500'
            }`}
          >
            {isPositiveTrend && <TrendUpIcon />}
            {isNegativeTrend && <TrendDownIcon />}
            {trendValue}%
          </span>
          {trend.label && (
            <span className="text-xs text-gray-400">{trend.label}</span>
          )}
        </div>
      )}
    </div>
  );
}
