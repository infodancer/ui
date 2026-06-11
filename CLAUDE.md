# CLAUDE.md

Guidance for Claude Code sessions in `github.com/infodancer/ui`.

## What this repo is

A small monorepo of three independently-tagged Go modules for the
infodancer/matthewjhunter portfolio's front end. The **root `ui` module** is a
CSS + template library: design tokens, a base stylesheet, and
`nav`/`footer`/`sidebar`/`meta` partials shipped in parallel Hugo and Go
html/template variants, plus vendored htmx. Two **nested modules** live
alongside it, each with its own `go.mod`, tag, `DESIGN`/`README`/`CLAUDE`, and
guidance:

- `markdown/` — `github.com/infodancer/ui/markdown`, the audited goldmark +
  bluemonday sanitizer. See [markdown/CLAUDE.md](markdown/CLAUDE.md).
- `mdedit/` — `github.com/infodancer/ui/mdedit`, the Markdown editor component
  (Go partials + vendored editor JS). See [mdedit/CLAUDE.md](mdedit/CLAUDE.md).

When working inside a nested module, `cd` into it and follow its CLAUDE.md. The
guidance below covers the **root `ui` module** specifically. See
[DESIGN.md](DESIGN.md) for the design proposal, scope, and rationale.

The root module is **not** a component framework or a comprehensive design
system. It's deliberately small: the tokens are the public API, and components
are extracted from feature modules only when duplication forces it.

## What NOT to change without explicit approval

- **Token names.** `--app-color-*`, `--app-font-*`, `--app-space-*`, `--app-radius-*`, `--app-max-width-*` are the public API. Renaming or removing a token breaks every consumer site and feature module. Adding a new token is fine.
- **Partial class hierarchy** (`.app-nav`, `.app-nav-brand`, `.app-nav-links`, `.app-nav-auth`, `.app-footer`, `.app-footer-brand`, `.app-footer-copyright`, `.app-footer-links`). Consumer site CSS targets these selectors for overrides; renaming breaks consumers.
- **Partial data shapes.** `NavData`, `FooterData` field names (Go); `.Site.Params.ui.*` and `.Site.Menus.*` keys (Hugo). Adding fields is fine; renaming or removing is breaking.
- **Public Go API surface in `ui.go`.** `AssetsFS()`, `PartialsFS()`, and the `NavData`/`FooterData` types are public; treat with the same versioning discipline as tokens.

## Conventions

- License: Apache-2.0. Don't change without explicit approval.
- Go version: track the latest patched release per the infodancer org standard. See [CONTRIBUTING.md](CONTRIBUTING.md).
- CSS: hand-written, no preprocessor, no bundler. Two files: `tokens.css` and `base.css`.
- Comments in CSS describe *why* a rule exists, not *what* it does. Token names already communicate purpose.
- **Minimal authored JS is allowed in the root module, but only for small, opt-in, cross-cutting *mechanism*** — the kind every consumer would otherwise re-implement identically (or pull from npm), with no per-site logic baked in. The action tracker (`track.js`, served via `AssetsFS`, emitted by `TrackerHead`) is the reference example: ~60 lines, framework-agnostic, declarative `data-track` attributes, vocabulary owned by the consumer. The image lightbox (`lightbox.js` / `LightboxHead`, declarative `data-lightbox` attributes on plain links, native `<dialog>`) is the second occupant of the category. Substantial or component-level interactivity (editors, widgets, anything stateful or opinionated) still belongs in a feature module (see `mdedit`) or the consumer — *"interactivity is the consumer's problem"* remains the default. Vendored third-party JS (htmx) stays opt-in via `HeadTags`. When in doubt, keep it out of root.

## Two-consumer integration

Hugo and Go html/template are first-class equals. Any change that would advantage one consumer pattern over the other needs justification. The parallel-partials approach (separate `.html` and `.gohtml` files) is the result of a deliberate decision documented in DESIGN.md — don't collapse them into one without revisiting that decision.

## Versioning

CSS token renames and partial-shape changes are breaking. See the Versioning section in DESIGN.md. Until v1.0, consumers pin to a specific tag.

## Related docs

- [DESIGN.md](DESIGN.md) — the source of truth for the v0.1 design.
- [`infodancer/infodancer/docs/web-portfolio-architecture.md`](https://github.com/infodancer/infodancer/blob/master/docs/web-portfolio-architecture.md) — the portfolio-level context.
