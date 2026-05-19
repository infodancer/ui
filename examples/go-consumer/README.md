# go-consumer

Minimal runnable example of consuming `infodancer/ui` from a Go `html/template` HTTP server. Single file `main.go`, own `go.mod` with a `replace` directive pointing at the parent module.

## Run

```bash
go run .
```

Opens on `:8080` by default (or `$PORT`). Visit `http://localhost:8080` to see the rendered page.

## What it shows

- Parsing the `ui/nav` and `ui/footer` partials into a consumer-side `*template.Template`.
- Mounting `ui.AssetsFS()` under `/static/ui/` via `http.FileServer`.
- Linking `tokens.css` and `base.css` from the consumer's base layout.
- Populating `ui.NavData` and `ui.FooterData` with realistic structured data.
- Reserving a `head_extra` block in the consumer's base layout for per-page injection (unused by this example, demonstrated by being declared).

## What it doesn't show

- Token overrides. Try adding `<link rel="stylesheet" href="/static/site.css">` after `base.css` in `basePage` and dropping a `site.css` with `:root { --app-color-accent: red; }` into the static handler to see palette inheritance work.
- Partial overrides. Pass a different template by name to swap out `ui/nav` — same Go `template.Template` API.
- Authentication. `NavData.User` is nil here; set it to a `&ui.NavUser{DisplayName: "alice"}` to see the authenticated nav variant.
