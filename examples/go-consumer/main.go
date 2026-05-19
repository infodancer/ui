// Minimal HTTP server demonstrating how to consume infodancer/ui from a Go
// html/template application.
//
// Run with:
//
//	cd examples/go-consumer
//	go run .
//
// Then open http://localhost:8080. The page renders the ui nav + footer
// around a tiny body, with the ui CSS tokens + base stylesheet served
// under /static/ui/.
//
// To experiment with the token system, add a stylesheet that overrides
// --app-* variables and serve it after base.css — the nav, footer, and
// body styles re-resolve through your overrides without any partial
// changes. That's the whole point.
package main

import (
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/infodancer/ui"
)

// basePage is the consumer's own base layout. It declares a head_extra
// block (empty by default; the consumer's own pages can override to
// inject site-specific <link>/<meta>/<script> tags) and pulls in the ui
// nav + footer partials around the main content.
const basePage = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{ .Title }} — infodancer/ui example</title>
<link rel="stylesheet" href="/static/ui/css/tokens.css">
<link rel="stylesheet" href="/static/ui/css/base.css">
{{ block "head_extra" . }}{{ end }}
</head>
<body>
{{ template "ui/nav" .Nav }}
<main class="app-container">
<article class="app-prose">
<h1>{{ .Title }}</h1>
{{ .Body }}
</article>
</main>
{{ template "ui/footer" .Footer }}
</body>
</html>
`

type viewData struct {
	Title  string
	Body   template.HTML
	Nav    ui.NavData
	Footer ui.FooterData
}

func main() {
	tmpl, err := template.New("base").Parse(basePage)
	if err != nil {
		log.Fatalf("parse base: %v", err)
	}
	if _, err := tmpl.ParseFS(ui.PartialsFS(), "*.gohtml"); err != nil {
		log.Fatalf("parse ui partials: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/static/ui/", http.StripPrefix("/static/ui/", http.FileServer(http.FS(ui.AssetsFS()))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data := viewData{
			Title: "Hello",
			Body: template.HTML(`<p>This page is rendered by a minimal Go HTTP server consuming the <code>infodancer/ui</code> module.</p>
<p>The token vocabulary controls the look. Override <code>--app-*</code> values in a stylesheet loaded after <code>base.css</code> to retheme the page without touching any partials.</p>
<p>Try resizing your browser — the layout is built from token primitives, not breakpoint media queries.</p>`),
			Nav: ui.NavData{
				BrandText: "Example",
				BrandURL:  "/",
				Links: []ui.NavLink{
					{Label: "Browse", URL: "/browse"},
					{Label: "About", URL: "/about"},
				},
			},
			Footer: ui.FooterData{
				BrandText: "Example",
				Copyright: "© 2026 Example.org",
				Links: []ui.FooterLink{
					{Label: "Privacy", URL: "/privacy"},
					{Label: "Contact", URL: "/contact"},
				},
			},
		}
		if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
			log.Printf("execute: %v", err)
		}
	})

	addr := ":" + getenv("PORT", "8080")
	log.Printf("listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
