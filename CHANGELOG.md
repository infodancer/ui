# Changelog

All notable changes to `infodancer/ui` are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Until `v1.0`, breaking changes can land in any release; consumers pin to a specific tag and treat upgrades deliberately. See the Versioning section in [DESIGN.md](DESIGN.md) for what counts as breaking.

## Unreleased

## v0.4.1 — 2026-05-29

### Added
- **Gated, multilevel nav menu.** `ui/nav` grows from a flat link list into a menu that gates each item on the viewer's auth state and roles, nests into dropdowns, and supports icon affordances with live state (the notification bell). New types: `Viewer{Authenticated, EmailVerified, Roles}` (with `HasRole`); `Gate` (AND-combined `RequireAuth`/`RequireAnon`/`RequireVerified`/`RequireRoles` (+`RequireAllRoles`)/`CustomGate`); `MenuItem` (`Kind` link/icon/separator, `Children`, `Badge`); `Registry` of named custom predicates; `Badge` (count/label/state/htmx `PollURL`). `Resolve(nav, viewer, reg)` filters a configured menu to what the viewer may see — fail-closed on a missing custom gate, empty dropdowns dropped, an icon with no surviving action kept muted, separators tidied. `ParseMenu` decodes the declarative JSON config (structs also carry `yaml` tags so a YAML host unmarshals into them without ui taking a YAML dependency). Dropdowns are CSS-only via native `<details>` — no JavaScript. ui defines its own `Viewer` rather than importing `github.com/infodancer/authz`, keeping the dependency surface at the standard library; an `authz.Principal` adapts in one line. New `base.css` chrome: `.app-nav-item`, `.app-nav-dropdown`, `.app-nav-menu`, `.app-nav-sep`, `.app-nav-bell` (`--muted`), `.app-nav-badge`. Hugo variant deferred (gating is a Go-consumer capability) — same documented exception as `ui/document`.

### Changed
- **`NavData` gains `Items []MenuItem`** alongside the existing `Links`. The `ui/nav` partial renders `Items` when present and falls back to `Links` otherwise, so existing flat-nav consumers are unaffected. `NavLink` / `NavData.Links` are now deprecated in favour of `Items`.

## v0.4.0 — 2026-05-27

### Added
- **Layer 2 base document template.** New `ui/document` partial (Go variant) renders a complete `<html>` page from a `DocumentData`, wiring in the Layer 1 pieces — ui CSS (open-props → tokens → base, then the consumer's `ExtraCSS`), `ui/meta`, `ui/nav`/`ui/footer`/`ui/sidebar`, the optional `ui/analytics` scripts, and htmx (surfaced through the `HeadTags` field, so htmx stays opt-in). The page body comes from template blocks the consumer defines — `content` (required), plus optional `title`, `head`, `nav`, `footer` — so a consumer that wants the full shell stops hand-rolling its own skeleton. Layer 2 is just another Layer 1 consumer; the dependency only flows Layer 2 → Layer 1. See the "Layer 1 vs Layer 2" section in [DESIGN.md](DESIGN.md) for the block contract.
- **analytics component.** New `ui/analytics` partial + `Analytics`/`Umami`/`Plausible` types render the optional web-analytics `<head>` scripts. ui owns the markup; the consumer passes its own script URLs and IDs, and a `nil` member emits nothing. `ui/document` includes it when `Analytics` is set; it can also be rendered standalone.
- **Hugo variants of `ui/document` and `ui/analytics` are deliberately deferred** (documented in DESIGN.md) — added when a Hugo consumer needs the full shell.

## v0.3.1 — 2026-05-25

### Added
- **meta component (SEO / social head tags).** New `ui/meta` partial (Go + Hugo) renders the standard discovery markup from a `Meta` data shape: `<meta name="description">`, `<link rel="canonical">`, the OpenGraph block (`og:type` defaulting to `website`, `og:title/description/url/site_name/image/locale`), and a Twitter card (`summary_large_image` when an image is set, else `summary`). Every field is optional — an empty field emits nothing. The `JSONLD` slot carries complete `<script type="application/ld+json">` elements so a page can emit several schema.org graphs; the new `JSONLD(v any)` helper marshals a value and wraps it, escaping `<`, `>`, `&` so the data can't break out of the script element. ui renders the tags but owns none of the content — the description copy, canonical policy, and schema graphs are the consumer's domain. Like the other partials it's pure markup with no base-template dependency; a page that doesn't render it emits nothing.
- **htmx component (opt-in).** ui now ships [htmx](https://htmx.org) v2.0.10 as its chosen interactivity layer, vendored under `assets/js/` and served via the existing `AssetsFS()`. `HeadTags(staticBase)` emits the `<script>` tag with a Subresource Integrity hash (self-hosted, no CDN). Go request/response helpers cover the boundary every htmx consumer otherwise re-implements: `IsRequest`, `IsBoosted`, `Target` (read `HX-*` request headers) and `Redirect`, `Refresh`, `PushURL`, `Retarget`, `Reswap`, `Trigger` (set `HX-*` response headers). `HTMXVersion` exposes the pinned version. This is "Layer 1" — pure mechanism with **no dependency on a base template**; a consumer that doesn't call `HeadTags` loads no JS. The DESIGN.md JavaScript exclusion is superseded; see the new "Interactivity: htmx" section.
- **Hugo variant of htmx.** `layouts/partials/htmx-head.html` is the Hugo counterpart of `HeadTags` — it `resources.GetMatch`es the vendored `js/htmx-*.min.js`, `fingerprint`s it (sha384), and emits the SRI'd `<script>`. The Hugo-computed integrity matches the Go side's pinned SRI (same bytes), so both consumer paths serve byte-identical htmx. Opt-in the same way: a site that doesn't call the partial loads no JS.

## v0.2.1 — 2026-05-23

### Added
- **Sidebar component.** `.app-sidebar-layout` grids a content column with an optional left and/or right aside (`.has-left` / `.has-right`; neither = full-width); `--app-sidebar-width` token sizes the asides. `.app-sidebar` is a quiet bordered panel of collapsible link sections — `.app-sidebar-section` is a native `<details>` (collapse needs no JS), `.app-sidebar-feed` lists links each with an optional `.app-sidebar-meta` line and `.app-sidebar-count` pill. New `ui/sidebar` partial (Go + Hugo) renders one side from a `SidebarData` (`SidebarSection`/`SidebarItem`); each section emits `data-sidebar-key` so a consumer can persist open/closed state with its own script. Clean default skin; consumers re-skin via token overrides (osg paints it as a parchment scroll).

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
