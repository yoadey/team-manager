import { describe, it, expect } from 'vitest';
import { formValues } from './forms';

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
