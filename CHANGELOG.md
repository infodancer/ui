# Changelog

All notable changes to `infodancer/ui` are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Until `v1.0`, breaking changes can land in any release; consumers pin to a specific tag and treat upgrades deliberately. See the Versioning section in [DESIGN.md](DESIGN.md) for what counts as breaking.

## Unreleased

## v0.2.0 — 2026-05-23

First tagged release. The earlier nav/footer partials and `--app-*` token
scaffolding were the conceptual v0.1; this is the first version a consumer
(osg) pins to a tag.

### Added
- `FooterData.BrandURL` (Go) and `.Site.Params.ui.brand_url` (Hugo) so the footer brand mark can link somewhere other than `/`. Defaults to `/` when empty, matching prior behavior.
- **Open Props as the value layer.** `assets/css/open-props.css` vendors a pinned copy of [Open Props](https://open-props.style/) v1.7.23 (no CDN at runtime, no npm in the build). `tokens.css` now sources its values from Open Props vars (`--gray-*`, `--size-*`, `--radius-*`, shadows) instead of hand-picked constants. The `--app-*` names are unchanged — consumers and feature modules still bind only to `--app-*`, never to Open Props vars directly. **Load order is now significant:** `open-props.css` → `tokens.css` → `base.css` → site overrides.
- **Dark theme.** `tokens.css` ships dark values, selected by `<html data-theme="dark">` (explicit, wins over OS) or `prefers-color-scheme: dark` (when no explicit `data-theme="light"`). Lets a site default to dark (osg) without fighting the OS setting.
- **Elevation tokens** `--app-shadow`, `--app-shadow-lg` (bound to Open Props shadows). Previously deferred; added now that osg needs raised surfaces.
- **Component classes** extracted for the osg directory integration: `.app-btn` / `.app-btn-secondary` (buttons-as-class, so `<a>` can be an action), `.app-tabs` / `.app-tab` (section tab strip with accent underline), `.app-panel` / `.app-panel-accent` (framed secondary region), `.app-page-header` / `.app-page-meta`, `.app-section`.

### Changed
- DESIGN.md no longer states a fixed token count in prose; the canonical roster is the `tokens.css` file and the assertion list in `ui_test.go`. This avoids drift when tokens are added.
- `NavUser` data shape in DESIGN.md reduced to `DisplayName` to match the shipped Go struct and the rendered partial output. The earlier `MenuURL` / `SignOutURL` fields were spec-only and never implemented.
- Bare `<button>` text color now uses `--app-color-accent-on` (was `--app-color-bg-raised`, which only worked when raised happened to be white).
