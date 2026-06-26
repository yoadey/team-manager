import createClient from 'openapi-fetch';
import type { paths } from './types.gen';

// The session is carried by an HttpOnly, encrypted cookie set by the backend.
// `credentials: 'include'` ensures the cookie is sent on every request.
export const apiClient = createClient<paths>({
  baseUrl: (import.meta.env.VITE_API_BASE_URL ?? '') + '/api/v1',
  credentials: 'include',
});
