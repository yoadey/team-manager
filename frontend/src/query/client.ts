import { QueryClient } from '@tanstack/react-query';
import { AuthError, ForbiddenError, ValidationError } from '@/utils/errors';

/** True for errors the server won't resolve on its own, so retrying is pointless. */
function isUnretryable(error: unknown): boolean {
  return error instanceof AuthError || error instanceof ForbiddenError || error instanceof ValidationError;
}

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (failureCount, error) => !isUnretryable(error) && failureCount < 2,
    },
    mutations: {
      retry: false,
    },
  },
});
