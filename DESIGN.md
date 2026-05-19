# infodancer/ui — design proposal

*Status: v0.1 initial design proposal, 2026-05-19. Token names and base CSS shapes are intended to settle quickly and then stay stable; component additions are not breaking. See "Versioning" below.*

## Purpose

`infodancer/ui` is the shared visual layer for the web modules and consumer sites in the infodancer/matthewjhunter portfolio (see `infodancer/infodancer/docs/web-portfolio-architecture.md` for the broader picture). It exists so that:

- Feature modules (faq, planned blog, timeline) can render with a default look that *automatically* matches a host site once the site declares its palette and type.
- Consumer sites (sf, osg, amyhunter, hunterfamily, herald, mail webadmin, infodancer.\*) can adopt a coherent look across their own pages without each one redesigning the wheel.
- Hugo-served sites and Go-`html/template`-served sites are equal first-class clients of the same vocabulary. Five of the seven active consumer sites are Hugo-primary today; the goal is not to push them to Go, it's to make `infodancer/ui` work as well in Hugo as it does in Go.

The package is deliberately small. v0.1 ships **design tokens** (CSS custom properties), a **base stylesheet** that uses them, and two canonical **partials** (`nav`, `footer`) in parallel Hugo and Go variants. Components beyond nav/footer are extracted from observed duplication later, not designed up front.

## Scope

### v0.1 includes

- A locked token vocabulary expressed as CSS custom properties under the `--app-*` prefix.
- A base stylesheet (`base.css`) that applies sensible defaults for typography, spacing, color, and the most-common element styles (body, headings, links, lists, code, basic form inputs, hr).
- `nav` and `footer` partials in **two parallel variants**: Hugo (`layouts/partials/nav.html`, `layouts/partials/footer.html`) and Go html/template (`partials/nav.gohtml`, `partials/footer.gohtml`). Both produce visually identical output; their data shapes are documented and idiomatic to their respective worlds.
- A Go module entry point (`ui.go`) that exposes the CSS files and Go partials as `embed.FS` accessors so Go consumers can mount them without forking.
- A Hugo module declaration (`hugo.toml`, plus the conventional `layouts/`, `assets/` directories) so Hugo consumers can import via `[[module.imports]]`.

### v0.1 explicitly excludes

- Cards, buttons-as-system, forms-as-design-system, full color components. Those live inside feature modules until duplication forces extraction. (The small set of recurring utility patterns extracted in v0.1 — badge base, comments, list chrome, sort tabs, tag chips, pager, search row, visually-hidden — *are* in `base.css`; see "Base stylesheet" below.)
- Dark mode. Tokens are designed so a dark variant is *trivial to add later* by overriding `:root` in a `@media (prefers-color-scheme: dark)` block, but v0.1 ships a single light palette by default and lets each consumer supply dark overrides if they want them.
- JavaScript. `infodancer/ui` is HTML + CSS. Interactivity is the consumer's problem (htmx, vanilla JS, whatever).
- Iconography. Consumers ship their own icons; the partials use plain text marks where applicable.
- Layout primitives beyond `max-width-*`. No grid system, no flex helper classes. Components compose with CSS directly.

### Out of scope, period

- A component library targeted at unrelated downstream users. `infodancer/ui` serves *this portfolio*, not the OSS public at large. It's released under Apache-2.0 and discoverable, but the design choices are tuned for the consumers we actually have.
- A build pipeline. CSS ships as-is; no preprocessor, no PostCSS, no bundling.

## The two-consumer model

A consumer that wants to use `infodancer/ui` falls into one of two categories:

### Hugo consumers

A Hugo site adds `github.com/infodancer/ui` as a module import in its config (`config/_default/module.toml` or wherever the site keeps module config). Hugo's module system then makes the partials and assets available:

- Partials resolve from `layouts/partials/nav.html` and `layouts/partials/footer.html` — the consumer calls them as `{{ partial "nav" . }}` and `{{ partial "footer" . }}`.
- CSS resolves from `assets/css/tokens.css` and `assets/css/base.css` — the consumer pipes them through `resources.Get`, optionally bundles with site-specific overrides, and emits a `<link rel="stylesheet">`.

The consumer site contributes its palette by either (a) declaring its own `:root { --app-* … }` in a stylesheet loaded *after* `tokens.css`, or (b) defining its tokens *before* loading `tokens.css` so the consumer's overrides are the inherited defaults. Either order works; we recommend (a) because it makes the override layer explicit.

### Go html/template consumers

A Go service imports `github.com/infodancer/ui` as a Go module. The root package exposes:

- `ui.AssetsFS() fs.FS` — the contents of `assets/` (tokens.css, base.css). Mount via `http.FileServer` under your static path.
- `ui.PartialsFS() fs.FS` — the Go html/template partials. Parse alongside your other templates so they're available as `{{ template "ui/nav" . }}` etc.

