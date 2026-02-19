import { useState, useEffect } from 'react';
import { configApi } from '../lib/api';

export default function RetryConfig() {
  const [maxRetries, setMaxRetries] = useState(3);
  const [retryDelay, setRetryDelay] = useState(1000);
  const [backoffMultiplier, setBackoffMultiplier] = useState(2);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const fetchConfig = async () => {
    try {
      const data = await configApi.get();
      setMaxRetries(data.retry_config.max_retries);
      setRetryDelay(data.retry_config.retry_delay);
      setBackoffMultiplier(data.retry_config.backoff_multiplier);
    } catch (error) {
      console.error('Failed to fetch config:', error);
      showMessage('error', 'Failed to load configuration');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchConfig();
  }, []);

  const showMessage = (type: 'success' | 'error', text: string) => {
    setMessage({ type, text });
    setTimeout(() => setMessage(null), 5000);
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      await configApi.updateRetryConfig({
        max_retries: maxRetries,
        retry_delay: retryDelay,
        backoff_multiplier: backoffMultiplier,
      });
      await fetchConfig();
      showMessage('success', 'Retry configuration updated successfully');
    } catch (error) {
      console.error('Failed to update retry config:', error);
      showMessage('error', 'Failed to update retry configuration');
    } finally {
      setSaving(false);
    }
  };

  const handleReset = () => {
    setMaxRetries(3);
    setRetryDelay(1000);
    setBackoffMultiplier(2);
  };

  const calculateRetrySchedule = () => {
    const schedule = [];
    let delay = retryDelay;
    for (let i = 1; i <= maxRetries; i++) {
      schedule.push({ attempt: i, delay: `${delay}ms` });
      delay = Math.floor(delay * backoffMultiplier);
    }
    return schedule;
  };

  if (loading) {
    return <div className="p-6">Loading configuration...</div>;
  }

  const retrySchedule = calculateRetrySchedule();

  return (
    <div className="p-6">
      <h1 className="text-3xl font-bold mb-6">Retry Configuration</h1>

      {message && (
        <div
          className={`mb-4 px-4 py-3 rounded ${
            message.type === 'success'
              ? 'bg-green-100 border border-green-400 text-green-700'
              : 'bg-red-100 border border-red-400 text-red-700'
          }`}
        >
          {message.text}
        </div>
      )}

      <div className="bg-white rounded-lg shadow-md p-6">
        <h2 className="text-xl font-bold mb-4">Retry Settings</h2>

        <div className="mb-4">
          <label htmlFor="maxRetries" className="block text-gray-700 text-sm font-bold mb-2">
            Max Retries
          </label>
          <input
            type="number"
            id="maxRetries"
            value={maxRetries}
            onChange={(e) => setMaxRetries(parseInt(e.target.value) || 0)}
            min="0"
            max="10"
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <p className="mt-2 text-sm text-gray-600">
            Maximum number of retry attempts for failed webhooks.
          </p>
        </div>

        <div className="mb-4">
          <label htmlFor="retryDelay" className="block text-gray-700 text-sm font-bold mb-2">
            Initial Retry Delay (ms)
          </label>
          <input
            type="number"
            id="retryDelay"
            value={retryDelay}
            onChange={(e) => setRetryDelay(parseInt(e.target.value) || 0)}
            min="100"
            max="60000"
            step="100"
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <p className="mt-2 text-sm text-gray-600">
            Initial delay before the first retry attempt.
          </p>
        </div>

        <div className="mb-6">
          <label htmlFor="backoffMultiplier" className="block text-gray-700 text-sm font-bold mb-2">
            Backoff Multiplier
          </label>
          <input
            type="number"
            id="backoffMultiplier"
            value={backoffMultiplier}
            onChange={(e) => setBackoffMultiplier(parseFloat(e.target.value) || 1)}
            min="1"
            max="10"
            step="0.1"
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <p className="mt-2 text-sm text-gray-600">
            Multiplier for exponential backoff between retries.
          </p>
        </div>

        <div className="flex space-x-4">
          <button
            onClick={handleSave}
            disabled={saving}
            className="bg-blue-500 text-white py-2 px-4 rounded-md hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
          >
            {saving ? 'Saving...' : 'Save Configuration'}
          </button>
          <button
            onClick={handleReset}
            className="bg-gray-500 text-white py-2 px-4 rounded-md hover:bg-gray-600 focus:outline-none focus:ring-2 focus:ring-gray-500"
          >
            Reset to Defaults
          </button>
        </div>
      </div>

      <div className="bg-white rounded-lg shadow-md p-6 mt-6">
        <h2 className="text-xl font-bold mb-4">Retry Schedule Preview</h2>
        {retrySchedule.length > 0 ? (
          <table className="min-w-full">
            <thead>
              <tr className="border-b">
                <th className="text-left py-2 px-4">Attempt</th>
                <th className="text-left py-2 px-4">Delay</th>
              </tr>
            </thead>
            <tbody>
              {retrySchedule.map((item) => (
                <tr key={item.attempt} className="border-b">
                  <td className="py-2 px-4">Retry #{item.attempt}</td>
                  <td className="py-2 px-4">{item.delay}</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <p className="text-gray-600">No retries will be attempted.</p>
        )}
      </div>
    </div>
  );
}
