import { useState, useEffect } from 'react';
import { dlqApi, DLQMessage } from '../lib/api';

export default function DLQ() {
  const [messages, setMessages] = useState<DLQMessage[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedMessage, setSelectedMessage] = useState<DLQMessage | null>(null);
  const [actionLoading, setActionLoading] = useState<{ [key: string]: boolean }>({});
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const fetchMessages = async () => {
    try {
      const data = await dlqApi.getMessages();
      setMessages(data);
    } catch (error) {
      console.error('Failed to fetch DLQ messages:', error);
      showMessage('error', 'Failed to load DLQ messages');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchMessages();
  }, []);

  const showMessage = (type: 'success' | 'error', text: string) => {
    setMessage({ type, text });
    setTimeout(() => setMessage(null), 5000);
  };

  const handleReplay = async (id: string) => {
    setActionLoading({ ...actionLoading, [id]: true });
    try {
      await dlqApi.replayMessage(id);
      showMessage('success', 'Message replayed successfully');
      await fetchMessages();
    } catch (error) {
      console.error('Failed to replay message:', error);
      showMessage('error', 'Failed to replay message');
    } finally {
      setActionLoading({ ...actionLoading, [id]: false });
    }
  };

  const handleDelete = async (id: string) => {
    if (!window.confirm('Are you sure you want to delete this message?')) {
      return;
    }
    setActionLoading({ ...actionLoading, [id]: true });
    try {
      await dlqApi.deleteMessage(id);
      showMessage('success', 'Message deleted successfully');
      await fetchMessages();
    } catch (error) {
      console.error('Failed to delete message:', error);
      showMessage('error', 'Failed to delete message');
    } finally {
      setActionLoading({ ...actionLoading, [id]: false });
    }
  };

  const handleBulkReplay = async () => {
    if (!window.confirm(`Are you sure you want to replay all ${messages.length} messages?`)) {
      return;
    }
    setActionLoading({ bulk: true });
    try {
      await Promise.all(messages.map((msg) => dlqApi.replayMessage(msg.id)));
      showMessage('success', 'All messages replayed successfully');
      await fetchMessages();
    } catch (error) {
      console.error('Failed to replay messages:', error);
      showMessage('error', 'Failed to replay some messages');
    } finally {
      setActionLoading({ bulk: false });
    }
  };

  const formatMessage = (message: string) => {
    try {
      const parsed = JSON.parse(message);
      return JSON.stringify(parsed, null, 2);
    } catch {
      return message;
    }
  };

  if (loading) {
    return <div className="p-6">Loading DLQ messages...</div>;
  }

  return (
    <div className="p-6">
      <h1 className="text-3xl font-bold mb-6">Dead Letter Queue</h1>

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
        <div className="flex justify-between items-center mb-4">
          <h2 className="text-xl font-bold">
            Failed Messages ({messages.length})
          </h2>
          {messages.length > 0 && (
            <button
              onClick={handleBulkReplay}
              disabled={actionLoading.bulk}
              className="bg-green-500 text-white py-2 px-4 rounded-md hover:bg-green-600 focus:outline-none focus:ring-2 focus:ring-green-500 disabled:opacity-50"
            >
              {actionLoading.bulk ? 'Replaying...' : 'Replay All'}
            </button>
          )}
        </div>

        {messages.length === 0 ? (
          <p className="text-gray-600">No failed messages in the queue.</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full">
              <thead>
                <tr className="border-b">
                  <th className="text-left py-2 px-4">Timestamp</th>
                  <th className="text-left py-2 px-4">Error</th>
                  <th className="text-left py-2 px-4">Retries</th>
                  <th className="text-left py-2 px-4">Actions</th>
                </tr>
              </thead>
              <tbody>
                {messages.map((msg) => (
                  <tr key={msg.id} className="border-b">
                    <td className="py-2 px-4">
                      {new Date(msg.timestamp).toLocaleString()}
                    </td>
                    <td className="py-2 px-4 max-w-xs truncate">
                      {msg.error}
                    </td>
                    <td className="py-2 px-4">{msg.retry_count}</td>
                    <td className="py-2 px-4">
                      <div className="flex space-x-2">
                        <button
                          onClick={() => setSelectedMessage(msg)}
                          className="bg-blue-500 text-white py-1 px-3 rounded text-sm hover:bg-blue-600"
                        >
                          View
                        </button>
                        <button
                          onClick={() => handleReplay(msg.id)}
                          disabled={actionLoading[msg.id]}
                          className="bg-green-500 text-white py-1 px-3 rounded text-sm hover:bg-green-600 disabled:opacity-50"
                        >
                          {actionLoading[msg.id] ? '...' : 'Replay'}
                        </button>
                        <button
                          onClick={() => handleDelete(msg.id)}
                          disabled={actionLoading[msg.id]}
                          className="bg-red-500 text-white py-1 px-3 rounded text-sm hover:bg-red-600 disabled:opacity-50"
                        >
                          {actionLoading[msg.id] ? '...' : 'Delete'}
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {selectedMessage && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-md p-6 max-w-4xl w-full max-h-[80vh] overflow-auto">
            <div className="flex justify-between items-center mb-4">
              <h2 className="text-xl font-bold">Message Details</h2>
              <button
                onClick={() => setSelectedMessage(null)}
                className="text-gray-500 hover:text-gray-700"
              >
                ✕
              </button>
            </div>
            <div className="space-y-4">
              <div>
                <h3 className="font-bold mb-2">Message ID</h3>
                <p className="text-gray-600">{selectedMessage.id}</p>
              </div>
              <div>
                <h3 className="font-bold mb-2">Timestamp</h3>
                <p className="text-gray-600">
                  {new Date(selectedMessage.timestamp).toLocaleString()}
                </p>
              </div>
              <div>
                <h3 className="font-bold mb-2">Error</h3>
                <p className="text-red-600">{selectedMessage.error}</p>
              </div>
              <div>
                <h3 className="font-bold mb-2">Retry Count</h3>
                <p className="text-gray-600">{selectedMessage.retry_count}</p>
              </div>
              <div>
                <h3 className="font-bold mb-2">Message Payload</h3>
                <pre className="bg-gray-100 p-4 rounded overflow-auto max-h-96">
                  {formatMessage(selectedMessage.message)}
                </pre>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
