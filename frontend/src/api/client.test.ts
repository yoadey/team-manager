// Verifies the onError middleware registered in client.ts: a rejected
// fetch() (offline, DNS failure, CORS block) must surface as a NetworkError,
// not a raw/untyped exception — see client.ts for why this matters
// (reportActionError's typed branch and retryable() both key off NetworkError).
//
// openapi-fetch captures `globalThis.fetch` once at createClient() time, so
// vi.stubGlobal('fetch', ...) after module load has no effect on apiClient —
// the per-request `fetch` option is used instead to inject the failing fetch.
import { describe, it, expect } from 'vitest';
import { NetworkError } from '@/utils/errors';
import { apiClient } from './client';

describe('apiClient onError middleware', () => {
  it('converts a rejected fetch() into a NetworkError', async () => {
    const failingFetch = () => Promise.reject(new TypeError('Failed to fetch'));

    await expect(
      apiClient.GET('/auth/me', { baseUrl: 'http://api.test', fetch: failingFetch }),
    ).rejects.toThrow(NetworkError);
  });

  it('preserves the underlying error message', async () => {
    const failingFetch = () => Promise.reject(new TypeError('network offline'));

    await expect(
      apiClient.GET('/auth/me', { baseUrl: 'http://api.test', fetch: failingFetch }),
    ).rejects.toThrow('network offline');
  });
});
