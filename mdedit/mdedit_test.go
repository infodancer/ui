package mdedit

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
)

// parseAll mirrors how a host wires the partials: ParseFS alongside its own
// templates. If the partials don't parse or define the documented names,
// every consumer breaks, so prove it here.
func parseAll(t *testing.T) *template.Template {
	t.Helper()
	tpl, err := template.ParseFS(PartialsFS(), "*.gohtml")
	if err != nil {
		t.Fatalf("ParseFS: %v", err)
	}
	for _, name := range []string{"mdedit/display", "mdedit/edit", "mdedit/preview"} {
		if tpl.Lookup(name) == nil {
			t.Fatalf("partial %q not defined", name)
		}
	}
	return tpl
}

func render(t *testing.T, tpl *template.Template, name string, data any) string {
	t.Helper()
	var buf bytes.Buffer
	if err := tpl.ExecuteTemplate(&buf, name, data); err != nil {
		t.Fatalf("execute %s: %v", name, err)
	}
	return buf.String()
}

func TestDisplay_RendersHTMLAndEditButton(t *testing.T) {
	tpl := parseAll(t)
	f := Field{
		ID:      "bio",
		HTML:    template.HTML("<p>hello</p>"),
		EditURL: "/bio/edit",
	}.WithDefaults()
	out := render(t, tpl, "mdedit/display", f)

	for _, want := range []string{`id="mdedit-bio"`, "<p>hello</p>", `hx-get="/bio/edit"`, `hx-target="#mdedit-bio"`} {
		if !strings.Contains(out, want) {
			t.Errorf("display missing %q in:\n%s", want, out)
		}
	}
}

func TestDisplay_EmptyHTMLShowsPlaceholderNotButtonlessVoid(t *testing.T) {
	tpl := parseAll(t)
	out := render(t, tpl, "mdedit/display", Field{ID: "x"}.WithDefaults())
	if !strings.Contains(out, "mdedit-empty") {
		t.Errorf("empty display should show placeholder, got:\n%s", out)
	}
	if strings.Contains(out, "mdedit-edit-btn") {
		t.Errorf("no EditURL means no edit button, got:\n%s", out)
	}
}