The consumer service contributes its palette the same way as Hugo: load `tokens.css` and then load a site-specific stylesheet that overrides the variables. The Go partials take a small documented `nav.Data` / `footer.Data` struct as their context.

### Why parallel partial files

Hugo and Go's html/template share a templating engine — Hugo is built on top of Go's templates — and at first glance one set of partial files could serve both worlds. The reason for parallel files anyway:

- Hugo's idiomatic data access uses `.Site.Params`, `.Site.Menus`, `partial` calls, `i18n`, `resources.Get`. Go consumers don't have those; they have a struct in their template context.
- Writing partials that work in both worlds via lowest-common-denominator syntax produces awkward consumer code on both sides (Hugo callers have to build the unfamiliar struct; Go callers have to fake site-level globals).
- The visual output is identical and the partials are small (~30 lines each). The duplication is acceptable, the per-consumer ergonomics aren't.

If a future maintenance burden makes consolidation attractive, we can revisit. v0.1 prefers idiomatic.

## Token vocabulary

Tokens use the `--app-*` prefix specifically so feature modules can chain through them. A faq stylesheet reads its colors via:

```css
.faq {
  --faq-color-fg: var(--app-color-fg, #1a1a1a);
  --faq-color-accent: var(--app-color-accent, #0b5394);
  /* … */
}
```

With `infodancer/ui` loaded on the page, `--app-color-fg` resolves and the faq surface inherits the host palette. Without it, the fallback constant kicks in and faq still renders fine standalone.

### The list

**Colors (11).** Designed to cover what every consumer site needs without forcing a full palette dictionary up front. More can be added without breaking; renames break every consumer.

| Token | Role |
|---|---|
| `--app-color-bg` | Page background |
| `--app-color-fg` | Primary foreground text |
| `--app-color-bg-raised` | Cards, panels, callouts — surfaces that sit above the page |
| `--app-color-fg-muted` | De-emphasized text: timestamps, captions, meta |
| `--app-color-border` | Separators, hairlines, input borders |
| `--app-color-accent` | Primary action color: links, primary button background |
| `--app-color-accent-hover` | Hover state for accent |
| `--app-color-accent-on` | Foreground color when painted onto an accent fill (e.g. text on an active sort tab, label on a primary button) |
| `--app-color-prose-fg` | Long-form reading text. Often equals `--app-color-fg`; given its own token so consumers with a dedicated reading mode can tune separately. |
| `--app-color-danger` | Errors, destructive actions, validation failures |
| `--app-color-success` | Confirmations, success states |

**Typography (6).** Three font stacks, three sizing/spacing primitives. Most consumer sites override the stacks; the line-height defaults are intentional and rarely overridden.

| Token | Role |
|---|---|
| `--app-font-body` | Body text font stack |
| `--app-font-display` | Headings and display text. May equal body or differ deliberately (sf uses Cormorant for display, Courier Prime for body). |
| `--app-font-mono` | Inline code, code blocks, fixed-width content |
| `--app-font-size-base` | Root font size — sets the scale for everything else via `rem` |
| `--app-line-height-body` | Body / prose line height |
| `--app-line-height-display` | Heading / display line height |

**Spacing (5).** A small scale, doubling at each step. Token names use t-shirt sizes rather than numeric scales because numeric scales create ambiguity about *what number* is the default.

| Token | Default |
|---|---|
| `--app-space-xs` | 4px |
| `--app-space-sm` | 8px |
| `--app-space` | 16px (the default; un-suffixed name) |
| `--app-space-lg` | 32px |
| `--app-space-xl` | 64px |

**Radii (3).**

| Token | Role |
|---|---|
| `--app-radius-sm` | Small radius (inputs, tight components) |
| `--app-radius` | Default radius (cards, buttons) |
| `--app-radius-pill` | Full-pill / capsule shapes |

**Layout (2).** Container width primitives. No grid system.

| Token | Role |
|---|---|
| `--app-max-width-prose` | Optimal reading width (~65ch) |
| `--app-max-width-page` | Page container max width |

### Token vocabulary rationale

The list is short — 27 tokens — because tokens are the *contract* with every consumer. Every name is a public API: renaming or removing one is a breaking change that propagates to every site and every feature module on the next CSS bump. So we keep the list tight, role-named (not value-named — never `--app-blue`), and prefer to add new tokens later from observed need rather than to ship a maximalist vocabulary that mostly goes unused.

Tokens *not* in v0.1 that have been considered and deferred:

