# Security Policy

## Reporting a Vulnerability

Use [GitHub's private vulnerability reporting](https://github.com/infodancer/ui/security/advisories/new) for any security-relevant issue. Public issue tracking is fine for non-security bugs.

## Scope

This repository hosts three modules; each has its own security model.

**Root `ui` module** (HTML + CSS, plus vendored htmx):

- CSS that affects content-injection safety in consumer sites (e.g., a selector that breaks a consumer's escaping assumptions).
- Hugo or Go partials that emit consumer-controlled content without proper escaping (the partials use the standard Hugo / Go html/template auto-escaping; an issue here would be a regression).
- The vendored htmx bundle is pinned and SRI-hashed (see its provenance notes); an integrity or version concern there is in scope.

**Nested modules** carry the security-sensitive code and document their own models:

- [`markdown/SECURITY.md`](markdown/SECURITY.md) — the sanitization boundary (goldmark + bluemonday, XSS corpus).
- [`mdedit/SECURITY.md`](mdedit/SECURITY.md) — the editor component and its vendored editor JS (single server-side renderer, pinned/SRI assets, client-side file load).

Out of scope: server-side code in consumers (their responsibility).
