package ui

import "html/template"

// DocumentData is the documented data shape for the `ui/document` partial —
// the "Layer 2" base document skeleton. It renders a complete <html> page that
// wires in the Layer 1 pieces (CSS, ui/meta, htmx via HeadTags, nav, footer,
// sidebars) so a consumer that wants the full shell doesn't hand-roll one.
// Any struct with these fields works (html/template duck-types by name);
// embedding DocumentData in a per-page struct is the typical pattern, so a
// handler can carry page data alongside the chrome.
//
// Content arrives through template blocks, not a field: the consumer's page
// template defines the blocks and ends with `{{template "ui/document" .}}`.
// The block contract:
//
//   - `content` — REQUIRED. The page body; rendered inside the main column.
//   - `title`   — optional. The <title> text; defaults to Meta.SiteName.
//   - `head`    — optional. Extra <head> markup (page <style>, preload, etc.).
//   - `nav`     — optional. Overrides the default `{{template "ui/nav" .Nav}}`.
//   - `footer`  — optional. Overrides the default `{{template "ui/footer" .Footer}}`.
//
// Layer 2 depends on Layer 1, never the reverse: a consumer can keep using the
// individual partials and [HeadTags] without ever rendering ui/document.
type DocumentData struct {
	Lang      string // <html lang>; defaults to "en"
	Theme     string // <html data-theme>; empty omits the attribute
	AssetBase string // URL prefix where AssetsFS is mounted, e.g. "/static/ui"

	// AssetVersion, when non-empty, is appended as a ?v= query to the ui CSS
	// links so a deploy can bust the cache after a ui upgrade. ExtraCSS URLs
	// carry their own versioning (the consumer builds those strings).
	AssetVersion string

	// ExtraCSS lists additional stylesheet URLs, linked after ui's
	// open-props/tokens/base in that order — put site palette overrides here.
	ExtraCSS []string

	// HeadTags is extra <head> HTML the consumer supplies verbatim. The htmx
	// hook lives here: set it to [HeadTags](AssetBase) to load htmx, or leave
	// it zero to load none (htmx stays opt-in, consistent with Layer 1).
	HeadTags template.HTML

	Meta      Meta       // SEO/social tags via ui/meta
	Analytics *Analytics // nil emits no analytics
	Nav       NavData    // default nav (overridable by the "nav" block)
	Footer    FooterData // default footer (overridable by the "footer" block)

	SidebarLeft  *SidebarData // nil = absent
	SidebarRight *SidebarData // nil = absent
}

// Analytics is the documented data shape for the `ui/analytics` partial: the
// optional web-analytics <script> tags for the infodancer stack. ui owns the
// markup; the consumer supplies its own IDs and script URLs (nothing
// app-specific is baked into the toolkit). A nil member emits nothing, so a
// consumer enables only what it uses.
type Analytics struct {
	Umami     *Umami
	Plausible *Plausible
}

// Umami configures the Umami analytics tag. Src is the tracker script URL
// (e.g. "https://analytics.example.net/script.js"); WebsiteID is the
// data-website-id for the tracked property.
type Umami struct {
	Src       string
	WebsiteID string
}

// Plausible configures the Plausible analytics tag. Src is the script URL for
// the property (Plausible's per-site script path). The partial also emits the
// standard queue/init shim so `plausible(...)` calls before load are captured.
type Plausible struct {
	Src string
}
