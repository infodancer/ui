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
	"embed"
	"io/fs"
)

//go:embed assets
var assetsRoot embed.FS

//go:embed partials/*.gohtml
var partialsRoot embed.FS

// AssetsFS returns an [fs.FS] rooted at the `assets/` directory. It
// contains `css/tokens.css` and `css/base.css`. Mount it via
// [http.FileServer] under your static path; consumers typically link the
// two CSS files in <head> with a third site-specific stylesheet loaded
// after for palette overrides.
func AssetsFS() fs.FS {
	sub, err := fs.Sub(assetsRoot, "assets")
	if err != nil {
		// Unreachable: the embed directive guarantees "assets" exists.
		panic("ui: AssetsFS: " + err.Error())
	}
	return sub
}

// PartialsFS returns an [fs.FS] containing the Go html/template partials
// (`nav.gohtml`, `footer.gohtml`). Parse them with [template.ParseFS]
// alongside your own templates. The partials define templates `ui/nav`
// and `ui/footer`; render via `{{ template "ui/nav" .Nav }}` etc.
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
