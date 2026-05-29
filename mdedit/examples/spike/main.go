// Command spike is a runnable demonstration of the mdedit component and its
// adapter seam, wired with EasyMDE. It is intentionally tiny: a couple of
// in-memory documents, no auth, no database — just enough to click through the
// display → edit → (preview) → save → display loop in a browser and confirm
// the seam works end to end.
//
//	cd examples/spike && go run .   # then open http://localhost:8099
//
// It shows two regions that share one component but differ by configuration:
//
//   - a long-form Document: markdown.Rich() + the "full" toolbar (the
//     session-notes case).
//   - a Comment: markdown.Comment(LinkRelativeOnly) + the "standard" toolbar —
//     a constrained subset (no headings/images/tables) where outbound links
//     are stripped but relative ones survive.
//
// Both prove: the textarea is the source of truth (try it with JS off),
// EasyMDE enhances it with a no-CDN text toolbar and no preview pane, and the
// server's markdown package is the sole renderer (preview == what Save stores).
package main

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"sync"

	uiassets "github.com/infodancer/ui"

	"github.com/infodancer/ui/markdown"
	"github.com/infodancer/ui/mdedit"
)

// htmx is vendored (not pulled from a CDN) so the demo runs fully offline and
// self-contained. htmx 2.0.4, BSD-Zero-Clause; see https://htmx.org.
//
//go:embed static/htmx.min.js
var staticFS embed.FS

const staticBase = "/static/mdedit"

// doc is one in-memory document behind a mutex.
type doc struct {
	mu sync.RWMutex
	md string
}

func (d *doc) get() string  { d.mu.RLock(); defer d.mu.RUnlock(); return d.md }
func (d *doc) set(s string) { d.mu.Lock(); defer d.mu.Unlock(); d.md = s }

// region is one editable area: a stored doc, the renderer that governs what
// its Markdown is allowed to become, and the Field configuration (toolbar,
// sizing) for the editor. Two regions exercise two profiles from one loop.
type region struct {
	tpl      *template.Template
	renderer *markdown.Renderer
	doc      *doc
	base     string // URL prefix, e.g. "/doc"
	id       string
	label    string
	toolbar  string
	rows     int
	maxLen   int
}

func (rg *region) field() mdedit.Field {
	return mdedit.Field{
		ID:          rg.id,
		Markdown:    rg.doc.get(),
		HTML:        rg.renderer.Render(rg.doc.get()),
		DisplayURL:  rg.base,
		EditURL:     rg.base + "/edit",
		SaveURL:     rg.base + "/save",
		PreviewURL:  rg.base + "/preview",
		LivePreview: true,
		Label:       rg.label,
		Toolbar:     rg.toolbar,
		Rows:        rg.rows,
		MaxLength:   rg.maxLen,
	}.WithDefaults()
}

func (rg *region) display(w http.ResponseWriter, _ *http.Request) {
	rg.exec(w, "mdedit/display", rg.field())
}
func (rg *region) edit(w http.ResponseWriter, _ *http.Request) { rg.exec(w, "mdedit/edit", rg.field()) }

func (rg *region) save(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	rg.doc.set(r.PostFormValue("markdown"))
	rg.exec(w, "mdedit/display", rg.field())
}

func (rg *region) preview(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	// Render the submitted Markdown (not the stored copy) through this
	// region's own renderer, so preview reflects unsaved edits AND the same
	// constraints Save will apply.
	f := rg.field()
	f.HTML = rg.renderer.Render(r.PostFormValue("markdown"))
	rg.exec(w, "mdedit/preview", f)
}

func (rg *region) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET "+rg.base, rg.display)
	mux.HandleFunc("GET "+rg.base+"/edit", rg.edit)
	mux.HandleFunc("POST "+rg.base+"/save", rg.save)
	mux.HandleFunc("POST "+rg.base+"/preview", rg.preview)
}

func (rg *region) exec(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := rg.tpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("execute %s: %v", name, err)
	}
}

func main() {
	tpl := template.Must(template.New("page").Parse(pageHTML))
	template.Must(tpl.ParseFS(mdedit.PartialsFS(), "*.gohtml"))

	document := &region{
		tpl: tpl, renderer: markdown.New(markdown.Rich()), doc: &doc{md: docSeed},
		base: "/doc", id: "doc", label: "Document body (Markdown)",
		toolbar: "full", rows: 14, maxLen: 30000,
	}
	comment := &region{
		tpl: tpl, renderer: markdown.New(markdown.Comment(markdown.LinkRelativeOnly)), doc: &doc{md: commentSeed},
		base: "/comment", id: "comment", label: "Your comment",
		toolbar: "standard", rows: 5, maxLen: 2000,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tpl.ExecuteTemplate(w, "page", map[string]mdedit.Field{
			"Doc":     document.field(),
			"Comment": comment.field(),
		}); err != nil {
			log.Printf("page: %v", err)
		}
	})
	document.routes(mux)
	comment.routes(mux)
	mux.Handle(staticBase+"/", http.StripPrefix(staticBase+"/", http.FileServer(http.FS(mdedit.AssetsFS()))))
	mux.Handle("/static/ui/", http.StripPrefix("/static/ui/", http.FileServer(http.FS(uiassets.AssetsFS()))))
	spikeStatic, _ := fs.Sub(staticFS, "static")
	mux.Handle("/static/spike/", http.StripPrefix("/static/spike/", http.FileServer(http.FS(spikeStatic))))

	addr := "localhost:8099"
	log.Printf("mdedit spike on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

const docSeed = "# mdedit spike\n\nClick **Edit**. The toolbar is EasyMDE; the *preview* and the saved\noutput both come from the server's markdown package (goldmark + bluemonday).\n\nTry pasting `<script>alert(1)</script>` and previewing — it is stripped.\n\n| feature | state |\n|---------|-------|\n| GFM tables | on |\n"

const commentSeed = "A **comment** with a [relative link](/doc) — try editing.\n\nAdd `## a heading`, an `![image](/x.png)`, a table, or `[outbound](https://example.com)`: the constrained preset drops them all, keeping the relative link."

// pageHTML is the host page. HeadTags injects the component's CSS/JS; htmx is
// the host's responsibility (here, vendored and served at /static/spike). The page
// data is a map[string]mdedit.Field with keys Doc and Comment.
var pageHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>mdedit spike</title>
<link rel="stylesheet" href="/static/ui/css/tokens.css">
` + string(mdedit.HeadTags(staticBase)) + `
<script defer src="/static/spike/htmx.min.js"></script>
<style>
  body { font-family: system-ui, sans-serif; max-width: 48rem; margin: 2rem auto; padding: 0 1rem; }
  .app-prose :first-child { margin-top: 0; }
  section { margin-block: 2.5rem; }
</style>
</head>
<body>
<h1>mdedit spike</h1>
<section>
  <h2>Long-form document — markdown.Rich(), full toolbar</h2>
  {{template "mdedit/display" .Doc}}
</section>
<section>
  <h2>Comment — markdown.Comment(relative-only), standard toolbar</h2>
  {{template "mdedit/display" .Comment}}
</section>
</body>
</html>
`
