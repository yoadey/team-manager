import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { usePushActions } from './usePushActions';

// Length chosen so the base64url-decode path (urlBase64ToUint8Array) sees a
// clean multiple-of-4 input with no padding, avoiding the class of encoding
// edge cases that aren't the point of these tests.
vi.mock('@/config', () => ({ config: { vapidPublicKey: 'test-vapid-public-key123' } }));

describe('usePushActions', () => {
  let toastMsg: ReturnType<typeof vi.fn>;
  let api: { push: { subscribe: ReturnType<typeof vi.fn>; unsubscribe: ReturnType<typeof vi.fn> } };
  let getSubscription: ReturnType<typeof vi.fn>;
  let subscribe: ReturnType<typeof vi.fn>;
  let requestPermission: ReturnType<typeof vi.fn>;
  let pushManager: { getSubscription: typeof getSubscription; subscribe: typeof subscribe };

  beforeEach(() => {
    toastMsg = vi.fn();
    api = { push: { subscribe: vi.fn().mockResolvedValue(undefined), unsubscribe: vi.fn().mockResolvedValue(undefined) } };
    getSubscription = vi.fn().mockResolvedValue(null);
    subscribe = vi.fn().mockResolvedValue({
      endpoint: 'https://push.example/abc',
      toJSON: () => ({ endpoint: 'https://push.example/abc', keys: { p256dh: 'p', auth: 'a' } }),
    });
    pushManager = { getSubscription, subscribe };
    requestPermission = vi.fn().mockResolvedValue('granted');

    vi.stubGlobal('navigator', {
      serviceWorker: { ready: Promise.resolve({ pushManager }) },
    });
    vi.stubGlobal('PushManager', function () {});
    vi.stubGlobal('Notification', { requestPermission });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it('reports supported when a VAPID key is configured and the browser has serviceWorker + PushManager', async () => {
    const { result } = renderHook(() => usePushActions(api as never, toastMsg as never));
    expect(result.current.support).toBe('supported');
    await waitFor(() => expect(result.current.subscribed).toBe(false));
  });

  it('detects an existing subscription on mount', async () => {
    getSubscription.mockResolvedValue({ endpoint: 'https://push.example/existing' });
    const { result } = renderHook(() => usePushActions(api as never, toastMsg as never));
    await waitFor(() => expect(result.current.subscribed).toBe(true));
  });

  it('enablePush requests permission, subscribes, and registers with the backend', async () => {
    const { result } = renderHook(() => usePushActions(api as never, toastMsg as never));
    await waitFor(() => expect(result.current.subscribed).toBe(false));

    await act(async () => {
      await result.current.enablePush();
    });

    expect(requestPermission).toHaveBeenCalled();
    expect(subscribe).toHaveBeenCalledWith(
      expect.objectContaining({ userVisibleOnly: true, applicationServerKey: expect.any(Uint8Array) }),
    );
    expect(api.push.subscribe).toHaveBeenCalledWith({ endpoint: 'https://push.example/abc', keys: { p256dh: 'p', auth: 'a' } });
    expect(result.current.subscribed).toBe(true);
    expect(toastMsg).toHaveBeenCalledWith('Push-Benachrichtigungen aktiviert');
  });

  it('enablePush shows an error toast and does not subscribe when permission is denied', async () => {
    requestPermission.mockResolvedValue('denied');
    const { result } = renderHook(() => usePushActions(api as never, toastMsg as never));
    await waitFor(() => expect(result.current.subscribed).toBe(false));

    await act(async () => {
      await result.current.enablePush();
    });

    expect(subscribe).not.toHaveBeenCalled();
    expect(api.push.subscribe).not.toHaveBeenCalled();
    expect(result.current.subscribed).toBe(false);
    expect(toastMsg).toHaveBeenCalledWith(expect.stringContaining('blockiert'), undefined, 'error');
  });

  it('disablePush unsubscribes locally and unregisters with the backend', async () => {
    const unsubscribe = vi.fn().mockResolvedValue(true);
    getSubscription.mockResolvedValue({ endpoint: 'https://push.example/existing', unsubscribe });
    const { result } = renderHook(() => usePushActions(api as never, toastMsg as never));
    await waitFor(() => expect(result.current.subscribed).toBe(true));

    await act(async () => {
      await result.current.disablePush();
    });

    expect(unsubscribe).toHaveBeenCalled();
    expect(api.push.unsubscribe).toHaveBeenCalledWith('https://push.example/existing');
    expect(result.current.subscribed).toBe(false);
    expect(toastMsg).toHaveBeenCalledWith('Push-Benachrichtigungen deaktiviert');
  });

  it('disablePush is a no-op against the backend when there is no local subscription', async () => {
    const { result } = renderHook(() => usePushActions(api as never, toastMsg as never));
    await waitFor(() => expect(result.current.subscribed).toBe(false));

    await act(async () => {
      await result.current.disablePush();
    });

    expect(api.push.unsubscribe).not.toHaveBeenCalled();
    expect(result.current.subscribed).toBe(false);
  });
});
