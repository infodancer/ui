package ui

import (
	"fmt"
	"html/template"
	"strings"
)

// The lightbox is "Layer 1" mechanism, like the action tracker: a small
// authored asset served via [AssetsFS] plus a head-tag emitter, with no
// dependency on any base template. It is the kind of widget every consumer
// site otherwise pulls from npm (mjh shipped simplelightbox) or re-implements;
// ui owns the mechanism, the consumer owns the markup. See
// assets/js/lightbox.js for the attribute contract.

// lightboxFilename is ui's authored lightbox under assets/js/.
const lightboxFilename = "lightbox.js"

// LightboxHead returns the <script> tag that loads ui's image lightbox. Place
// it in your page <head>. staticBase is the URL prefix where you mounted
// [AssetsFS] (e.g. "/static/ui", with or without a trailing slash); the script
// is served from "<staticBase>/js/lightbox.js".
//
// Mark up images as links to their full-size files:
//
//	<a href="photo-full.jpg" data-lightbox="trip"><img src="thumb.jpg" alt="A caption"></a>
//
// Links sharing a data-lightbox value form a gallery: prev/next buttons and
// arrow keys navigate it in document order, with a position counter. The
// caption comes from data-lightbox-caption, else the linked image's alt text.
// Without JavaScript the links simply open the full image — the lightbox is
// progressive enhancement.
//
// Like [TrackerHead] and unlike [HeadTags], the tag carries no Subresource
// Integrity hash: lightbox.js is a first-party asset served same-origin from
// your own [AssetsFS] mount and iterated alongside the module, so SRI (which
// defends against a tampered third-party/CDN file, as with vendored htmx)
// buys nothing here. The script is deferred, so it executes after the
// document parses.
func LightboxHead(staticBase string) template.HTML {
	base := strings.TrimRight(staticBase, "/")
	return template.HTML(fmt.Sprintf( //nolint:gosec // base is the consumer's fixed mount path, HTML-escaped below
		`<script src="%s/js/%s" defer></script>`,
		template.HTMLEscapeString(base), lightboxFilename,
	))
}
