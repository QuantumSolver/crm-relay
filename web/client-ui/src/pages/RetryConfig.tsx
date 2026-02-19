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
      setMaxRetries(data.retry_config?.max_retries ?? 3);
      setRetryDelay(data.retry_config?.retry_delay ?? 1000);
      setBackoffMultiplier(data.retry_config?.backoff_multiplier ?? 2);
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
    return (
      <div className="flex items-center justify-center h-64">
        <div className="flex flex-col items-center gap-4">
          <svg className="animate-spin h-10 w-10 text-emerald-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          <p className="text-gray-600 font-medium">Loading configuration...</p>
        </div>
      </div>
    );
  }

  const retrySchedule = calculateRetrySchedule();

  return (
    <div className="space-y-8 animate-fade-in">
      <div>
        <h1 className="text-3xl font-bold text-gray-900 mb-2">Retry Configuration</h1>
        <p className="text-gray-600">Configure webhook retry behavior with exponential backoff</p>
      </div>

      {message && (
        <div
          className={`flex items-center gap-3 px-4 py-3 rounded-lg ${message.type === 'success'
              ? 'bg-green-50 border border-green-200 text-green-700'
              : 'bg-red-50 border border-red-200 text-red-700'
            }`}
        >
          {message.type === 'success' ? (
            <svg className="w-5 h-5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
            </svg>
          ) : (
            <svg className="w-5 h-5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
            </svg>
          )}
          {message.text}
        </div>
      )}

      <div className="bg-white rounded-xl shadow-md p-6">
        <h2 className="text-xl font-bold text-gray-900 mb-6 flex items-center gap-2">
          <svg className="w-6 h-6 text-emerald-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          Retry Settings
        </h2>

        <div className="space-y-6">
          <div>
            <label htmlFor="maxRetries" className="block text-sm font-semibold text-gray-700 mb-2">
              Max Retries
            </label>
            <input
              type="number"
              id="maxRetries"
              value={maxRetries}
              onChange={(e) => setMaxRetries(parseInt(e.target.value) || 0)}
              min="0"
              max="10"
              className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
            />
            <p className="mt-2 text-sm text-gray-600 flex items-start gap-2">
              <svg className="w-4 h-4 text-gray-400 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
              </svg>
              Maximum number of retry attempts for failed webhooks.
            </p>
          </div>

          <div>
            <label htmlFor="retryDelay" className="block text-sm font-semibold text-gray-700 mb-2">
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
              className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
            />
            <p className="mt-2 text-sm text-gray-600 flex items-start gap-2">
              <svg className="w-4 h-4 text-gray-400 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
              </svg>
              Initial delay before the first retry attempt.
            </p>
          </div>

          <div>
            <label htmlFor="backoffMultiplier" className="block text-sm font-semibold text-gray-700 mb-2">
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
              className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
            />
            <p className="mt-2 text-sm text-gray-600 flex items-start gap-2">
              <svg className="w-4 h-4 text-gray-400 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
              </svg>
              Multiplier for exponential backoff between retries.
            </p>
          </div>
        </div>

        <div className="flex gap-4 mt-8">
          <button
            onClick={handleSave}
            disabled={saving}
            className="inline-flex items-center gap-2 bg-gradient-to-r from-emerald-500 to-teal-600 text-white py-2.5 px-6 rounded-lg hover:from-emerald-600 hover:to-teal-700 focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed font-medium shadow-md hover:shadow-lg transition-all duration-200"
          >
            {saving ? (
              <>
                <svg className="animate-spin h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                Saving...
              </>
            ) : (
              <>
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                Save Configuration
              </>
            )}
          </button>
          <button
            onClick={handleReset}
            className="inline-flex items-center gap-2 bg-white text-gray-700 border border-gray-300 py-2.5 px-6 rounded-lg hover:bg-gray-50 hover:border-gray-400 focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:ring-offset-2 font-medium shadow-sm transition-all duration-200"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            Reset to Defaults
          </button>
        </div>
      </div>

      <div className="bg-white rounded-xl shadow-md p-6">
        <h2 className="text-xl font-bold text-gray-900 mb-6 flex items-center gap-2">
          <svg className="w-6 h-6 text-emerald-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
          </svg>
          Retry Schedule Preview
        </h2>
        {retrySchedule.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="min-w-full">
              <thead>
                <tr className="border-b border-gray-200">
                  <th className="text-left py-3 px-4 font-semibold text-gray-700">Attempt</th>
                  <th className="text-left py-3 px-4 font-semibold text-gray-700">Delay</th>
                  <th className="text-left py-3 px-4 font-semibold text-gray-700">Cumulative Time</th>
                </tr>
              </thead>
              <tbody>
                {retrySchedule.map((item, index) => {
                  const cumulativeDelay = retrySchedule.slice(0, index + 1).reduce((sum, s) => {
                    const ms = parseInt(s.delay);
                    return sum + ms;
                  }, 0);
                  return (
                    <tr key={item.attempt} className="border-b border-gray-100 hover:bg-gray-50 transition-colors">
                      <td className="py-3 px-4">
                        <span className="inline-flex items-center gap-2">
                          <span className="w-6 h-6 bg-emerald-100 text-emerald-700 rounded-full flex items-center justify-center text-xs font-semibold">
                            {item.attempt}
                          </span>
                          <span className="font-medium text-gray-900">Retry #{item.attempt}</span>
                        </span>
                      </td>
                      <td className="py-3 px-4 font-mono text-sm text-gray-600">{item.delay}</td>
                      <td className="py-3 px-4 font-mono text-sm text-gray-600">{cumulativeDelay}ms</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="text-center py-8">
            <svg className="w-12 h-12 text-gray-400 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <p className="text-gray-600">No retries will be attempted.</p>
          </div>
        )}
      </div>
    </div>
  );
}
