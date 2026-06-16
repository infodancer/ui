package ui_test

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"github.com/infodancer/ui"
)

func TestPartialsFS_HasLayer2(t *testing.T) {
	t.Parallel()
	tmpl, err := template.ParseFS(ui.PartialsFS(), "*.gohtml")
	if err != nil {
		t.Fatalf("ParseFS: %v", err)
	}
	for _, name := range []string{"ui/document", "ui/analytics"} {
		if tmpl.Lookup(name) == nil {
			t.Errorf("template %q not registered after ParseFS", name)
		}
	}
}

func TestAnalytics_RendersUmamiAndPlausible(t *testing.T) {
	t.Parallel()
	tmpl, err := template.ParseFS(ui.PartialsFS(), "*.gohtml")
	if err != nil {
		t.Fatal(err)
	}
	data := &ui.Analytics{
		Umami:     &ui.Umami{Src: "https://analytics.example.net/script.js", WebsiteID: "abc-123"},
		Plausible: &ui.Plausible{Src: "https://stats.example.net/js/pa-XYZ.js"},
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "ui/analytics", data); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		`<script defer src="https://analytics.example.net/script.js" data-website-id="abc-123"></script>`,
		`<script async src="https://stats.example.net/js/pa-XYZ.js"></script>`,
		`plausible.init()`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("analytics output missing %q\n--- got ---\n%s", want, out)
		}
	}
}

func TestAnalytics_UmamiPerformanceAndReplay(t *testing.T) {
	t.Parallel()
	tmpl, err := template.ParseFS(ui.PartialsFS(), "*.gohtml")
	if err != nil {
		t.Fatal(err)
	}

	// Defaults (Performance/Replay false): no data-performance, no recorder.js.
	var off bytes.Buffer
	if err := tmpl.ExecuteTemplate(&off, "ui/analytics", &ui.Analytics{
		Umami: &ui.Umami{Src: "https://analytics.example.net/script.js", WebsiteID: "abc-123"},
	}); err != nil {
		t.Fatal(err)
	}
	if out := off.String(); strings.Contains(out, "data-performance") || strings.Contains(out, "recorder.js") {
		t.Errorf("defaults should not emit performance/recorder, got:\n%s", out)
	}

	// Both enabled: performance attribute on the tracker tag, plus a recorder.js
	// tag whose URL is derived from Src (same origin/path) with the same id.
	var on bytes.Buffer
	if err := tmpl.ExecuteTemplate(&on, "ui/analytics", &ui.Analytics{
		Umami: &ui.Umami{Src: "https://analytics.example.net/script.js", WebsiteID: "abc-123", Performance: true, Replay: true},
	}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<script defer src="https://analytics.example.net/script.js" data-website-id="abc-123" data-performance="true"></script>`,
		`<script defer src="https://analytics.example.net/recorder.js" data-website-id="abc-123"></script>`,
	} {
		if !strings.Contains(on.String(), want) {
			t.Errorf("analytics output missing %q\n--- got ---\n%s", want, on.String())
		}
	}
}

func TestUmami_RecorderSrc(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ src, want string }{
		{"https://analytics.example.net/script.js", "https://analytics.example.net/recorder.js"},
		{"https://a.example.net/path/custom.js", "https://a.example.net/path/recorder.js"},
		{"script.js", "recorder.js"},
	} {
		if got := (ui.Umami{Src: tc.src}).RecorderSrc(); got != tc.want {
			t.Errorf("RecorderSrc(%q) = %q, want %q", tc.src, got, tc.want)
		}
	}
}

func TestAnalytics_NilMembersOmit(t *testing.T) {
	t.Parallel()
	tmpl, err := template.ParseFS(ui.PartialsFS(), "*.gohtml")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "ui/analytics", &ui.Analytics{}); err != nil {
		t.Fatal(err)
	}
	if out := strings.TrimSpace(buf.String()); out != "" {
		t.Errorf("empty analytics should emit nothing, got:\n%s", out)
	}
}

// parseDocWithPage parses the ui partials first, then the consumer page last,
// so the page's block definitions (content/title/nav/footer) win over
// ui/document's empty defaults — the same parse order OSG's render package uses.
func parseDocWithPage(t *testing.T, page string) *template.Template {
	t.Helper()
	tmpl, err := template.ParseFS(ui.PartialsFS(), "*.gohtml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpl.New("page").Parse(page); err != nil {
		t.Fatal(err)
	}
	return tmpl
}

func TestDocument_RendersFullPage(t *testing.T) {
	t.Parallel()
	tmpl := parseDocWithPage(t,
		`{{ define "title" }}Home · Example{{ end }}`+
			`{{ define "content" }}<p>BODY-MARKER</p>{{ end }}`+
			`{{ template "ui/document" . }}`)

	data := ui.DocumentData{
		Lang:         "en",
		Theme:        "light",
		AssetBase:    "/static/ui",
		AssetVersion: "abc123",
		ExtraCSS:     []string{"/static/osg/css/site.css?v=abc"},
		HeadTags:     ui.HeadTags("/static/ui"),
		Meta:         ui.Meta{SiteName: "Example", Description: "Desc."},
		Analytics:    &ui.Analytics{Umami: &ui.Umami{Src: "https://a.example.net/s.js", WebsiteID: "wid-1"}},
		Nav:          ui.NavData{BrandText: "Example", Links: []ui.NavLink{{Label: "Campaigns", URL: "/campaign/"}}},
		Footer:       ui.FooterData{BrandText: "Example", Copyright: "Chronicles."},
		SidebarRight: &ui.SidebarData{Sections: []ui.SidebarSection{
			{Key: "tools", Title: "Tools", Items: []ui.SidebarItem{{Label: "Namegen", URL: "/tools/namegen/"}}},
		}},
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "page", data); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"<!doctype html>",
		`<html lang="en" data-theme="light">`,
		"<title>Home · Example</title>",
		`<link rel="stylesheet" href="/static/ui/css/open-props.css?v=abc123">`,
		`<link rel="stylesheet" href="/static/ui/css/tokens.css?v=abc123">`,
		`<link rel="stylesheet" href="/static/ui/css/base.css?v=abc123">`,
		`<link rel="stylesheet" href="/static/osg/css/site.css?v=abc">`,
		`integrity="sha384-`, // htmx via HeadTags
		`<meta property="og:site_name" content="Example">`,
		`data-website-id="wid-1"`, // analytics
		`class="app-nav"`,         // default nav from ui/nav
		`href="/campaign/"`,
		`<p>BODY-MARKER</p>`, // content block
		`class="app-sidebar-layout has-right"`,
		`class="app-footer"`, // default footer from ui/footer
	} {
		if !strings.Contains(out, want) {
			t.Errorf("document output missing %q\n--- got ---\n%s", want, out)
		}
	}
}

