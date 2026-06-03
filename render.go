package ui

import (
	"bytes"
	_ "embed"
	"html/template"
	"net/http"
	"sync"
)

// docPageSrc holds the title/content blocks RenderDocument fills into
// ui/document, plus the errorbody fragment RenderError composes. Kept out of
// the partials FS so a consumer's ParseFS(PartialsFS(), …) never inherits these
// definitions.
//
//go:embed docpage.gohtml
var docPageSrc string

// pageTmpl is the private template set shared by RenderDocument and RenderError:
// the embedded ui partials (ui/document and the Layer 1 pieces it calls) plus
// the title/content/errorbody definitions. Built once, reused.
var pageTmpl = sync.OnceValue(func() *template.Template {
	t := template.Must(template.New("ui-page").ParseFS(partialsRoot, "partials/*.gohtml"))
	return template.Must(t.Parse(docPageSrc))
})

// DocumentPage backs [RenderDocument]: the host page chrome plus a body. Embed
// it (or DocumentData) and populate the chrome from the consumer's Chrome hook /
// DocumentChrome so the page wears the site's normal nav, footer, theme, and CSS.
type DocumentPage struct {
	DocumentData

	// Status is the HTTP status written to the response. Zero defaults to 200.
	Status int
	// Title is the <title> text; empty falls back to Meta.SiteName.
	Title string
	// Body is the content-column markup. It is template.HTML — caller-controlled
	// and trusted — so the consumer owns the escaping of anything it
	// interpolates. For a plain title+message error page, use [RenderError],
	// whose Message is an escaped string.
	Body template.HTML
}

// RenderDocument renders Body wrapped in the document chrome carried by
// d.DocumentData and writes it with d.Status (0 → 200). It is the leaf page
// renderer: ui supplies chrome + the self-contained body, never a consumer's
// page templates, FuncMaps, or shell. The page is rendered into a buffer first,
// so a template failure returns an error with nothing written.
func RenderDocument(w http.ResponseWriter, d DocumentPage) error {
	status := d.Status
	if status == 0 {
		status = http.StatusOK
	}
	return renderPage(w, status, d)
}

// ErrorData backs [RenderError]: the host page chrome plus the error specifics.
type ErrorData struct {
	DocumentData

	// Status is the HTTP status code written to the response.
	Status int
	// Title is the <h1> and (with Meta.SiteName) the <title>.
	Title string
	// Message is the body paragraph. It is a plain string and is escaped, so
	// it is safe to interpolate request-derived text (a path, a slug); for a
	// body that needs markup, use [RenderDocument].
	Message string
}

// RenderError renders a standard error page — Title and Message wrapped in the
// document chrome on d.DocumentData — and writes it with d.Status. It is the
// error-specific preset over [RenderDocument]: it composes the (escaped) error
// body and delegates, so the document-wrapping lives in one place.
//
// It is the shared replacement across the infodancer stack for a bare
// ui/document or a text/plain http.Error, so every error page matches the
// site's normal chrome.
func RenderError(w http.ResponseWriter, d ErrorData) error {
	var body bytes.Buffer
	if err := pageTmpl().ExecuteTemplate(&body, "errorbody", d); err != nil {
		return err
	}
	return RenderDocument(w, DocumentPage{
		DocumentData: d.DocumentData,
		Status:       d.Status,
		Title:        d.Title,
		// errorbody escaped Title/Message already; this is its trusted output.
		Body: template.HTML(body.String()), //nolint:gosec // sanitized by the errorbody template
	})
}

// renderPage executes ui/document for d into a buffer, then writes it at status.
func renderPage(w http.ResponseWriter, status int, d any) error {
	var buf bytes.Buffer
	if err := pageTmpl().ExecuteTemplate(&buf, "ui/document", d); err != nil {
		return err
	}
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	}
	w.WriteHeader(status)
	_, err := w.Write(buf.Bytes())
	return err
}
