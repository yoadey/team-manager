## 1. Backend

- [ ] 1.1 `service.go`: change `NotificationModule`'s `default:` case so
      `HasReadAccess` rejects it (e.g. return a module value that never
      matches a granted permission, or restructure the two functions so
      "unclassified" is handled once, fail-closed, in one place)
- [ ] 1.2 `service_test.go`: add a case for an unrecognized notification
      type, asserting it is excluded from `List` for a member with only
      `read`/`none` on ordinary modules (and, separately, that a
      settings-admin's blanket access is unaffected if that's how the
      existing fail-closed path is structured)

## 2. Verification

- [ ] 2.1 `cd backend && make test-unit`
- [ ] 2.2 `cd backend && make lint`