func TestEdit_WiresSeamAttributesAndPreview(t *testing.T) {
	tpl := parseAll(t)
	f := Field{
		ID:          "post",
		Markdown:    "# hi",
		DisplayURL:  "/post",
		SaveURL:     "/post/save",
		PreviewURL:  "/post/preview",
		LivePreview: true,
		Label:       "Body",
		MaxLength:   5000,
	}.WithDefaults()
	out := render(t, tpl, "mdedit/edit", f)

	for _, want := range []string{
		`data-mdedit`, `data-mdedit-adapter="easymde"`,
		`data-mdedit-toolbar="full"`,
		`data-mdedit-preview-url="/post/preview"`,
		`data-mdedit-preview-target="#mdedit-preview-post"`,
		`data-mdedit-live="1"`,
		`hx-post="/post/save"`,
		`maxlength="5000"`,
		`# hi`, // markdown round-trips into the textarea
		`id="mdedit-preview-post"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("edit missing %q in:\n%s", want, out)
		}
	}
}

func TestEdit_NoPreviewURLOmitsPane(t *testing.T) {
	tpl := parseAll(t)
	f := Field{ID: "n", SaveURL: "/s", DisplayURL: "/d"}.WithDefaults()
	out := render(t, tpl, "mdedit/edit", f)
	if strings.Contains(out, "mdedit-preview-btn") || strings.Contains(out, `id="mdedit-preview-n"`) {
		t.Errorf("no PreviewURL should omit preview button and pane, got:\n%s", out)
	}
}

func TestEdit_MarkdownIsEscapedInTextarea(t *testing.T) {
	tpl := parseAll(t)
	// A closing </textarea> in the source must not break out of the field.
	f := Field{ID: "x", SaveURL: "/s", DisplayURL: "/d", Markdown: "</textarea><script>alert(1)</script>"}.WithDefaults()
	out := render(t, tpl, "mdedit/edit", f)
	if strings.Contains(out, "<script>alert(1)</script>") {
		t.Errorf("textarea content must be HTML-escaped, got:\n%s", out)
	}
}

func TestHeadTags_EmitsPinnedSRI(t *testing.T) {
	out := string(HeadTags("/static/mdedit/"))
	for _, want := range []string{
		`/static/mdedit/vendor/easymde.min.css`,
		`/static/mdedit/vendor/easymde.min.js`,
		easyMDEJSIntegrity,
		easyMDECSSIntegrity,
		`/static/mdedit/adapters/easymde.js`,
		`/static/mdedit/mdedit.js`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("HeadTags missing %q in:\n%s", want, out)
		}
	}
	// Trailing slash on staticBase must be normalized (no //).
	if strings.Contains(out, "//vendor") {
		t.Errorf("HeadTags should normalize trailing slash, got:\n%s", out)
	}
}

func TestWithDefaults(t *testing.T) {
	f := Field{}.WithDefaults()
	if f.Adapter != "easymde" {
		t.Errorf("default adapter = %q, want easymde", f.Adapter)
	}
	if f.Rows != 12 {
		t.Errorf("default rows = %d, want 12", f.Rows)
	}
	if f.Toolbar != "full" {
		t.Errorf("default toolbar = %q, want full", f.Toolbar)
	}
}

func TestEdit_AllowFileLoadRendersInput(t *testing.T) {
	tpl := parseAll(t)
	f := Field{ID: "post", SaveURL: "/s", DisplayURL: "/d", AllowFileLoad: true}.WithDefaults()
	out := render(t, tpl, "mdedit/edit", f)
	for _, want := range []string{
		`type="file"`,
		`data-mdedit-load="mdedit-ta-post"`,               // wires to this field's textarea
		`accept=".md,.markdown,text/markdown,text/plain"`, // text only, no image upload here
		"mdedit-load",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("AllowFileLoad edit missing %q in:\n%s", want, out)
		}
	}
}

func TestEdit_NoFileLoadByDefault(t *testing.T) {
	tpl := parseAll(t)
	f := Field{ID: "post", SaveURL: "/s", DisplayURL: "/d"}.WithDefaults()
	out := render(t, tpl, "mdedit/edit", f)
	for _, absent := range []string{`type="file"`, "data-mdedit-load", "mdedit-load"} {
		if strings.Contains(out, absent) {
			t.Errorf("default (AllowFileLoad false) should omit %q, got:\n%s", absent, out)
		}
	}
}

func TestEdit_UploadURLWiresAttribute(t *testing.T) {
	tpl := parseAll(t)
	f := Field{ID: "note", SaveURL: "/s", DisplayURL: "/d", UploadURL: "/campaign/x/notes/image"}.WithDefaults()
	out := render(t, tpl, "mdedit/edit", f)
	if !strings.Contains(out, `data-mdedit-upload="/campaign/x/notes/image"`) {
		t.Errorf("UploadURL should wire data-mdedit-upload, got:\n%s", out)
	}
}

func TestEdit_NoUploadByDefault(t *testing.T) {
	tpl := parseAll(t)
	f := Field{ID: "note", SaveURL: "/s", DisplayURL: "/d"}.WithDefaults()
	out := render(t, tpl, "mdedit/edit", f)
	if strings.Contains(out, "data-mdedit-upload") {
		t.Errorf("no UploadURL should omit data-mdedit-upload, got:\n%s", out)
	}
}

func TestEdit_ToolbarProfileRenders(t *testing.T) {
	tpl := parseAll(t)
	f := Field{ID: "c", SaveURL: "/s", DisplayURL: "/d", Toolbar: "standard"}.WithDefaults()
	out := render(t, tpl, "mdedit/edit", f)
	if !strings.Contains(out, `data-mdedit-toolbar="standard"`) {
		t.Errorf("explicit toolbar profile should render, got:\n%s", out)
	}
}
