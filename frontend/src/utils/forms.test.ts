import { describe, it, expect, vi } from 'vitest';
import { formValues, clearBusyIfOwned } from './forms';

interface SampleForm extends Record<string, unknown> {
  title: string;
  count: number;
  flag: boolean;
}

describe('formValues', () => {
  it('returns the form buffer as a typed partial view', () => {
    const state = { form: { title: 'Hello', count: 3, flag: true } };
    const f = formValues<SampleForm>(state);
    expect(f.title).toBe('Hello');
    expect(f.count).toBe(3);
    expect(f.flag).toBe(true);
  });

  it('reflects an empty buffer as an empty object (fields undefined)', () => {
    const state = { form: {} };
    const f = formValues<SampleForm>(state);
    expect(f.title).toBeUndefined();
    expect(f.count).toBeUndefined();
  });

  it('returns the same underlying object reference (no copy)', () => {
    const form = { title: 'x' };
    const state = { form };
    expect(formValues<SampleForm>(state)).toBe(form);
  });
});

describe('clearBusyIfOwned', () => {
  it('clears busy when it still holds the value this action set', () => {
    const setState = vi.fn();
    const S = () => ({ busy: 'save' });
    clearBusyIfOwned(S, setState, 'save');
    expect(setState).toHaveBeenCalledWith({ busy: null });
  });

  // Regression test: a save and a delete used to both unconditionally clear
  // `busy` to null once their own request resolved. Since `busy` is one
  // shared string across the whole app (every Save button reads
  // `busy === 'save'`), a delete resolving while a differently-typed save is
  // still in flight (or vice versa) would incorrectly re-enable the other
  // action's UI mid-request, inviting a double-submit.
  it('does NOT clear busy when a different action has since taken it over', () => {
    const setState = vi.fn();
    const S = () => ({ busy: 'save' }); // a save started after this delete began
    clearBusyIfOwned(S, setState, 'delete');
    expect(setState).not.toHaveBeenCalled();
  });

  it('does NOT clear busy when it is already null (no self-clobber on a no-op)', () => {
    const setState = vi.fn();
    const S = () => ({ busy: null });
    clearBusyIfOwned(S, setState, 'save');
    expect(setState).not.toHaveBeenCalled();
  });
});
