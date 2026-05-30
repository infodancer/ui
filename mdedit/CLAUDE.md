# CLAUDE.md

Guidance for Claude Code sessions in the `mdedit` module
(`github.com/infodancer/ui/mdedit`).

## What this is

A reusable Markdown display/edit component for the Go + htmx consumers in the
infodancer / matthewjhunter portfolio (osg, faq, future blog/timeline). It
ships: client assets behind an editor adapter seam (`assets/`), Go
html/template partials (`partials/`), and the `Field` data shape. Rendering and
sanitizing live in the sibling [`markdown`](../markdown/README.md) module (the
shared, independently-versioned security boundary); this module depends on it
only conceptually — the host calls `markdown` and passes the result in as
`Field.HTML`. A **nested module** in the [infodancer/ui](../CLAUDE.md) monorepo
with its own `go.mod` and tag (`mdedit/v0.1.x`). See [DESIGN.md](DESIGN.md) and
[README.md](README.md).

It is deliberately small and **host-agnostic about storage, auth, and
authorization** — the consuming app owns those. Editing is a Go + htmx concern
only; there is no Hugo variant (a static site has no backend to store edits).

## What NOT to change without explicit approval

- **Public API surface.** `AssetsFS`, `PartialsFS`, `HeadTags`, and the `Field`
  type are public. The partial template names (`mdedit/display`, `mdedit/edit`,
  `mdedit/preview`) and the `Field` field names are consumed by host templates —
  renaming or removing is breaking. Adding fields is fine.
- **The adapter seam contract** in `assets/mdedit.js` (`mdedit.register` and
  the controller shape: `getValue`/`setValue`/`sync`/`destroy`/`onChange`).
  Adapters and hosts depend on it.
- **Vendored assets.** `assets/vendor/*` are pinned third-party files with
  recorded SRI hashes. To upgrade, re-fetch per `PROVENANCE.md`, recompute the
  hashes, update the constants in `mdedit.go`, and review the diff.

## Conventions

- License: Apache-2.0.
- Go: track the latest patched release per the infodancer org standard.
- No build step for client assets: hand-written JS/CSS plus vendored prebuilt
  bundles. No npm, no bundler. CSS uses the repo's `--app-*` tokens with
  fallbacks.
- The server is the only Markdown renderer. Client-side preview (e.g.
  EasyMDE's marked.js) stays disabled so what authors see matches what is
  stored and sanitized.

## Build (nested module in a monorepo)

The repo `go.work` shadows the parent infodancer workspace so in-tree builds
resolve. The `examples/spike` is its own module with `replace` directives — run
it with `GOWORK=off go run .`. CI fans out over the module matrix with
`GOWORK=off`.

## Workflow

Per the org standard: GitHub issue before branching (`feature/<n>` or
`bug/<n>`), commits reference the issue, PRs merge to main.

## Status notes

- The osg and faq migrations onto `markdown` (the reason the sibling module
  exists) are **done**; this editor layers the UI on top.
- **File load** (`Field.AllowFileLoad`) is shipped: load a local `.md` file
  into the editor client-side; never uploaded.
- **Image support** is deferred and split three ways — `markdown` decides what
  `<img>`/`src` survives (a planned relative-only policy), mdedit inserts a
  `![](url)` reference (a future `Field.UploadURL` + adapter `imageUploadFunction`),
  and a separate future media module stores/serves the bytes. mdedit never
  touches storage. See [DESIGN.md](DESIGN.md) and
  [infodancer/ui#14](https://github.com/infodancer/ui/issues/14).
