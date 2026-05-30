# mdedit

A reusable Markdown display/edit component for the infodancer / matthewjhunter
Go + htmx stack. It renders stored Markdown as sanitized HTML, and — behind an
optional Edit button — swaps in a client editor, posts the result to your
backend, and returns to display. Rendering and sanitizing is delegated to
[`markdown`](../markdown/README.md), the one audited goldmark + bluemonday
pipeline shared across the portfolio.

It is **not** a framework. It owns the UI loop and the client-editor seam; your
application owns storage, authentication, and authorization, and calls
`markdown` to render.

A nested module in the [infodancer/ui](../README.md) monorepo:

```go
import "github.com/infodancer/ui/mdedit"
import "github.com/infodancer/ui/markdown" // the render/sanitize pipeline
```

## What ships

| Piece | What it is |
|-------|-----------|
| `AssetsFS()` | Client assets: a vendored, SRI-pinned editor (EasyMDE), the adapter seam, the loader, and token-styled CSS. Mount under a static path. |
| `PartialsFS()` | Go `html/template` partials: `mdedit/display`, `mdedit/edit`, `mdedit/preview`. |
| `Field` | The data shape those partials render — current Markdown, rendered HTML, and the four loop URLs. |
| `HeadTags(base)` | The `<link>`/`<script>` tags (with SRI) for your page `<head>`. |

## The loop

`display` shows rendered HTML and an Edit button. Edit `hx-get`s the `edit`
partial and swaps it in. Save `hx-post`s the Markdown; your handler renders +
sanitizes + persists and returns the `display` partial. Cancel returns display
unchanged. Preview, when enabled, `hx-post`s to a server endpoint that renders
through the **same** pipeline as display — so the preview matches what Save
will store, not a client approximation.

The `<textarea>` is the source of truth and works with **no JavaScript**. The
editor is progressive enhancement on top.

## The adapter seam

The client editor sits behind a one-function contract (`assets/mdedit.js`), so
swapping editors is a data change (`Field.Adapter`), not a template or server
change. EasyMDE is the only adapter registered today; CodeMirror 6 and Toast UI
are candidate adapters. See `assets/mdedit.js` for the contract and
`assets/adapters/easymde.js` for the reference implementation.

EasyMDE is configured deliberately: its bundled client preview is **disabled**
(the server is the only renderer), and it never fetches an icon font from a CDN
(`autoDownloadFontAwesome: false`). The vendored files are pinned and
SRI-hashed — see `assets/vendor/PROVENANCE.md`.

## Loading a local file

Set `Field.AllowFileLoad` to add a "Load file…" control that loads a local
`.md` file into the editor, as if the author had typed it. The file is read in
the browser and dropped into the textarea — it is **never uploaded**; it rides
the normal Save POST and goes through the same `markdown` sanitization as typed
content, so it adds no server attack surface. Off by default; enable it for
long-form pages, not comments. Image upload is a separate, deferred concern
(see [DESIGN.md](DESIGN.md) and
[infodancer/ui#14](https://github.com/infodancer/ui/issues/14)).

## Try it

The example spike is a separate module; run it with the workspace disabled so
it resolves its own `replace` directives:

```
cd examples/spike && GOWORK=off go run .
```

Open <http://localhost:8099>. Edit the document, preview it (paste a
`<script>` tag and watch it get stripped), save, and try "Load file…" on the
long-form region.

## Status

Pre-1.0 (tag `mdedit/v0.1.x`). The `Field` shape, partial template names, and
the `assets/mdedit.js` adapter contract are the public surface; treat them with
versioning discipline. The render/sanitize policy is versioned separately in
[`markdown`](../markdown/README.md). See [DESIGN.md](DESIGN.md).

## License

Apache-2.0. See [the repository LICENSE](../LICENSE).
