import { PieChart, Pie, Cell, ResponsiveContainer, Legend, Tooltip } from 'recharts';

interface SeverityChartProps {
  data: Array<{ key: string; doc_count: number }>;
}

const SEVERITY_COLORS: Record<string, string> = {
  '1': '#ef4444', // Critical - red
  '2': '#f97316', // High - orange
  '3': '#eab308', // Medium - yellow
  '4': '#3b82f6', // Low - blue
  '5': '#6b7280', // Informational - gray
};

const SEVERITY_LABELS: Record<string, string> = {
  '1': 'Critical',
  '2': 'High',
  '3': 'Medium',
  '4': 'Low',
  '5': 'Info',
};

export function SeverityChart({ data }: SeverityChartProps) {
  const chartData = data.map(item => ({
    name: SEVERITY_LABELS[item.key] || `Severity ${item.key}`,
    value: item.doc_count,
    severity: item.key,
  }));

  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Events by Severity</h3>
      <ResponsiveContainer width="100%" height={300}>
        <PieChart>
          <Pie
            data={chartData}
            cx="50%"
            cy="50%"
            labelLine={false}
            label={({ name, percent }) => `${name}: ${((percent as number) * 100).toFixed(0)}%`}
            outerRadius={80}
            fill="#8884d8"
            dataKey="value"
          >
            {chartData.map((entry, index) => (
              <Cell key={`cell-${index}`} fill={SEVERITY_COLORS[entry.severity] || '#6b7280'} />
            ))}
          </Pie>
          <Tooltip />
          <Legend />
        </PieChart>
      </ResponsiveContainer>
    </div>
  );
}