- **Shadow scale** (`--app-shadow-sm`, `--app-shadow`, `--app-shadow-lg`). Shadows are highly design-specific; defer until two consumers want the same shadow language.
- **Color palette beyond the 10** (e.g., named hues like `--app-color-info`, `--app-color-warning`). The 10 cover roles; explicit named hues are component-specific and live in components.
- **Z-index scale**. Z-index needs are local to each component; a global scale invites layering bugs.
- **Animation timing tokens** (`--app-duration-fast`, etc.). Defer until a consumer needs them.
- **Breakpoint tokens.** Media queries don't read CSS custom properties cleanly; use the same numeric breakpoints across consumers by convention until that proves wrong.

## Base stylesheet (`base.css`)

`base.css` applies sensible defaults that use the tokens. It owns:

- A minimal CSS reset (box-sizing, margin/padding reset, image responsiveness).
- `html { font-size: var(--app-font-size-base); }` — sets the rem scale.
- `body` — applies body font, line-height, color, background.
- `h1` through `h6` — display font, line-height, margins on a consistent scale using `--app-space-*`.
- `p`, `ul`, `ol`, `dl` — sensible vertical rhythm.
- `a` — accent color, underline, hover.
- `code`, `pre` — mono font, muted-bg surface, padding via `--app-space-xs`.
- `hr` — single hairline using `--app-color-border`.
- `input`, `textarea`, `select`, `button` — minimum to be readable: inherit font, sensible padding, border using `--app-color-border`, radius using `--app-radius-sm`. Not a full forms-as-design-system; just enough to not look broken.
- `.app-container` — a single layout helper: `max-width: var(--app-max-width-page); margin: 0 auto; padding: var(--app-space);`. Enough to center a page; not a grid system.

`base.css` *also* owns the nav/footer CSS (under `.app-nav` and `.app-footer` selectors). The partials carry markup only; styles live alongside the rest of the base sheet so consumers load exactly two files (`tokens.css` + `base.css`) plus their own overrides. Standard CSS cascade order handles consumer overrides cleanly when site CSS loads after `base.css`.

`base.css` ships a small set of **utility class patterns** that recur across feature modules. They live here once instead of being re-implemented per module. Each is opinionated only at the chrome level — colors, spacing, borders — and inherits all token values so consumer overrides flow through:

| Class | Pattern |
|---|---|
| `.app-list-header`, `.app-list-sorts`, `.app-list-empty` | Header strip + sort-tab row + empty state for a list page |
| `.app-sort`, `.app-sort.is-active` | Clickable sort/filter tab; active state paints with `--app-color-accent` + `--app-color-accent-on` |
| `.app-tag-list`, `.app-tag`, `.app-tag-count` | Inline horizontal chip list, single chip, count modifier |
| `.app-search-form`, `.app-search-empty` | Search input row + zero-results message |
| `.app-pager`, `.app-pager-pos` | Pagination row + position indicator |
| `.app-badge` | Inline status pill, base styling only — semantic variants belong to feature modules |
| `.app-card` | Raised surface — padding, bg-raised, border, radius. Composes with module classes (`.app-card faq-q-card`) for layout. |
| `.app-card-grid` | Responsive auto-fill grid of cards (12rem minmax columns, sm gap) |
| `.app-comment-list`, `.app-comment` | Muted secondary thread under a primary item |
| `.app-visually-hidden` | Standard a11y screen-reader-only helper |

