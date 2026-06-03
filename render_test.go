package ui_test

import (
	"html/template"
	"net/http/httptest"
	"strings"
	"testing"

	ui "github.com/infodancer/ui"
)

// RenderError must wrap the error body in the full document chrome carried by
// the embedded DocumentData — nav, footer, ui asset bundle, and the consumer's
// ExtraCSS — not a bare ui/document shell.
func TestRenderError_WrapsBodyInChrome(t *testing.T) {
	d := ui.ErrorData{
		DocumentData: ui.DocumentData{
			Lang: "en", Theme: "light",
			AssetBase: "/static/ui", AssetVersion: "v1",
			ExtraCSS: []string{"/site.css"},
			Meta:     ui.Meta{SiteName: "Example"},
			Nav: ui.NavData{BrandText: "Example", Items: []ui.MenuItem{
				{Key: "campaigns", Label: "Campaigns", URL: "/campaign/"},
			}},
			Footer: ui.FooterData{BrandText: "Example", Links: []ui.FooterLink{
				{Label: "Contribute", URL: "/contribute/"},
			}},
		},
		Status:  404,
		Title:   "Not found",
		Message: "Nothing exists at this address.",
	}

	rec := httptest.NewRecorder()
	if err := ui.RenderError(rec, d); err != nil {
		t.Fatalf("RenderError: %v", err)
	}

	if rec.Code != 404 {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"<!doctype html>",                     // a complete document
		"<h1>Not found</h1>",                  // the error title
		"Nothing exists at this address.",     // the message
		"<title>Not found · Example</title>",  // title block
		`href="/campaign/"`,                   // nav item — chrome
		`href="/contribute/"`,                 // footer — chrome
		"/site.css",                           // consumer ExtraCSS — chrome
		`href="/static/ui/css/base.css?v=v1"`, // ui asset bundle — chrome
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q\n---\n%s", want, body)
		}
	}
}

// RenderDocument wraps an arbitrary (trusted) body in the document chrome.
func TestRenderDocument_WrapsBodyInChrome(t *testing.T) {
	rec := httptest.NewRecorder()
	err := ui.RenderDocument(rec, ui.DocumentPage{
		DocumentData: ui.DocumentData{
			AssetBase: "/static/ui",
			Meta:      ui.Meta{SiteName: "Example"},
			Nav:       ui.NavData{Items: []ui.MenuItem{{Key: "c", Label: "Campaigns", URL: "/campaign/"}}},
			Footer:    ui.FooterData{Links: []ui.FooterLink{{Label: "Contribute", URL: "/contribute/"}}},
		},
		Title: "Verify your email",
		Body:  template.HTML(`<p>Please <a href="/account/">verify</a> first.</p>`),
	})
	if err != nil {
		t.Fatalf("RenderDocument: %v", err)
	}
	if rec.Code != 200 { // Status 0 defaults to 200
		t.Errorf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"<!doctype html>",
		"<title>Verify your email · Example</title>",
		`<p>Please <a href="/account/">verify</a> first.</p>`, // body passes through unescaped
		`href="/campaign/"`,   // nav chrome
		`href="/contribute/"`, // footer chrome
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q\n---\n%s", want, body)
		}
	}
}

// An explicit status is honored.
func TestRenderDocument_HonorsStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	_ = ui.RenderDocument(rec, ui.DocumentPage{
		DocumentData: ui.DocumentData{Meta: ui.Meta{SiteName: "S"}},
		Status:       403,
		Title:        "Forbidden",
		Body:         template.HTML("<p>no</p>"),
	})
	if rec.Code != 403 {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

// The message and title are escaped — they are plain strings, not template.HTML.
func TestRenderError_EscapesUserText(t *testing.T) {
	rec := httptest.NewRecorder()
	err := ui.RenderError(rec, ui.ErrorData{
		DocumentData: ui.DocumentData{Meta: ui.Meta{SiteName: "S"}},
		Status:       404,
		Title:        "t",
		Message:      `<script>alert(1)</script>`,
	})
	if err != nil {
		t.Fatalf("RenderError: %v", err)
	}
	if strings.Contains(rec.Body.String(), "<script>alert") {
		t.Errorf("message not escaped:\n%s", rec.Body.String())
	}
}
