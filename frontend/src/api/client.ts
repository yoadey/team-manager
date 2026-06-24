import createClient from 'openapi-fetch';
import type { paths } from './types.gen';

const TOKEN_KEY = 'tv_jwt';

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY);
}

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

export const apiClient = createClient<paths>({
  baseUrl: (import.meta.env.VITE_API_BASE_URL ?? '') + '/api/v1',
});

apiClient.use({
  onRequest({ request }) {
    const token = getToken();
    if (token) {
      request.headers.set('Authorization', `Bearer ${token}`);
    }
    return request;
  },
  onResponse({ response }) {
    if (response.status === 401) {
      clearToken();
    }
    return response;
  },
});
