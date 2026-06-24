// Global test setup executed once per test file before the suite runs.
// Registers jest-dom matchers (toBeInTheDocument, etc.) for React Testing
// Library assertions and clears persisted state between tests so that the
// localStorage-backed service layer always starts from a deterministic seed.
import '@testing-library/jest-dom/vitest';
import { afterEach, beforeEach } from 'vitest';
import { cleanup } from '@testing-library/react';

beforeEach(() => {
  localStorage.clear();
});

afterEach(() => {
  cleanup();
});
