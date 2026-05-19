# infodancer/ui

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

Shared design tokens, base CSS, and `nav` / `footer` partials for the [infodancer](https://github.com/infodancer) / [matthewjhunter](https://github.com/matthewjhunter) web portfolio. Hugo and Go `html/template` are equal first-class consumers; the same tokens style both worlds with no fork.

> Maintained as a personal utility for sites and modules in the portfolio. Issues and PRs welcome but response times vary. See [SECURITY.md](SECURITY.md) for vulnerability reporting.

## What it is

A small CSS + template library:

- **Design tokens** — CSS custom properties under the `--app-*` prefix covering color, typography, spacing, radii, and layout primitives. Tokens are the public API; everything else is implementation detail.
- **Base stylesheet** — a minimal reset plus sensible defaults for typography, lists, links, code, basic form elements. ~150 lines.
- **`nav` and `footer` partials** — top nav strip and footer, shipped in **parallel Hugo and Go variants** producing identical output but each idiomatic to its host engine.

What it deliberately is *not*: a component framework, a JS toolkit, an icon set, or a build pipeline. See [DESIGN.md](DESIGN.md) for scope rationale.

## Design

[DESIGN.md](DESIGN.md) is the source of truth for the token vocabulary, the two-consumer model, the partial data shapes, and the versioning policy.

## Quickstart — Hugo consumer

Add `infodancer/ui` to your site's module imports:

```toml
# config/_default/module.toml
[[module.imports]]
  path = "github.com/infodancer/ui"
```

Pipe the CSS and render the partials from your base layout:

```html
{{ $tokens := resources.Get "css/tokens.css" }}
{{ $base := resources.Get "css/base.css" }}
{{ $site := resources.Get "css/site.css" }}
{{ $bundle := slice $tokens $base $site | resources.Concat "css/bundle.css" | minify | fingerprint }}
<link rel="stylesheet" href="{{ $bundle.RelPermalink }}" integrity="{{ $bundle.Data.Integrity }}">

{{ partial "nav" . }}
<main>{{ block "main" . }}{{ end }}</main>
{{ partial "footer" . }}
```

Provide your site's palette in `assets/css/site.css`:

```css
:root {
  --app-color-accent: #8c6520;
  --app-font-display: "Cormorant", Georgia, serif;
}
```

## Quickstart — Go html/template consumer

```bash
go get github.com/infodancer/ui
```

Mount the assets and parse the partials at startup:

```go
import "github.com/infodancer/ui"

mux.Handle("/static/ui/", http.StripPrefix("/static/ui/", http.FileServer(http.FS(ui.AssetsFS()))))

tmpl, err := template.ParseFS(ui.PartialsFS(), "*.gohtml")
```

Pass a `ui.NavData` (or any struct with matching fields) to the partial:

```gohtml
{{ template "ui/nav" .Nav }}
<main>{{ block "main" . }}{{ end }}</main>
{{ template "ui/footer" .Footer }}
```

## Status

v0.1 initial design (2026-05-19). Token vocabulary proposed but not yet locked; first real integration drives the lock. Not yet tagged.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for local dev commands and conventions.

## License

Apache-2.0. See [LICENSE](LICENSE).
