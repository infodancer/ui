package ui

import (
	"fmt"
	"html/template"
	"strings"
)

// The action tracker is "Layer 1" mechanism, like htmx: a small authored asset
// served via [AssetsFS] plus a head-tag emitter, with no dependency on any
// base template. It is the JS companion to the ui/analytics partial — that
// partial emits the analytics vendor's <script>; track.js turns declarative
// data-track attributes into that vendor's custom events.
//
// It is deliberately separate from [HeadTags] (htmx): a consumer opts into the
// tracker with [TrackerHead] independently of its interactivity stack. ui owns
// the mechanism; the consumer owns the vocabulary (which elements carry
// data-track and what the event names/categories mean). See assets/js/track.js
// for the attribute contract.

// trackerFilename is ui's authored action tracker under assets/js/.
const trackerFilename = "track.js"

// TrackerHead returns the <script> tag that loads ui's action tracker. Place it
// in your page <head>. staticBase is the URL prefix where you mounted
// [AssetsFS] (e.g. "/static/ui", with or without a trailing slash); the script
// is served from "<staticBase>/js/track.js".
//
// Pair it with the ui/analytics partial (which loads the analytics vendor) and
// add data-track attributes to the actions you want measured:
//
//	<form ... data-track="profile_update" data-track-category="profile">
//	<button ... data-track="tool_generate" data-track-category="generator" data-track-tool="name">
//
// Unlike [HeadTags], the tag carries no Subresource Integrity hash: track.js is
// a first-party asset served same-origin from your own [AssetsFS] mount and
// iterated alongside the module, so SRI (which defends against a tampered
// third-party/CDN file, as with vendored htmx) buys nothing here. The script is
// deferred, so it executes after the document parses.
func TrackerHead(staticBase string) template.HTML {
	base := strings.TrimRight(staticBase, "/")
	return template.HTML(fmt.Sprintf( //nolint:gosec // base is the consumer's fixed mount path, HTML-escaped below
		`<script src="%s/js/%s" defer></script>`,
		template.HTMLEscapeString(base), trackerFilename,
	))
}
