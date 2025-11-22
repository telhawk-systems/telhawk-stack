interface SeverityBadgeProps {
  severity: 'critical' | 'high' | 'medium' | 'low' | 'info' | number;
  size?: 'sm' | 'md';
  showDot?: boolean;
}

const severityConfig = {
  critical: {
    label: 'Critical',
    bgClass: 'bg-severity-critical-bg',
    textClass: 'text-severity-critical',
    dotClass: 'bg-severity-critical',
  },
  high: {
    label: 'High',
    bgClass: 'bg-severity-high-bg',
    textClass: 'text-severity-high',
    dotClass: 'bg-severity-high',
  },
  medium: {
    label: 'Medium',
    bgClass: 'bg-severity-medium-bg',
    textClass: 'text-severity-medium',
    dotClass: 'bg-severity-medium',
  },
  low: {
    label: 'Low',
    bgClass: 'bg-severity-low-bg',
    textClass: 'text-severity-low',
    dotClass: 'bg-severity-low',
  },
  info: {
    label: 'Info',
    bgClass: 'bg-severity-info-bg',
    textClass: 'text-severity-info',
    dotClass: 'bg-severity-info',
  },
};

// Map numeric severity to named severity
function mapNumericSeverity(severity: number): keyof typeof severityConfig {
  if (severity >= 5) return 'critical';
  if (severity >= 4) return 'high';
  if (severity >= 3) return 'medium';
  if (severity >= 2) return 'low';
  return 'info';
}

export function SeverityBadge({ severity, size = 'md', showDot = true }: SeverityBadgeProps) {
  const severityKey = typeof severity === 'number' ? mapNumericSeverity(severity) : severity;
  const config = severityConfig[severityKey];

  const sizeClasses = size === 'sm'
    ? 'text-xs px-2 py-0.5'
    : 'text-xs px-2.5 py-1';

  const dotSize = size === 'sm' ? 'w-1.5 h-1.5' : 'w-2 h-2';

  return (
    <span
      className={`inline-flex items-center gap-1.5 font-medium rounded ${sizeClasses} ${config.bgClass} ${config.textClass}`}
    >
      {showDot && <span className={`${dotSize} rounded-full ${config.dotClass}`} />}
      {config.label}
    </span>
  );
}
