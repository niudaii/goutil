# seckit (`github.com/niudaii/goutil/seckit`)

Independent Go submodule providing reusable HTTP client, OpenID Connect cookie auth, notification, MITRE CVE client, and HTTP server lifecycle primitives, isolated from the root `github.com/niudaii/goutil` `go.mod` dependency tree.

## Packages

- `httpx`: Standardized `net/http` client builder with timeouts, redirect control, proxy URL support, retry/backoff (honoring `Retry-After` and context cancellation), bounded body reads, and JSON helpers.
- `mitre`: MITRE CVE v5 API client (`cveawg.mitre.org`) with CVSS v4.0/v3.1/v3.0/v2.0 scoring, CNA/ADP fallback, bounded TTL cache, and product extraction.
- `notify`: POPO Webhook and OpenAPI Robot notification clients with rune/UTF-8-byte message splitting, file upload with retry, and structured error logging + error return.
- `oidc` & `oidc/ginoidc`: Stateless AES-256-GCM encrypted cookie OpenID Connect client supporting `net/http` and Gin, HS256 and RS256 (JWKS) ID token verification, strict origin/CSRF checks, `X-API-Key` validation, and bounded local revocation for logout/replay defense.
- `serverx` & `serverx/ginserverx`: `signal.NotifyContext` graceful HTTP shutdown, independent `/healthz` probes, panic recovery, and security response headers distinguishing immutable hashed `/assets/*` from `no-store` HTML/API responses.
- `ui` & `ui/ginui`: Shared SecOps Refined Slate-Blue design tokens (`secops-tokens.css`), CSP-safe shell helpers (`secops-shell.js`: theme switcher, non-blocking toasts, clipboard copy, search adornments, CVE links, `/` & `Esc` shortcuts, suite nav), real-time configurable suite navigation (`/secops/nav.json`), and unified AuthGate login page template (`RenderAuthPage`).

## Real-Time Suite Navigation (`ui.ResolveNav`)

`/secops/nav.json` resolves navigation items dynamically on each request (with `Cache-Control: no-store`) in the following priority order:

1. `SECOPS_NAV_URL`: Remote HTTP/HTTPS JSON endpoint (`[]NavItem` or `{"items": [...]}`), cached in memory for `SECOPS_NAV_CACHE_TTL` (default `30s`, minimum `1s`) with a `2s` request timeout and stale-while-error fallback. Parses JSON regardless of `Content-Type` and supports static `.html` endpoints (such as `nav.json.html`) serving either raw JSON or `<script type="application/json">...</script>`.
2. `SECOPS_NAV_FILE`: Local JSON file path watched by `mtime` + size for immediate hot reload without restarting services.
3. `SECOPS_NAV_JSON`: Inline JSON string environment override.
4. `Config.Defaults` + per-item `SECOPS_NAV_<KEY>_URL` overrides (`"-"` hides an item).

## Installation

Released as a Go submodule under tag `seckit/vX.Y.Z`:

```sh
go get github.com/niudaii/goutil/seckit@v0.3.0
```
