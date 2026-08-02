## 1. Frontend

- [ ] 1.1 `config.ts`: remove `storageKeyPrefix`, `mockDelayMin`,
      `mockDelayMax` fields and their `import.meta.env` reads
- [ ] 1.2 `frontend/.env.example`: remove `VITE_STORAGE_KEY_PREFIX`,
      `VITE_MOCK_DELAY_MIN`, `VITE_MOCK_DELAY_MAX`
- [ ] 1.3 `config.test.ts`: remove the now-obsolete assertions for these
      variables
- [ ] 1.4 `CLAUDE.md`: delete the two now-nonexistent rows from the
      frontend env-var table

## 2. Verification

- [ ] 2.1 `npm run typecheck`
- [ ] 2.2 `npm test`
- [ ] 2.3 `npm run lint`
