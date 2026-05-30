# Security Policy — mdedit module

## Reporting

This module is part of the [infodancer/ui](../README.md) monorepo; follow the
repository-wide policy in [the root SECURITY.md](../SECURITY.md). Use
[GitHub's private vulnerability reporting](https://github.com/infodancer/ui/security/advisories/new).
Do not open public issues for security problems.

## Security model

`mdedit` treats all Markdown source as untrusted.

- **One sanitization boundary.** Rendering is delegated to the
  [`markdown`](../markdown/README.md) module: goldmark parses, then bluemonday
  filters against an allowlist. bluemonday is the boundary regardless of
  goldmark options — even with raw HTML passthrough enabled (`Rich`),
  script/style/iframe/object/embed/form and every `on*` handler are stripped,
  and dangerous URL schemes (`javascript:`, `vbscript:`, `data:text/html`,
  `file:`) are rejected. Its XSS corpus runs against every preset; see that
  module's [SECURITY.md](../markdown/SECURITY.md).

- **Single renderer.** The server is the only thing that renders Markdown to
  HTML. The client editor's own preview is disabled, so previews and stored
  output are produced by the same audited code — no client/server divergence
  for an attacker to exploit.

- **File load adds no server surface.** When `Field.AllowFileLoad` is set, the
  author can load a local Markdown file into the editor. It is read in the
  browser (`FileReader`) and placed into the textarea; it is never uploaded as
  a file. It reaches the server only via the normal Save POST and is rendered
  and sanitized identically to typed Markdown. A size cap and a NUL-byte check
  keep an oversized or binary file out of the page.

- **Pinned, self-contained client assets.** Vendored editor files are pinned
  to a specific version with recorded Subresource Integrity hashes
  (`assets/vendor/PROVENANCE.md`); the emitted `<script>`/`<link>` tags carry
  those hashes. No runtime CDN, no npm dependency tree. The editor never
  fetches an icon font from a third party.

- **Boundaries this module does not own.** Authentication, authorization, and
  CSRF protection on the Save/Preview endpoints are the host application's
  responsibility. `mdedit` renders and sanitizes; it does not decide who may
  edit. Image upload (when added) is likewise a host concern — see
  [DESIGN.md](DESIGN.md) for the contract and the host's security obligations.

- **Output contract.** Only the sanitized output of a `markdown.Renderer` may be
  cast to `template.HTML` for user-authored fields. The `Field.HTML` shown in
  display mode must come from the renderer, never from raw author input.
