import { useEffect, useState } from 'react';
import { apiKeysApi, APIKey } from '../lib/api';

export default function APIKeys() {
  const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showEditModal, setShowEditModal] = useState(false);
  const [editingKey, setEditingKey] = useState<APIKey | null>(null);
  const [formData, setFormData] = useState({ name: '', platform: '' });

  const fetchApiKeys = async () => {
    try {
      const data = await apiKeysApi.list();
      setApiKeys(data.api_keys);
    } catch (error) {
      console.error('Failed to fetch API keys:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchApiKeys();
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await apiKeysApi.create(formData);
      setShowCreateModal(false);
      setFormData({ name: '', platform: '' });
      fetchApiKeys();
    } catch (error) {
      console.error('Failed to create API key:', error);
    }
  };

  const handleUpdate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingKey) return;
    try {
      await apiKeysApi.update(editingKey.id, {
        name: formData.name || editingKey.name,
        is_active: editingKey.is_active,
      });
      setShowEditModal(false);
      setEditingKey(null);
      setFormData({ name: '', platform: '' });
      fetchApiKeys();
    } catch (error) {
      console.error('Failed to update API key:', error);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this API key?')) return;
    try {
      await apiKeysApi.delete(id);
      fetchApiKeys();
    } catch (error) {
      console.error('Failed to delete API key:', error);
    }
  };

  const handleToggleActive = async (key: APIKey) => {
    try {
      await apiKeysApi.update(key.id, { is_active: !key.is_active });
      fetchApiKeys();
    } catch (error) {
      console.error('Failed to toggle API key:', error);
    }
  };

  const openEditModal = (key: APIKey) => {
    setEditingKey(key);
    setFormData({ name: key.name, platform: key.platform });
    setShowEditModal(true);
  };

  const copyToClipboard = (key: string) => {
    navigator.clipboard.writeText(key);
    alert('API key copied to clipboard!');
  };

  if (loading) {
    return <div className="p-6">Loading API keys...</div>;
  }

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">API Keys</h1>
        <button
          onClick={() => setShowCreateModal(true)}
          className="bg-blue-500 text-white px-4 py-2 rounded-md hover:bg-blue-600"
        >
          Create API Key
        </button>
      </div>

      <div className="bg-white rounded-lg shadow-md overflow-hidden">
        <table className="min-w-full">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Name
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Platform
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Key
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Status
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
            {apiKeys.map((key) => (
              <tr key={key.id}>
                <td className="px-6 py-4 whitespace-nowrap">{key.name}</td>
                <td className="px-6 py-4 whitespace-nowrap">{key.platform}</td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <div className="flex items-center space-x-2">
                    <code className="text-sm bg-gray-100 px-2 py-1 rounded">
                      {key.key.substring(0, 8)}...
                    </code>
                    <button
                      onClick={() => copyToClipboard(key.key)}
                      className="text-blue-500 hover:text-blue-700"
                    >
                      Copy
                    </button>
                  </div>
                </td>
                <td className="px-6 py-4 whitespace-nowrap">
                  <span
                    className={`px-2 py-1 rounded-full text-xs ${
                      key.is_active
                        ? 'bg-green-100 text-green-800'
                        : 'bg-red-100 text-red-800'
                    }`}
                  >
                    {key.is_active ? 'Active' : 'Inactive'}
                  </span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {new Date(key.created_at).toLocaleDateString()}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm space-x-2">
                  <button
                    onClick={() => openEditModal(key)}
                    className="text-blue-500 hover:text-blue-700"
                  >
                    Edit
                  </button>
                  <button
                    onClick={() => handleToggleActive(key)}
                    className="text-yellow-500 hover:text-yellow-700"
                  >
                    {key.is_active ? 'Disable' : 'Enable'}
                  </button>
                  <button
                    onClick={() => handleDelete(key.id)}
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
            <h2 className="text-xl font-bold mb-4">Create API Key</h2>
            <form onSubmit={handleCreate}>
              <div className="mb-4">
                <label className="block text-gray-700 text-sm font-bold mb-2">
                  Name
                </label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md"
                  required
                />
              </div>
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
      {showEditModal && editingKey && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h2 className="text-xl font-bold mb-4">Edit API Key</h2>
            <form onSubmit={handleUpdate}>
              <div className="mb-4">
                <label className="block text-gray-700 text-sm font-bold mb-2">
                  Name
                </label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md"
                  required
                />
              </div>
              <div className="flex justify-end space-x-2">
                <button
                  type="button"
                  onClick={() => {
                    setShowEditModal(false);
                    setEditingKey(null);
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
