// Contract test guarding the dual service-layer implementation. The app ships
// two implementations of the same `api` surface — the localStorage mock
// (serviceLayer.ts) and the real HTTP client (serviceLayerReal.ts) — and they
// must stay in lockstep: any namespace/method added to one but not the other is
// a latent bug (the mock passes tests while production 404s, or vice versa).
//
// In the test environment VITE_API_BASE_URL is unset, so the exported `api`
// resolves to the mock; realApi is imported directly. This test fails loudly if
// their shapes diverge.
import { describe, it, expect } from 'vitest';
import { api as mockApi } from './serviceLayer';
import { realApi } from './serviceLayerReal';

type Shape = Record<string, string[]>;

/**
 * Maps each namespace to its sorted list of public method names. Underscore-
 * prefixed members are implementation-private helpers (e.g. the mock's `_mk`)
 * and are excluded — only the public, app-facing surface is part of the contract.
 */
function shapeOf(obj: Record<string, unknown>): Shape {
  const shape: Shape = {};
  for (const [key, value] of Object.entries(obj)) {
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      shape[key] = Object.entries(value as Record<string, unknown>)
        .filter(([k, v]) => typeof v === 'function' && !k.startsWith('_'))
        .map(([k]) => k)
        .sort();
    }
  }
  return shape;
}

describe('service-layer contract (mock vs real)', () => {
  const mock = shapeOf(mockApi as unknown as Record<string, unknown>);
  const real = shapeOf(realApi as unknown as Record<string, unknown>);

  it('exposes the same namespaces', () => {
    expect(Object.keys(mock).sort()).toEqual(Object.keys(real).sort());
  });

  it.each(Object.keys(real))('namespace "%s" exposes the same methods', (ns) => {
    expect(mock[ns]).toEqual(real[ns]);
  });
});
