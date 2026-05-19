# CLAUDE.md

Guidance for Claude Code sessions in `github.com/infodancer/ui`.

## What this repo is

A small CSS + template library: design tokens, a base stylesheet, and `nav`/`footer` partials shipped in parallel Hugo and Go html/template variants. Consumed by feature modules (faq, planned blog, timeline) and consumer sites in the infodancer/matthewjhunter portfolio. See [DESIGN.md](DESIGN.md) for the design proposal, scope, and rationale.

This is **not** a component framework or a comprehensive design system. It's deliberately small: the tokens are the public API, and components are extracted from feature modules only when duplication forces it.

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
- No JS. The library is HTML + CSS. Interactivity is the consumer's problem.

## Two-consumer integration

Hugo and Go html/template are first-class equals. Any change that would advantage one consumer pattern over the other needs justification. The parallel-partials approach (separate `.html` and `.gohtml` files) is the result of a deliberate decision documented in DESIGN.md — don't collapse them into one without revisiting that decision.

## Versioning

CSS token renames and partial-shape changes are breaking. See the Versioning section in DESIGN.md. Until v1.0, consumers pin to a specific tag.

## Related docs

- [DESIGN.md](DESIGN.md) — the source of truth for the v0.1 design.
- [`infodancer/infodancer/docs/web-portfolio-architecture.md`](https://github.com/infodancer/infodancer/blob/master/docs/web-portfolio-architecture.md) — the portfolio-level context.
