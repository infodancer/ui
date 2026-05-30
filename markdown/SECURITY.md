# Security Policy — markdown module

## Reporting

This module is part of the [infodancer/ui](../README.md) monorepo; follow the
repository-wide policy in [the root SECURITY.md](../SECURITY.md). Use
[GitHub's private vulnerability reporting](https://github.com/infodancer/ui/security/advisories/new).
Do not open public issues for security problems.

## Security model

This module treats all Markdown source as untrusted and is itself a security
boundary — it is the one place the portfolio's sanitization policy is defined.

- **bluemonday is the boundary, always.** goldmark parses; bluemonday filters
  against an allowlist. The boundary holds regardless of the goldmark options:
  even with raw-HTML passthrough (`Rich`), script/style/iframe/object/embed/
  form and every `on*` event-handler attribute are stripped, and dangerous URL
  schemes (`javascript:`, `vbscript:`, `data:text/html`, `file:`) are rejected.

- **The corpus is the contract.** `markdown_test.go` carries an XSS corpus that
  runs against every preset, including the raw-HTML-enabled one. Changes to the
  policy must keep it green; new attack vectors are added to it, never removed.

- **Output contract.** Only the sanitized output of a `Renderer` may be cast to
  `template.HTML`. Callers must not cast raw author input.

- **The `LinkChecker` hook is trusted code.** It runs *after* bluemonday, as an
  additive gate on already-sanitized links, so it cannot weaken the core
  sanitization of element/attribute structure. But an href it *rewrites*
  (`LinkDecision.Href`) is used verbatim and is not re-sanitized — a consuming
  application that rewrites a link to a dangerous scheme owns that result. The
  hook is for tightening or redirecting links, not a place to relax the policy.

- **Image `src` policy (planned).** Strict/Rich currently sit on bluemonday's
  UGCPolicy, which permits `<img>` with http/https/relative `src` — so external
  image URLs survive today, a tracking/referer-leak vector. A planned
  `ImageRelativeOnly`-style policy (mirroring `LinkRelativeOnly`) makes
  relative/same-origin the default with external behind an explicit opt-in. See
  [infodancer/ui#14](https://github.com/infodancer/ui/issues/14).

- **Not in scope.** This module renders and sanitizes. Authentication,
  authorization, CSRF, and storage belong to the consuming application.
