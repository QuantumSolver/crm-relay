import axios from 'axios';

// Use relative path to work with any domain
const API_BASE_URL = import.meta.env.VITE_API_URL || '';

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add auth token to requests
api.interceptors.request.use((config) => {
  // Read token from zustand persist storage
  const authStorage = localStorage.getItem('auth-storage');
  if (authStorage) {
    try {
      const parsed = JSON.parse(authStorage);
      const token = parsed?.state?.token;
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
    } catch (e) {
      console.error('Failed to parse auth storage:', e);
    }
  }
  return config;
});

// Handle auth errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Don't intercept 401 from login endpoint — let the auth store handle it
      const requestUrl = error.config?.url || '';
      if (requestUrl.includes('/api/auth/login')) {
        return Promise.reject(error);
      }
      // Clear auth storage for other 401s (expired token, etc.)
      localStorage.removeItem('auth-storage');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  user: {
    id: string;
    username: string;
    role: string;
  };
  expires_at: number;
}

export interface APIKey {
  id: string;
  name: string;
  key: string;
  platform: string;
  created_at: string;
  updated_at: string;
  is_active: boolean;
}

export interface WebhookEndpoint {
  id: string;
  platform: string;
  path: string;
  http_method: string;
  headers: Record<string, string>;
  retry_config: {
    max_retries: number;
    retry_delay: number;
    retry_multiplier: number;
  };
  created_at: string;
  updated_at: string;
}

export interface Metrics {
  webhooks_received: number;
  webhooks_processed: number;
  webhooks_failed: number;
  webhooks_retried: number;
  queue_depth: number;
  pending_messages: number;
  average_latency_ms: number;
  last_webhook_time: string;
}

// Auth API
export const authApi = {
  login: async (data: LoginRequest): Promise<LoginResponse> => {
    const response = await api.post<LoginResponse>('/api/auth/login', data);
    return response.data;
  },

  getCurrentUser: async () => {
    const response = await api.get('/api/auth/me');
    return response.data;
  },
};

// API Keys API
export const apiKeysApi = {
  list: async (): Promise<{ api_keys: APIKey[] }> => {
    const response = await api.get<{ api_keys: APIKey[] }>('/api/keys');
    return response.data;
  },

  create: async (data: { name: string; platform: string }): Promise<APIKey> => {
    const response = await api.post<APIKey>('/api/keys', data);
    return response.data;
  },

  update: async (id: string, data: { name?: string; is_active?: boolean }): Promise<APIKey> => {
    const response = await api.put<APIKey>(`/api/keys/${id}`, data);
    return response.data;
  },

  delete: async (id: string): Promise<{ success: boolean; message: string }> => {
    const response = await api.delete<{ success: boolean; message: string }>(`/api/keys/${id}`);
    return response.data;
  },
};

// Endpoints API
export const endpointsApi = {
  list: async (): Promise<{ endpoints: WebhookEndpoint[] }> => {
    const response = await api.get<{ endpoints: WebhookEndpoint[] }>('/api/endpoints');
    return response.data;
  },

  create: async (data: {
    platform: string;
    path: string;
    http_method: string;
    headers?: Record<string, string>;
  }): Promise<WebhookEndpoint> => {
    const response = await api.post<WebhookEndpoint>('/api/endpoints', data);
    return response.data;
  },

  update: async (
    id: string,
    data: {
      platform?: string;
      path?: string;
      http_method?: string;
      headers?: Record<string, string>;
    }
  ): Promise<WebhookEndpoint> => {
    const response = await api.put<WebhookEndpoint>(`/api/endpoints/${id}`, data);
    return response.data;
  },

  delete: async (id: string): Promise<{ success: boolean; message: string }> => {
    const response = await api.delete<{ success: boolean; message: string }>(`/api/endpoints/${id}`);
    return response.data;
  },
};

// Metrics API
export const metricsApi = {
  get: async (): Promise<Metrics> => {
    const response = await api.get<Metrics>('/api/metrics');
    return response.data;
  },

  getQueueDepth: async (): Promise<{ queue_depth: number }> => {
    const response = await api.get<{ queue_depth: number }>('/api/queue-depth');
    return response.data;
  },

  getPendingMessages: async (): Promise<{ pending_messages: number }> => {
    const response = await api.get<{ pending_messages: number }>('/api/pending-messages');
    return response.data;
  },
};

export default api;
