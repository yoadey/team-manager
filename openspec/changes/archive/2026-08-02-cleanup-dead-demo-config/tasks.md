## 1. Frontend

- [x] 1.1 `config.ts`: remove `storageKeyPrefix`, `mockDelayMin`,
      `mockDelayMax` fields and their `import.meta.env` reads (and the
      now-unused `numberEnv` helper)
- [x] 1.2 `frontend/.env.example`: remove `VITE_STORAGE_KEY_PREFIX`,
      `VITE_MOCK_DELAY_MIN`, `VITE_MOCK_DELAY_MAX`
- [x] 1.3 `config.test.ts`: remove the now-obsolete assertions for these
      variables
- [x] 1.4 `CLAUDE.md`: delete the two now-nonexistent rows from the
      frontend env-var table
- [x] 1.5 (found during implementation) `src/mocks/handlers.ts`: fix a
      stale comment referencing `VITE_MOCK_DELAY_MIN/MAX` as if the MSW
      demo backend still honored it

## 2. Verification

- [x] 2.1 `npm run typecheck`
- [x] 2.2 `npm test` (`config.test.ts` — full suite covered by CI)
- [x] 2.3 `npm run lint` (scoped: `eslint` on changed files)
