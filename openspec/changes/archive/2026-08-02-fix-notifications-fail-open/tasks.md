## 1. Backend

- [x] 1.1 `service.go`: change `NotificationModule`'s `default:` case so
      `HasReadAccess` rejects it (returns a new `unclassifiedModule`
      sentinel that isn't one of the six known module strings, so it
      falls through to `HasReadAccess`'s existing fail-closed `default:`)
- [x] 1.2 `service_test.go`: add a case for an unrecognized notification
      type, asserting it is excluded from `List` even for a member with
      write access to every known module

## 2. Verification

- [x] 2.1 `cd backend && make test-unit`
- [x] 2.2 `cd backend && make lint`
