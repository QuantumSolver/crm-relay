import axios, { AxiosInstance, AxiosError } from 'axios';

// API base URL - use relative path to work with any domain
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '';

// Types
export interface User {
  username: string;
  created_at: string;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  user: User;
}

export interface Metrics {
  WebhooksReceived: number;
  WebhooksProcessed: number;
  WebhooksFailed: number;
  WebhooksRetried: number;
  AverageLatency: number;
  LastWebhookTime: string;
}

export interface DLQMessage {
  id: string;
  message: string;
  error: string;
  timestamp: string;
  retry_count: number;
}

export interface ConfigResponse {
  local_endpoint: string;
  retry_config: {
    max_retries: number;
    retry_delay: number;
    backoff_multiplier: number;
  };
}

// Create axios instance
const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add auth token to requests
apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Handle 401 errors
apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

// Auth API
export const authApi = {
  login: async (credentials: LoginRequest): Promise<LoginResponse> => {
    const response = await apiClient.post<LoginResponse>('/api/auth/login', credentials);
    return response.data;
  },

  getCurrentUser: async (): Promise<User> => {
    const response = await apiClient.get<User>('/api/auth/me');
    return response.data;
  },
};

// Config API
export const configApi = {
  get: async (): Promise<ConfigResponse> => {
    const response = await apiClient.get<ConfigResponse>('/api/config');
    return response.data;
  },

  updateLocalEndpoint: async (endpoint: string): Promise<void> => {
    await apiClient.put('/api/config/local-endpoint', { local_endpoint: endpoint });
  },

  updateRetryConfig: async (config: {
    max_retries: number;
    retry_delay: number;
    backoff_multiplier: number;
  }): Promise<void> => {
    await apiClient.put('/api/config/retry', config);
  },

  testEndpoint: async (endpoint: string): Promise<void> => {
    await apiClient.post('/api/config/test-endpoint', { endpoint });
  },
};

// DLQ API
export const dlqApi = {
  getMessages: async (): Promise<DLQMessage[]> => {
    const response = await apiClient.get<DLQMessage[]>('/api/dlq');
    return response.data;
  },

  replayMessage: async (messageId: string): Promise<void> => {
    await apiClient.post(`/api/dlq/${messageId}/replay`);
  },

  deleteMessage: async (messageId: string): Promise<void> => {
    await apiClient.delete(`/api/dlq/${messageId}`);
  },
};

// Metrics API
export const metricsApi = {
  get: async (): Promise<Metrics> => {
    const response = await apiClient.get<Metrics>('/api/metrics');
    return response.data;
  },
};

export default apiClient;
