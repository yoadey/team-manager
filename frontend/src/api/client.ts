import createClient from 'openapi-fetch';
import type { paths } from './types.gen';
import { NetworkError } from '@/utils/errors';

// The session is carried by an HttpOnly, encrypted cookie set by the backend.
// `credentials: 'include'` ensures the cookie is sent on every request.
export const apiClient = createClient<paths>({
  baseUrl: (import.meta.env.VITE_API_BASE_URL ?? '') + '/api/v1',
  credentials: 'include',
});

// A rejected fetch() (offline, DNS failure, CORS block, aborted request) never
// reaches serviceLayerReal's check()/checkOk() — those only classify errors
// carried in a *resolved* result (result.error / result.response.status).
// Without this, a genuine connectivity failure surfaces as a raw, untyped
// error: reportActionError falls through to its generic branch (an
// unlocalized "Failed to fetch" toast instead of the translated
// error.network message) and retryable() never retries it, since it only
// retries NetworkError. This onError middleware normalizes every such
// rejection into a NetworkError so both paths work as intended.
apiClient.use({
  onError({ error }) {
    return new NetworkError(error instanceof Error ? error.message : 'Network error');
  },
});
