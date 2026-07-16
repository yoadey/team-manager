// Node MSW server used by src/test/setup.ts to intercept the generated
// openapi-fetch client's requests in Vitest (jsdom environment).
import { setupServer } from 'msw/node';
import { handlers } from './handlers';

export const server = setupServer(...handlers);
