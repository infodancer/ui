// Package mdedit is a reusable Markdown display/edit component for the
// infodancer/matthewjhunter Go + htmx stack.
//
// It ships three things, mirroring the infodancer/ui consumption model:
//
//   - [AssetsFS] — the client assets (a vendored editor, the adapter seam,
//     the loader, and token-styled CSS) to mount under a static path.
//   - [PartialsFS] — Go html/template partials defining `mdedit/display`,
//     `mdedit/edit`, and `mdedit/preview`, parsed alongside host templates.
//   - [Field] — the data shape those partials render.
//
// The component owns the display↔edit↔save↔preview UI loop and the client
// editor seam. It does NOT own storage, authentication, or authorization:
// the host application wires handlers for the Field URLs and decides who
// may edit and where the Markdown is persisted. Rendering goes through the
// sibling markdown module (one audited goldmark + bluemonday policy).
//
// Editing is inherently a server-backed (Go + htmx) concern — there is no
// Hugo variant of these partials, because a static site has no backend to
// store the result. Hugo sites consume the rendered HTML, never the editor.
package mdedit

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"strings"
)

//go:embed assets
var assetsRoot embed.FS

//go:embed partials/*.gohtml
var partialsRoot embed.FS

// Subresource Integrity hashes for the vendored, pinned editor files. See
// assets/vendor/PROVENANCE.md for the source and how to recompute these.
const (
	easyMDEJSIntegrity  = "sha384-YDXeUfPZ4SP6vJpnF+ZMmf4B1bax6yd4Q/aNbkvLidRD843hPG5RE67M0IYT4LOq"
	easyMDECSSIntegrity = "sha384-3AvV7152TgYAMYdGZPqG9BpmSH2ZW6ewTDL0QV5PyNkl19KMI+yLMdJz183N8A2d"
)

// AssetsFS returns an [fs.FS] rooted at the component's `assets/` directory.
// Mount it via [http.FileServer] under a static path (the same prefix you
// pass to [HeadTags]). It contains:
//
//	vendor/easymde.min.{js,css}  pinned third-party editor (see PROVENANCE.md)
//	adapters/easymde.js          the EasyMDE adapter registration
//	mdedit.js                    the loader + adapter registry + value sync
//	mdedit.css                   token-styled component CSS (uses --app-* vars)
func AssetsFS() fs.FS {
	sub, err := fs.Sub(assetsRoot, "assets")
	if err != nil {
		panic("mdedit: AssetsFS: " + err.Error()) // unreachable: embed guarantees the dir
	}
	return sub
}

// PartialsFS returns an [fs.FS] of the Go html/template partials. Parse them
// with [template.ParseFS] alongside your own templates; they define
// `mdedit/display`, `mdedit/edit`, and `mdedit/preview`, each rendered
// against a [Field].
func PartialsFS() fs.FS {
	sub, err := fs.Sub(partialsRoot, "partials")
	if err != nil {
		panic("mdedit: PartialsFS: " + err.Error())
	}
	return sub
}

// HeadTags returns the <link>/<script> tags to place in the host page's
// <head>. staticBase is the URL prefix where AssetsFS is mounted (no
// trailing slash), e.g. "/static/mdedit". The vendored files carry SRI
// hashes; htmx itself is the host's responsibility and is not emitted here.
//
// Emitted in dependency order: the vendor bundle defines the editor global,
// the loader defines window.mdedit, then the adapter registers against it.
// All are deferred. (The seam also tolerates an adapter loading before the
// loader by queuing its registration, so order is robust, not fragile.)
func HeadTags(staticBase string) template.HTML {
	b := strings.TrimRight(staticBase, "/")
	var sb strings.Builder
	fmt.Fprintf(&sb, `<link rel="stylesheet" href="%s/vendor/easymde.min.css" integrity="%s" crossorigin="anonymous">`+"\n", b, easyMDECSSIntegrity)
	fmt.Fprintf(&sb, `<link rel="stylesheet" href="%s/mdedit.css">`+"\n", b)
	fmt.Fprintf(&sb, `<script defer src="%s/vendor/easymde.min.js" integrity="%s" crossorigin="anonymous"></script>`+"\n", b, easyMDEJSIntegrity)
	fmt.Fprintf(&sb, `<script defer src="%s/mdedit.js"></script>`+"\n", b)
	fmt.Fprintf(&sb, `<script defer src="%s/adapters/easymde.js"></script>`+"\n", b)
	return template.HTML(sb.String()) //nolint:gosec // fixed template, staticBase is host-controlled config
}

// Field is the data a host handler passes to the mdedit partials. One Field
// represents a single editable Markdown region through its whole lifecycle;
// the same struct drives display, edit, and preview renders.
//
// ID must be unique on the page (it keys the wrapper element and the htmx
// swap target). The four URLs are the loop's endpoints, all host-owned:
//
//	DisplayURL  GET  → returns the `mdedit/display` partial (Cancel target)
//	EditURL     GET  → returns the `mdedit/edit` partial (Edit button target)
//	SaveURL     POST → persists form field "markdown", returns display partial
//	PreviewURL  POST → renders form field "markdown", returns `mdedit/preview`
//	            ("" disables the preview pane entirely)
type Field struct {
	ID       string        // unique per page; keys the wrapper + swap target
	Markdown string        // raw source (edit mode + persistence)
	HTML     template.HTML // sanitized render (display mode); set via markdown package

	DisplayURL string
	EditURL    string
	SaveURL    string
	PreviewURL string // "" disables preview

	Label     string // optional textarea label
	Rows      int    // textarea rows; defaults to 12 when zero
	MaxLength int    // textarea maxlength; omitted when zero

	// Adapter names the client editor to mount ("easymde" is the only one
	// registered today). Empty defaults to "easymde". This is the seam:
	// swapping editors is a data change, not a template change.
	Adapter string

	// Toolbar selects which editing controls the adapter shows, matching
	// how constrained the field's content is. Empty defaults to "full".
	//
	//	"minimal"  bold, italic, link — for tiny inputs
	//	"standard" + strikethrough, inline code, lists, blockquote — the
	//	           comment subset (pair with markdown.Comment on the server)
	//	"full"     + headings, code blocks — long-form pages (session notes)
	//
	// The toolbar is a display affordance only; the server's markdown
	// preset is what actually constrains what survives. Keep the two in
	// agreement (a "standard" toolbar with a Rich server preset just means
	// authors can hand-type Markdown the toolbar has no button for).
	Toolbar string

	// LivePreview opts this field into debounced server-rendered preview as
	// the author types (the dual-render experiment). When false, preview is
	// button-triggered only. Requires PreviewURL.
	LivePreview bool

	// AllowFileLoad adds a control that lets the author load a local Markdown
	// file into the editor, as if they had typed it. The file is read in the
	// browser and dropped into the textarea — it never leaves the client
	// except via the normal Save POST, so it adds no server attack surface
	// and goes through the same render/sanitize path as typed Markdown.
	// Loading replaces the current content (with a confirm if it is
	// non-empty). Off by default; enable it for long-form pages, not comments.
	AllowFileLoad bool

	Errors []string // validation messages shown above the edit form
}

// WithDefaults returns a copy of f with zero-valued display fields filled
// in (Adapter "easymde", Toolbar "full", Rows 12). Handlers can call it
// before rendering so templates stay free of default logic.
func (f Field) WithDefaults() Field {
	if f.Adapter == "" {
		f.Adapter = "easymde"
	}
	if f.Toolbar == "" {
		f.Toolbar = "full"
	}
	if f.Rows == 0 {
		f.Rows = 12
	}
	return f
}
