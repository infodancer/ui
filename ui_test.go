package ui_test

import (
	"bytes"
	"html/template"
	"io/fs"
	"strings"
	"testing"

	"github.com/infodancer/ui"
)

func TestAssetsFS_HasCSSFiles(t *testing.T) {
	t.Parallel()
	fsys := ui.AssetsFS()
	for _, path := range []string{"css/tokens.css", "css/base.css"} {
		b, err := fs.ReadFile(fsys, path)
		if err != nil {
			t.Errorf("AssetsFS missing %s: %v", path, err)
			continue
		}
		if len(b) == 0 {
			t.Errorf("AssetsFS file %s is empty", path)
		}
	}
}

// TestTokensCSS_HasAllDocumentedTokens guards against silent drift between
// DESIGN.md's documented token vocabulary and the actual tokens.css.
// If you add or rename a token, update both files and this list.
func TestTokensCSS_HasAllDocumentedTokens(t *testing.T) {
	t.Parallel()
	b, err := fs.ReadFile(ui.AssetsFS(), "css/tokens.css")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)

	tokens := []string{
		// Colors
		"--app-color-bg", "--app-color-fg", "--app-color-bg-raised",
		"--app-color-fg-muted", "--app-color-border", "--app-color-accent",
		"--app-color-accent-hover", "--app-color-accent-on",
		"--app-color-prose-fg", "--app-color-danger", "--app-color-success",
		// Typography
		"--app-font-body", "--app-font-display", "--app-font-mono",
		"--app-font-size-base", "--app-line-height-body", "--app-line-height-display",
		// Spacing
		"--app-space-xs", "--app-space-sm", "--app-space",
		"--app-space-lg", "--app-space-xl",
		// Radii
		"--app-radius-sm", "--app-radius", "--app-radius-pill",
		// Layout
		"--app-max-width-prose", "--app-max-width-page",
	}
	for _, tok := range tokens {
		// Match the property declaration ("--app-foo:") to avoid false
		// positives from substring matches (e.g. --app-space matching
		// --app-space-xs).
		if !strings.Contains(src, tok+":") {
			t.Errorf("tokens.css missing %q declaration", tok)
		}
	}
}

// TestBaseCSS_HasAllUtilityClasses guards against silent drift between
// DESIGN.md's documented utility-class roster and base.css. If you add
// or rename a utility class, update both.
func TestBaseCSS_HasAllUtilityClasses(t *testing.T) {
	t.Parallel()
	b, err := fs.ReadFile(ui.AssetsFS(), "css/base.css")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)

	classes := []string{
		// Layout + prose helpers
		".app-container", ".app-prose",
		// Status text colors
		".app-danger", ".app-success",
		// Chrome
		".app-nav", ".app-nav-brand", ".app-nav-links",
		".app-footer", ".app-footer-brand", ".app-footer-links",
		// List chrome
		".app-list-header", ".app-list-sorts", ".app-list-empty",
		// Sort link
		".app-sort",
		// Tag chips
		".app-tag-list", ".app-tag", ".app-tag-count",
		// Search
		".app-search-form", ".app-search-empty",
		// Pager
		".app-pager", ".app-pager-pos",
		// Badge
		".app-badge",
		// Card primitives
		".app-card", ".app-card-grid",
		// Comments
		".app-comment-list", ".app-comment",
		// Accessibility
		".app-visually-hidden",
	}
	for _, cls := range classes {
		if !strings.Contains(src, cls) {
			t.Errorf("base.css missing %q declaration", cls)
		}
	}
}

func TestPartialsFS_HasNavAndFooter(t *testing.T) {
	t.Parallel()
	fsys := ui.PartialsFS()
	for _, path := range []string{"nav.gohtml", "footer.gohtml"} {
		b, err := fs.ReadFile(fsys, path)
		if err != nil {
			t.Errorf("PartialsFS missing %s: %v", path, err)
			continue
		}
		if len(b) == 0 {
			t.Errorf("PartialsFS file %s is empty", path)
		}
	}
}

