package ui

import (
	"bytes"
	_ "embed"
	"html/template"
	"net/http"
	"sync"
)

// errorPageSrc holds the content/title blocks RenderError fills into
// ui/document. Kept out of the partials FS so a consumer's
// ParseFS(PartialsFS(), …) never inherits these block definitions.
//
//go:embed errorpage.gohtml
var errorPageSrc string

// ErrorData backs [RenderError]: the host page chrome plus the error specifics.
// Embed it (or DocumentData) the way page handlers already embed DocumentData,
// and populate the chrome from the consumer's Chrome hook / DocumentChrome so
// the error wears the same nav, footer, theme, and CSS as a normal page.
type ErrorData struct {
	DocumentData

	// Status is the HTTP status code written to the response.
	Status int
	// Title is the <h1> and (with Meta.SiteName) the <title>.
	Title string
	// Message is the body paragraph. It is a plain string and is escaped, so
	// it is safe to interpolate request-derived text (a path, a slug); for a
	// body that needs markup, that is the job of the planned RenderDocument.
	Message string
}

// errorTmpl is RenderError's private template set: the embedded ui partials
// (ui/document and the Layer 1 pieces it calls) plus the error content/title
// blocks. Built once, reused — the set is immutable after construction.
var errorTmpl = sync.OnceValue(func() *template.Template {
	t := template.Must(template.New("ui-error").ParseFS(partialsRoot, "partials/*.gohtml"))
	return template.Must(t.Parse(errorPageSrc))
})

// RenderError renders a standard error page — the title and message wrapped in
// the document chrome carried by d.DocumentData — and writes it with d.Status.
//
// It is the leaf error renderer shared across the infodancer stack: a host or a
// mounted module builds its chrome (via its Chrome hook / DocumentChrome) and
// calls this instead of emitting a bare ui/document or a text/plain http.Error,
// so every error page matches the site's normal chrome. The page is rendered
// into a buffer first, so a template failure returns an error with nothing
// written — the caller can fall back to http.Error.
func RenderError(w http.ResponseWriter, d ErrorData) error {
	var buf bytes.Buffer
	if err := errorTmpl().ExecuteTemplate(&buf, "ui/document", d); err != nil {
		return err
	}
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	}
	w.WriteHeader(d.Status)
	_, err := w.Write(buf.Bytes())
	return err
}
