// Package ui exposes the infodancer/ui shared design tokens, base
// stylesheet, and nav/footer partials to Go html/template consumers.
//
// The same files also ship the repo as a Hugo module for Hugo consumers
// (see hugo.toml + layouts/ + assets/ at the repo root). This package is
// just the Go-facing API.
//
// Quickstart:
//
//	import "github.com/infodancer/ui"
//
//	// Serve CSS under /static/ui/css/{tokens,base}.css
//	mux.Handle("/static/ui/", http.StripPrefix("/static/ui/", http.FileServer(http.FS(ui.AssetsFS()))))
//
//	// Parse partials alongside your own templates
//	tmpl, err := template.ParseFS(ui.PartialsFS(), "*.gohtml")
//
//	// Render via {{ template "ui/nav" .Nav }} and {{ template "ui/footer" .Footer }}
//
// See DESIGN.md at the repo root for the token vocabulary, partial data
// shapes, and versioning policy.
package ui

import (
	"bytes"
	"embed"
	"encoding/json"
	"html/template"
	"io/fs"
)

//go:embed assets
var assetsRoot embed.FS

//go:embed partials/*.gohtml
var partialsRoot embed.FS

// AssetsFS returns an [fs.FS] rooted at the `assets/` directory. It
// contains `css/tokens.css`, `css/base.css`, and the vendored htmx library
// at `js/htmx-<version>.min.js` (see [HeadTags] and [HTMXVersion]). Mount it
// via [http.FileServer] under your static path; consumers typically link the
// two CSS files in <head> with a third site-specific stylesheet loaded
// after for palette overrides, and add htmx with [HeadTags] if they want it.
func AssetsFS() fs.FS {
	sub, err := fs.Sub(assetsRoot, "assets")
	if err != nil {
		// Unreachable: the embed directive guarantees "assets" exists.
		panic("ui: AssetsFS: " + err.Error())
	}
	return sub
}

// PartialsFS returns an [fs.FS] containing the Go html/template partials
// (`nav.gohtml`, `footer.gohtml`, `sidebar.gohtml`, `meta.gohtml`). Parse them
// with [template.ParseFS] alongside your own templates. The partials define
// templates `ui/nav`, `ui/footer`, `ui/sidebar`, and `ui/meta`; render via
// `{{ template "ui/nav" .Nav }}` etc.
func PartialsFS() fs.FS {
	sub, err := fs.Sub(partialsRoot, "partials")
	if err != nil {
		panic("ui: PartialsFS: " + err.Error())
	}
	return sub
}

// NavData is the documented data shape for the `ui/nav` partial. Any
// struct with the same fields works (html/template duck-types by name);
// this type is provided as a convenience.
type NavData struct {
	BrandText string
	BrandURL  string // defaults to "/" when empty
	Links     []NavLink
	User      *NavUser // nil renders the sign-in link
	SignInURL string   // defaults to "/login" when empty
}

// NavLink is a primary nav link.
type NavLink struct {
	Label string
	URL   string
}

// NavUser carries authenticated-user display info. When nil on NavData,
// the partial renders the sign-in link instead.
type NavUser struct {
	DisplayName string
}

// FooterData is the documented data shape for the `ui/footer` partial.
// Any struct with the same fields works.
type FooterData struct {
	BrandText string
	BrandURL  string // defaults to "/" when empty
	Copyright string // free-form; consumer formats (e.g. "© 2026 Example.org")
	Links     []FooterLink
}

// FooterLink is a footer link.
type FooterLink struct {
	Label string
	URL   string
}

// SidebarData is the documented data shape for the `ui/sidebar` partial: one
// aside panel of collapsible link sections. Any struct with the same fields
// works. The partial is side-unaware — the page layout decides whether this
// is a left or right aside (add .has-left / .has-right to .app-sidebar-layout
// and wrap the render in the matching <aside>). A page may render two,
// passing a separate SidebarData to each side.
type SidebarData struct {
	Sections []SidebarSection
}

// SidebarSection is one collapsible group within a sidebar. Key is a stable
// identifier emitted as data-sidebar-key, so a consumer's optional
// persistence script can remember the open/closed state across pages. Open
// sets the default expanded state.
type SidebarSection struct {
	Key   string
	Title string
	Open  bool
	Items []SidebarItem
}

// SidebarItem is one link in a sidebar section. Meta is an optional muted
// secondary line (a timestamp, a campaign name, a count).
type SidebarItem struct {
	Label string
	URL   string
	Meta  string
}

// Meta is the documented data shape for the `ui/meta` partial: the SEO and
// social <head> tags a page emits. Any struct with the same fields works.
//
// Every field is optional — an empty field emits nothing, so a page fills
// only what it has. ui renders the tags but has no opinion about their
// content: the description copy, canonical policy, and the schema.org graph
// are the consuming app's domain, not a generic toolkit's. The partial wires
// the values it's given into the standard description / OpenGraph / Twitter /
// JSON-LD markup.
//
// Defaults the partial supplies when a field is empty:
//   - Type   -> "website" (og:type)
//   - the Twitter card is "summary_large_image" when Image is set, else "summary"
//
// JSONLD carries complete <script type="application/ld+json"> elements. Build
// each with [JSONLD], which marshals a value and escapes it so it cannot break
// out of the script element. The slot is plural because a page commonly emits
// several graphs (e.g. an Article plus a BreadcrumbList).
type Meta struct {
	Description string          // meta description + og/twitter description
	Canonical   string          // <link rel="canonical"> + og:url
	Title       string          // og:title / twitter:title (the page <title> stays the page's own)
	SiteName    string          // og:site_name
	Type        string          // og:type; defaults "website"
	Image       string          // absolute URL; og:image + twitter:image
	Locale      string          // og:locale (e.g. "en_US")
	JSONLD      []template.HTML // complete <script> elements; build with [JSONLD]
}

// JSONLD marshals v as JSON-LD and wraps it in a
// <script type="application/ld+json"> element suitable for the document
// <head>. v is typically a map[string]any (or a slice/struct) describing a
// schema.org graph.
//
// Marshaling uses encoding/json, which escapes '<', '>', and '&' as \u00XX
// sequences. That keeps a value containing "</script>" from terminating the
// surrounding element — the data can carry arbitrary text without enabling a
// markup-injection breakout. The result is therefore safe to emit verbatim as
// template.HTML.
func JSONLD(v any) (template.HTML, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	buf.WriteString(`<script type="application/ld+json">`)
	buf.Write(b)
	buf.WriteString(`</script>`)
	return template.HTML(buf.String()), nil //nolint:gosec // json.Marshal escapes <,>,& so the body cannot break out of the script element
}