These were extracted after observing the same patterns in faq (and they're identical to what blog and timeline will need). Feature modules use the class names directly; they don't need to re-declare the visual treatment.

`base.css` does **not** apply opinionated typography scale ratios beyond what `--app-space-*` already provides, and does not own component classes for cards or buttons-as-system (those live in feature modules until duplication forces extraction).

## Partial shapes

### `nav`

A top navigation strip. Both variants render:

- A brand mark (text-only by default; consumers replace with logo via the brand slot)
- Zero or more primary nav links
- An optional auth affordance (user menu when authenticated, sign-in link when not)

**Hugo data shape** — driven by site config:

```toml
# in the consumer's hugo config
[params.ui]
  brand_text = "Speculative Fiction"
  brand_url = "/"

[[menu.main]]
  name = "Browse"
  url = "/browse"
  weight = 10
```

The Hugo partial reads `$.Site.Params.ui.brand_text`, `$.Site.Params.ui.brand_url`, `$.Site.Menus.main`, and `$.Site.Params.user` (consumer-provided, optional).

**Go data shape** — driven by an explicit struct:

```go
type NavData struct {
    BrandText string
    BrandURL  string
    Links     []NavLink
    User      *NavUser   // nil = anonymous
    SignInURL string
}

type NavLink struct {
    Label string
    URL   string
}

type NavUser struct {
    DisplayName string
    MenuURL     string  // typically /account or /profile
    SignOutURL  string
}
```

The Go partial reads these fields directly.

Both variants emit the same HTML structure with class names prefixed `app-nav-*` so consumer CSS can override targeted bits without forking the partial.

### `footer`

A small footer strip. Both variants render:

- A small brand mark
- Copyright / credit text
- Optional secondary links (e.g., privacy, contact)

The Hugo and Go data shapes mirror `nav` in spirit, smaller in scope.

## Integration: Hugo consumer quickstart

1. Add `infodancer/ui` to the site's module config:

   ```toml
   # config/_default/module.toml
   [[module.imports]]
     path = "github.com/infodancer/ui"
   ```

2. Pipe the CSS in `head`:

   ```html
   {{ $tokens := resources.Get "css/tokens.css" }}
   {{ $base := resources.Get "css/base.css" }}
   {{ $site := resources.Get "css/site.css" }}  {{/* the consumer's overrides */}}
   {{ $bundle := slice $tokens $base $site | resources.Concat "css/bundle.css" | minify | fingerprint }}
   <link rel="stylesheet" href="{{ $bundle.RelPermalink }}" integrity="{{ $bundle.Data.Integrity }}">
   ```

3. Render the partials:

   ```html
   {{ partial "nav" . }}
   <main>{{ block "main" . }}{{ end }}</main>
   {{ partial "footer" . }}
   ```

4. Provide site palette overrides in `assets/css/site.css`:

   ```css
   :root {
     --app-color-accent: #8c6520;
     --app-font-display: "Cormorant", Georgia, serif;
     --app-font-body: "Courier Prime", "Courier New", monospace;
     /* … */
   }
   ```

## Integration: Go html/template consumer quickstart

1. Add to `go.mod`: `require github.com/infodancer/ui v0.1.0`.

2. Mount the static assets and parse the partials at startup:

   ```go
   import "github.com/infodancer/ui"

   mux.Handle("/static/ui/", http.StripPrefix("/static/ui/", http.FileServer(http.FS(ui.AssetsFS()))))

   tmpl, err := template.ParseFS(ui.PartialsFS(), "*.gohtml")
   // then parse your own templates into the same set
   ```

3. Provide site palette overrides in your own static CSS, loaded after `tokens.css` and `base.css`:

   ```html
   <link rel="stylesheet" href="/static/ui/css/tokens.css">
   <link rel="stylesheet" href="/static/ui/css/base.css">
   <link rel="stylesheet" href="/static/site.css">
   ```

4. Call partials from your base layout:

   ```gohtml
   {{ template "ui/nav" .Nav }}
   <main>{{ block "main" . }}{{ end }}</main>
   {{ template "ui/footer" .Footer }}
   ```

   Where `.Nav` is a `ui.NavData` (or a struct with the same shape — html/template duck-types by field name).

## Versioning

The token vocabulary is the public API. v0.1 establishes the names; subsequent v0.1.x and v0.2.x releases preserve them.

- **Breaking changes** (require major-version bump in the v1+ era; in v0.x they're documented in CHANGELOG and consumers handle the migration):
  - Renaming a token
  - Removing a token
  - Changing a partial's HTML structure in ways that break consumer CSS selectors targeting `.app-nav-*` or `.app-footer-*` classes
  - Changing a partial's documented data shape (struct fields for Go, site-param keys for Hugo)
- **Non-breaking** (minor / patch bumps):
  - Adding a new token
  - Adding a new partial
  - Tweaking a default value of a token (numeric color shifts, spacing nudges)
  - Internal restructuring that doesn't affect tokens, partial HTML, or data shapes

Until v1.0, consumers should pin to a specific tag and treat upgrades deliberately.

## Open questions

- **Hugo module version pinning** — how Hugo modules handle versioning across the consumer set needs a practical check during the first integration (likely sf's faq mount or osg's session-notes work).
- **Whether `nav` and `footer` deserve more configuration knobs** in v0.1 (dropdown menus, mega-nav, footer columns). The current shape is "minimum viable"; revisit after two consumers have integrated.
- **Whether handler-only modules (contact, newsletter) want a tiny `ui`-aware response template** so their success/error fragments look native instead of unstyled. Current lean: no, hosts keep owning. Reopen if a consumer asks.
- **Print styles** — do we ship a `@media print` block in `base.css` v0.1? Probably yes, minimally — print-friendly link rendering and reasonable margins. TBD during implementation.

## What "done" looks like for v0.1

- All files in this proposal exist and parse. **(In progress as of 2026-05-19 — initial commit lands scaffolding + first-pass tokens/CSS/partials.)**
- The token list is reviewed and locked.
- Runnable worked examples land under `examples/go-consumer/` and `examples/hugo-consumer/` once the API has had one real integration. (Deferred from initial commit so the API can shift cheaply during first use.)
- One real consumer has integrated end-to-end (faq M6a-12 is the natural first integration; alternatively a tiny static page within osg or sf serves as the first proof).
- Tagged `v0.1.0` only after that first integration confirms the API and the token vocabulary are stable.
