import { useState, useEffect } from 'react';
import { configApi, ConfigResponse } from '../lib/api';

export default function Config() {
  const [config, setConfig] = useState<ConfigResponse | null>(null);
  const [localEndpoint, setLocalEndpoint] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const fetchConfig = async () => {
    try {
      const data = await configApi.get();
      setConfig(data);
      setLocalEndpoint(data.local_endpoint);
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
      await configApi.updateLocalEndpoint(localEndpoint);
      await fetchConfig();
      showMessage('success', 'Configuration updated successfully');
    } catch (error) {
      console.error('Failed to update config:', error);
      showMessage('error', 'Failed to update configuration');
    } finally {
      setSaving(false);
    }
  };

  const handleTest = async () => {
    setTesting(true);
    try {
      await configApi.testEndpoint(localEndpoint);
      showMessage('success', 'Endpoint is reachable');
    } catch (error) {
      console.error('Endpoint test failed:', error);
      showMessage('error', 'Endpoint is not reachable');
    } finally {
      setTesting(false);
    }
  };

  if (loading) {
    return <div className="p-6">Loading configuration...</div>;
  }

  return (
    <div className="p-6">
      <h1 className="text-3xl font-bold mb-6">Configuration</h1>

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
        <h2 className="text-xl font-bold mb-4">Local Webhook Endpoint</h2>

        <div className="mb-4">
          <label htmlFor="localEndpoint" className="block text-gray-700 text-sm font-bold mb-2">
            Local Webhook URL
          </label>
          <input
            type="url"
            id="localEndpoint"
            value={localEndpoint}
            onChange={(e) => setLocalEndpoint(e.target.value)}
            placeholder="http://localhost:3000/webhook"
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <p className="mt-2 text-sm text-gray-600">
            This is the endpoint where webhooks will be forwarded after being received from the server.
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
            onClick={handleTest}
            disabled={testing}
            className="bg-green-500 text-white py-2 px-4 rounded-md hover:bg-green-600 focus:outline-none focus:ring-2 focus:ring-green-500 disabled:opacity-50"
          >
            {testing ? 'Testing...' : 'Test Connectivity'}
          </button>
        </div>
      </div>

      {config && (
        <div className="bg-white rounded-lg shadow-md p-6 mt-6">
          <h2 className="text-xl font-bold mb-4">Current Configuration</h2>
          <div className="space-y-2">
            <div className="flex justify-between">
              <span className="text-gray-600">Local Endpoint:</span>
              <span className="font-medium">{config.local_endpoint}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-600">Max Retries:</span>
              <span className="font-medium">{config.retry_config.max_retries}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-600">Retry Delay:</span>
              <span className="font-medium">{config.retry_config.retry_delay}ms</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-600">Backoff Multiplier:</span>
              <span className="font-medium">{config.retry_config.backoff_multiplier}x</span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