func TestPartialsFS_ParsesAsTemplates(t *testing.T) {
	t.Parallel()
	tmpl, err := template.ParseFS(ui.PartialsFS(), "*.gohtml")
	if err != nil {
		t.Fatalf("ParseFS: %v", err)
	}
	for _, name := range []string{"ui/nav", "ui/footer"} {
		if got := tmpl.Lookup(name); got == nil {
			t.Errorf("template %q not registered after ParseFS", name)
		}
	}
}

func TestNav_RendersAuthenticated(t *testing.T) {
	t.Parallel()
	tmpl, err := template.ParseFS(ui.PartialsFS(), "*.gohtml")
	if err != nil {
		t.Fatal(err)
	}
	data := ui.NavData{
		BrandText: "Example Site",
		BrandURL:  "/",
		Links: []ui.NavLink{
			{Label: "Browse", URL: "/browse"},
			{Label: "About", URL: "/about"},
		},
		User: &ui.NavUser{DisplayName: "alice"},
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "ui/nav", data); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		`class="app-nav"`,
		`class="app-nav-brand"`,
		"Example Site",
		`href="/browse"`,
		"Browse",
		"alice",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("nav output missing %q\noutput:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Sign in") {
		t.Errorf("authenticated nav should not show the sign-in link\noutput:\n%s", out)
	}
}

func TestNav_RendersAnonymous(t *testing.T) {
	t.Parallel()
	tmpl, err := template.ParseFS(ui.PartialsFS(), "*.gohtml")
	if err != nil {
		t.Fatal(err)
	}
	data := ui.NavData{BrandText: "Example Site"} // User == nil, BrandURL empty
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "ui/nav", data); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Sign in") {
		t.Errorf("anonymous nav should show Sign in\noutput:\n%s", out)
	}
	if !strings.Contains(out, `href="/login"`) {
		t.Errorf("anonymous nav should default SignInURL to /login\noutput:\n%s", out)
	}
	if !strings.Contains(out, `href="/"`) {
		t.Errorf("nav should default BrandURL to /\noutput:\n%s", out)
	}
}

func TestFooter_RendersWithLinks(t *testing.T) {
	t.Parallel()
	tmpl, err := template.ParseFS(ui.PartialsFS(), "*.gohtml")
	if err != nil {
		t.Fatal(err)
	}
	data := ui.FooterData{
		BrandText: "Example",
		Copyright: "© 2026 Example.org",
		Links: []ui.FooterLink{
			{Label: "Privacy", URL: "/privacy"},
			{Label: "Contact", URL: "/contact"},
		},
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "ui/footer", data); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		`class="app-footer"`,
		"© 2026 Example.org",
		`href="/privacy"`,
		"Contact",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("footer output missing %q\noutput:\n%s", want, out)
		}
	}
}

// TestRealisticConsumerIntegration exercises the consumption pattern that
// README and DESIGN.md document. If this breaks, the documented integration
// path is wrong and one or both needs to be updated.
func TestRealisticConsumerIntegration(t *testing.T) {
	t.Parallel()
	const baseHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>{{ .Title }}</title>
<link rel="stylesheet" href="/static/ui/css/tokens.css">
<link rel="stylesheet" href="/static/ui/css/base.css">
{{ block "head_extra" . }}{{ end }}
</head>
<body>
{{ template "ui/nav" .Nav }}
<main class="app-container">{{ .Body }}</main>
{{ template "ui/footer" .Footer }}
</body>
</html>
`
	tmpl, err := template.New("base").Parse(baseHTML)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpl.ParseFS(ui.PartialsFS(), "*.gohtml"); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "base", struct {
		Title  string
		Body   template.HTML
		Nav    ui.NavData
		Footer ui.FooterData
	}{
		Title: "Hello",
		Body:  template.HTML("<p>Hello world.</p>"),
		Nav: ui.NavData{
			BrandText: "Example",
			BrandURL:  "/",
		},
		Footer: ui.FooterData{
			BrandText: "Example",
			Copyright: "© 2026 Example",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"<title>Hello</title>",
		`class="app-nav"`,
		"<p>Hello world.</p>",
		`class="app-footer"`,
		`href="/static/ui/css/tokens.css"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("integration output missing %q", want)
		}
	}
}
