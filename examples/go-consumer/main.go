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
	"log/slog"
	"net/http"
	"os"

	"github.com/infodancer/logging"
	"github.com/infodancer/logging/httplog"
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
	// Logging is the house standard rather than the standard library's log
	// package: logfmt with lowercased levels, and one access log line per
	// request from the shared middleware. A consumer copying this example
	// gets a server that is legible to a log pipeline from the first run.
	logger := logging.NewLogger(getenv("LOG_LEVEL", "info"))
	slog.SetDefault(logger)

	tmpl, err := template.New("base").Parse(basePage)
	if err != nil {
		logger.Error("parsing the base template failed", "err", err)
		os.Exit(1)
	}
	if _, err := tmpl.ParseFS(ui.PartialsFS(), "*.gohtml"); err != nil {
		logger.Error("parsing the ui partials failed", "err", err)
		os.Exit(1)
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
			// Headers are already out by the time a template fails midway, so
			// there is nothing to tell the client: the log is the only report.
			logger.Error("executing the page template failed", "err", err)
		}
	})

	addr := ":" + getenv("PORT", "8080")
	srv := &http.Server{
		Addr:     addr,
		Handler:  httplog.Middleware(logger)(mux),
		ErrorLog: httplog.ErrorLog(logger),
	}
	logger.Info("listening", "url", "http://localhost"+addr)
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
