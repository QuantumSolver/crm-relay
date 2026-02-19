import { useEffect, useState } from 'react';
import { metricsApi, Metrics } from '../lib/api';

export default function Dashboard() {
  const [metrics, setMetrics] = useState<Metrics | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchMetrics = async () => {
    try {
      const data = await metricsApi.get();
      setMetrics(data);
    } catch (error) {
      console.error('Failed to fetch metrics:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchMetrics();
    const interval = setInterval(fetchMetrics, 5000); // Refresh every 5 seconds
    return () => clearInterval(interval);
  }, []);

  if (loading) {
    return <div className="p-6">Loading metrics...</div>;
  }

  if (!metrics) {
    return <div className="p-6">Failed to load metrics</div>;
  }

  const cards = [
    { title: 'Webhooks Processed', value: metrics.WebhooksProcessed, color: 'bg-green-500' },
    { title: 'Webhooks Failed', value: metrics.WebhooksFailed, color: 'bg-red-500' },
    { title: 'Webhooks Retried', value: metrics.WebhooksRetried, color: 'bg-yellow-500' },
    { title: 'Average Latency', value: `${metrics.AverageLatency}ms`, color: 'bg-blue-500' },
  ];

  return (
    <div className="p-6">
      <h1 className="text-3xl font-bold mb-6">Dashboard</h1>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        {cards.map((card) => (
          <div key={card.title} className="bg-white rounded-lg shadow-md p-6">
            <h3 className="text-gray-600 text-sm font-medium mb-2">{card.title}</h3>
            <p className={`text-3xl font-bold text-white ${card.color} rounded-lg p-4`}>
              {card.value}
            </p>
          </div>
        ))}
      </div>

      <div className="bg-white rounded-lg shadow-md p-6">
        <h2 className="text-xl font-bold mb-4">System Information</h2>
        <div className="space-y-2">
          <div className="flex justify-between">
            <span className="text-gray-600">Webhooks Received:</span>
            <span className="font-medium">{metrics.WebhooksReceived.toLocaleString()}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-gray-600">Last Webhook:</span>
            <span className="font-medium">
              {metrics.LastWebhookTime
                ? new Date(metrics.LastWebhookTime).toLocaleString()
                : 'Never'}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}
