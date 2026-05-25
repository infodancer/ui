package ui

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

// htmx is the chosen interactivity layer for the infodancer stack: the
// minified library is vendored under assets/js/ (served via [AssetsFS]) and
// the helpers below cover the request/response boundary every consumer would
// otherwise re-implement. See [HeadTags] to load it.
//
// This is "Layer 1" — pure mechanism, with no dependency on any base template.
// A consumer keeps its own HTML document, drops [HeadTags] in its <head>, and
// uses these helpers in handlers. The forthcoming optional base document
// template ("Layer 2") is just another consumer of this layer; htmx support
// here is fully usable without it. A consumer that wants a different
// interactivity stack simply never calls [HeadTags] — nothing forces htmx on.
const (
	// HTMXVersion is the vendored htmx release. The asset filename and the SRI
	// hash below are pinned to it; bump all three together.
	HTMXVersion = "2.0.10"

	htmxFilename = "htmx-" + HTMXVersion + ".min.js"
	// htmxSRI is the Subresource Integrity hash of assets/js/htmx-<ver>.min.js.
	// Regenerate when bumping the version:
	//
	//	printf 'sha384-%s\n' "$(openssl dgst -sha384 -binary assets/js/htmx-<ver>.min.js | openssl base64 -A)"
	htmxSRI = "sha384-H5SrcfygHmAuTDZphMHqBJLc3FhssKjG7w/CeCpFReSfwBWDTKpkzPP8c+cLsK+V"
)

// HeadTags returns the <script> tag that loads the vendored htmx library, with
// a Subresource Integrity hash so the browser rejects a tampered file. Place it
// in your page <head>. staticBase is the URL prefix where you mounted
// [AssetsFS] (e.g. "/static/ui", with or without a trailing slash); the script
// is served from "<staticBase>/js/<file>".
//
//	mux.Handle("/static/ui/", http.StripPrefix("/static/ui/", http.FileServer(http.FS(ui.AssetsFS()))))
//	// in the page <head>: {{ .HTMXHead }}  where HTMXHead = ui.HeadTags("/static/ui")
//
// The script is deferred, so it executes after the document parses; htmx then
// processes the DOM on load. Loading it self-hosted (not from a CDN) keeps the
// dependency inside your origin and the SRI honest.
func HeadTags(staticBase string) template.HTML {
	base := strings.TrimRight(staticBase, "/")
	return template.HTML(fmt.Sprintf(
		`<script src="%s/js/%s" integrity="%s" crossorigin="anonymous" defer></script>`,
		template.HTMLEscapeString(base), htmxFilename, htmxSRI,
	))
}

// IsRequest reports whether r was issued by htmx, via the HX-Request header.
// Use it to decide between returning a full page (direct navigation) and a
// fragment (htmx swap) from the same handler.
func IsRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// IsBoosted reports whether r came from an hx-boost'd link or form (HX-Boosted).
// Boosted requests are htmx requests that still expect a full-page-shaped
// response (htmx swaps the <body>), so handlers that branch on [IsRequest]
// usually want to treat a boosted request like direct navigation.
func IsBoosted(r *http.Request) bool {
	return r.Header.Get("HX-Boosted") == "true"
}

// Target returns the id of the element that issued the request (HX-Target), or
// "" when absent. Lets one handler serve different fragments by where the swap
// will land.
func Target(r *http.Request) string {
	return r.Header.Get("HX-Target")
}

// Redirect tells htmx to navigate the browser to url (HX-Redirect). Use this
// instead of [http.Redirect] when responding to an htmx request: htmx swaps the
// response body and ignores a 3xx Location, so a normal redirect silently does
// nothing.
func Redirect(w http.ResponseWriter, url string) {
	w.Header().Set("HX-Redirect", url)
}

// Refresh tells htmx to do a full page reload (HX-Refresh), discarding the
// swap. Useful after a change that invalidates the whole page.
func Refresh(w http.ResponseWriter) {
	w.Header().Set("HX-Refresh", "true")
}

// PushURL pushes url into the browser address bar and history (HX-Push-Url) so
// a fragment swap leaves a bookmarkable, back-button-able URL. Pass "false" to
// suppress htmx's default history push.
func PushURL(w http.ResponseWriter, url string) {
	w.Header().Set("HX-Push-Url", url)
}

// Retarget overrides the element the response is swapped into (HX-Retarget);
// css is a CSS selector. Lets a handler redirect its own output elsewhere —
// e.g. swap an error banner into a different region than the form that posted.
func Retarget(w http.ResponseWriter, css string) {
	w.Header().Set("HX-Retarget", css)
}

// Reswap overrides how the response is swapped (HX-Reswap), e.g. "outerHTML",
// "innerHTML", "beforeend", or "none".
func Reswap(w http.ResponseWriter, spec string) {
	w.Header().Set("HX-Reswap", spec)
}

// Trigger asks the client to fire a named DOM event after the swap
// (HX-Trigger), which other elements can listen for to coordinate updates.
func Trigger(w http.ResponseWriter, event string) {
	w.Header().Set("HX-Trigger", event)
}
