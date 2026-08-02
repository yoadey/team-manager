## 1. Mailer CRLF defense-in-depth

- [ ] 1.1 `smtp.go`: add a `strings.ContainsAny(s, "\r\n")` guard inside
      `buildMessage`/`send` for `to`/`from`/`subject`, returning an error
      instead of sending if any contains CR/LF
- [ ] 1.2 `smtp_test.go`: cover a `to`/`subject` containing `\r\n` being
      rejected by the mailer itself (not relying on upstream validation)

## 2. Retention job lint noise

- [ ] 2.1 `retention.go`: add `//nolint:gosec` with the existing
      "table/column names are fixed internal literals, never user input"
      justification to the `fmt.Sprintf`-built statements

## 3. Auth handler status code

- [ ] 3.1 `handler.go`: `GetMyPhoto` returns `apierror.Unauthorized`
      (401) instead of `apierror.NotFound` (404) when `UserFromContext`
      fails
- [ ] 3.2 `handler_test.go`: cover the 401 response for this path if not
      already exercised

## 4. OpenAPI version field

- [ ] 4.1 `openapi.yaml`: derive `info.version` from the build version at
      generation time, or remove the field in favor of the existing
      `API-Version`/`Deprecation`/`Sunset` header scheme

## 5. Verification

- [ ] 5.1 `cd backend && make test-unit`
- [ ] 5.2 `cd backend && make lint`
- [ ] 5.3 `cd backend && make generate` clean (no drift) if `openapi.yaml`
      changed
