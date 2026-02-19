import { useEffect, useState } from 'react';
import { endpointsApi, WebhookEndpoint } from '../lib/api';

export default function Endpoints() {
  const [endpoints, setEndpoints] = useState<WebhookEndpoint[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [editingEndpoint, setEditingEndpoint] = useState<WebhookEndpoint | null>(null);
  const [formData, setFormData] = useState({
    platform: '',
    path: '',
    http_method: 'POST',
    headers: '{}',
  });

  const fetchEndpoints = async () => {
    try {
      const data = await endpointsApi.list();
      setEndpoints(data.endpoints);
    } catch (error) {
      console.error('Failed to fetch endpoints:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchEndpoints();
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      let headers = {};
      try {
        headers = JSON.parse(formData.headers);
      } catch (err) {
        alert('Invalid JSON for headers');
        return;
      }
      await endpointsApi.create({
        ...formData,
        headers,
      });
      setShowCreateModal(false);
      setFormData({ platform: '', path: '', http_method: 'POST', headers: '{}' });
      fetchEndpoints();
    } catch (error) {
      console.error('Failed to create endpoint:', error);
    }
  };

  const handleUpdate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingEndpoint) return;
    try {
      let headers = editingEndpoint.headers;
      if (formData.headers !== '{}') {
        try {
          headers = JSON.parse(formData.headers);
        } catch (err) {
          alert('Invalid JSON for headers');
          return;
        }
      }
      await endpointsApi.update(editingEndpoint.id, {
        platform: formData.platform || editingEndpoint.platform,
        path: formData.path || editingEndpoint.path,
        http_method: formData.http_method || editingEndpoint.http_method,
        headers,
      });
      setShowEditModal(false);
      setEditingEndpoint(null);
      setFormData({ platform: '', path: '', http_method: 'POST', headers: '{}' });
      fetchEndpoints();
    } catch (error) {
      console.error('Failed to update endpoint:', error);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this endpoint?')) return;
    try {
      await endpointsApi.delete(id);
      fetchEndpoints();
    } catch (error) {
      console.error('Failed to delete endpoint:', error);
    }
  };

  const openEditModal = (endpoint: WebhookEndpoint) => {
    setEditingEndpoint(endpoint);
    setFormData({
      platform: endpoint.platform,
      path: endpoint.path,
      http_method: endpoint.http_method,
      headers: JSON.stringify(endpoint.headers, null, 2),
    });
    setShowEditModal(true);
  };

  if (loading) {
    return <div className="p-6">Loading endpoints...</div>;
  }

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">Webhook Endpoints</h1>
        <button
          onClick={() => setShowCreateModal(true)}
          className="bg-blue-500 text-white px-4 py-2 rounded-md hover:bg-blue-600"
        >
          Create Endpoint
        </button>
      </div>

      <div className="bg-white rounded-lg shadow-md overflow-hidden">
        <table className="min-w-full">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Platform
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Path
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Method
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Headers
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Created
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Actions
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {endpoints.map((endpoint) => (
              <tr key={endpoint.id}>
                <td className="px-6 py-4 whitespace-nowrap">{endpoint.platform}</td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <code className="text-sm bg-gray-100 px-2 py-1 rounded">
                    {endpoint.path}
                  </code>
                </td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <span
                    className={`px-2 py-1 rounded text-xs font-medium ${
                      endpoint.http_method === 'POST'
                        ? 'bg-blue-100 text-blue-800'
                        : endpoint.http_method === 'GET'
                        ? 'bg-green-100 text-green-800'
                        : 'bg-gray-100 text-gray-800'
                    }`}
                  >
                    {endpoint.http_method}
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {Object.keys(endpoint.headers).length} headers
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {new Date(endpoint.created_at).toLocaleDateString()}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm space-x-2">
                  <button
                    onClick={() => openEditModal(endpoint)}
                    className="text-blue-500 hover:text-blue-700"
                  >
                    Edit
                  </button>
                  <button
                    onClick={() => handleDelete(endpoint.id)}
                    className="text-red-500 hover:text-red-700"
                  >
                    Delete
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Create Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h2 className="text-xl font-bold mb-4">Create Webhook Endpoint</h2>
            <form onSubmit={handleCreate}>
              <div className="mb-4">
                <label className="block text-gray-700 text-sm font-bold mb-2">
                  Platform
                </label>
                <input
                  type="text"
                  value={formData.platform}
                  onChange={(e) => setFormData({ ...formData, platform: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md"
                  required
                />
              </div>
              <div className="mb-4">
                <label className="block text-gray-700 text-sm font-bold mb-2">
                  Path
                </label>
                <input
                  type="text"
                  value={formData.path}
                  onChange={(e) => setFormData({ ...formData, path: e.target.value })}
                  placeholder="/webhook/platform"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md"
                  required
                />
              </div>
              <div className="mb-4">
                <label className="block text-gray-700 text-sm font-bold mb-2">
                  HTTP Method
                </label>
                <select
                  value={formData.http_method}
                  onChange={(e) => setFormData({ ...formData, http_method: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md"
                >
                  <option value="POST">POST</option>
                  <option value="GET">GET</option>
                  <option value="PUT">PUT</option>
                  <option value="DELETE">DELETE</option>
                </select>
              </div>
              <div className="mb-4">
                <label className="block text-gray-700 text-sm font-bold mb-2">
                  Headers (JSON)
                </label>
                <textarea
                  value={formData.headers}
                  onChange={(e) => setFormData({ ...formData, headers: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md font-mono text-sm"
                  rows={4}
                />
              </div>
              <div className="flex justify-end space-x-2">
                <button
                  type="button"
                  onClick={() => setShowCreateModal(false)}
                  className="px-4 py-2 border border-gray-300 rounded-md hover:bg-gray-50"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 bg-blue-500 text-white rounded-md hover:bg-blue-600"
                >
                  Create
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Edit Modal */}
      {showEditModal && editingEndpoint && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h2 className="text-xl font-bold mb-4">Edit Webhook Endpoint</h2>
            <form onSubmit={handleUpdate}>
              <div className="mb-4">
                <label className="block text-gray-700 text-sm font-bold mb-2">
                  Platform
                </label>
                <input
                  type="text"
                  value={formData.platform}
                  onChange={(e) => setFormData({ ...formData, platform: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md"
                  required
                />
              </div>
              <div className="mb-4">
                <label className="block text-gray-700 text-sm font-bold mb-2">
                  Path
                </label>
                <input
                  type="text"
                  value={formData.path}
                  onChange={(e) => setFormData({ ...formData, path: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md"
                  required
                />
              </div>
              <div className="mb-4">
                <label className="block text-gray-700 text-sm font-bold mb-2">
                  HTTP Method
                </label>
                <select
                  value={formData.http_method}
                  onChange={(e) => setFormData({ ...formData, http_method: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md"
                >
                  <option value="POST">POST</option>
                  <option value="GET">GET</option>
                  <option value="PUT">PUT</option>
                  <option value="DELETE">DELETE</option>
                </select>
              </div>
              <div className="mb-4">
                <label className="block text-gray-700 text-sm font-bold mb-2">
                  Headers (JSON)
                </label>
                <textarea
                  value={formData.headers}
                  onChange={(e) => setFormData({ ...formData, headers: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md font-mono text-sm"
                  rows={4}
                />
              </div>
              <div className="flex justify-end space-x-2">
                <button
                  type="button"
                  onClick={() => {
                    setShowEditModal(false);
                    setEditingEndpoint(null);
                  }}
                  className="px-4 py-2 border border-gray-300 rounded-md hover:bg-gray-50"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 bg-blue-500 text-white rounded-md hover:bg-blue-600"
                >
                  Update
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
