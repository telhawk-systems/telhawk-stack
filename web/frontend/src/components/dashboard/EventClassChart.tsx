import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts';

interface EventClassChartProps {
  data: Array<{ key: string; doc_count: number }>;
}

export function EventClassChart({ data }: EventClassChartProps) {
  const chartData = data.map(item => ({
    class: item.key.replace(' Activity', '').replace(' Finding', ''),
    count: item.doc_count,
  })).slice(0, 10); // Top 10

  return (
    <div className="bg-white rounded-lg shadow-md p-6">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Top Event Classes</h3>
      <ResponsiveContainer width="100%" height={300}>
        <BarChart data={chartData} layout="vertical">
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis type="number" />
          <YAxis 
            dataKey="class" 
            type="category" 
            width={120}
          />
          <Tooltip />
          <Legend />
          <Bar dataKey="count" fill="#3b82f6" name="Events" />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