func TestDocument_HTMXOptOut(t *testing.T) {
	t.Parallel()
	tmpl := parseDocWithPage(t,
		`{{ define "content" }}x{{ end }}{{ template "ui/document" . }}`)
	var buf bytes.Buffer
	// No HeadTags set => no htmx script.
	if err := tmpl.ExecuteTemplate(&buf, "page", ui.DocumentData{AssetBase: "/static/ui"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "htmx-") {
		t.Errorf("document with zero HeadTags should not load htmx\n--- got ---\n%s", buf.String())
	}
}

func TestDocument_NavFooterOverride(t *testing.T) {
	t.Parallel()
	tmpl := parseDocWithPage(t,
		`{{ define "content" }}x{{ end }}`+
			`{{ define "nav" }}<nav id="custom-nav">CUSTOM</nav>{{ end }}`+
			`{{ define "footer" }}<footer id="custom-footer">CF</footer>{{ end }}`+
			`{{ template "ui/document" . }}`)
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "page", ui.DocumentData{AssetBase: "/static/ui"}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, `id="custom-nav"`) || !strings.Contains(out, `id="custom-footer"`) {
		t.Errorf("nav/footer block overrides not honored\n--- got ---\n%s", out)
	}
	if strings.Contains(out, `class="app-nav"`) {
		t.Errorf("overriding the nav block should suppress the default ui/nav\n--- got ---\n%s", out)
	}
}
