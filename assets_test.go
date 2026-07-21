package ui_test

import (
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/infodancer/ui"
)

// assetCase pins one first-party JS asset: its embedded path, a marker string
// that should appear in the served bytes, and the Head function that emits
// its <script> tag.
type assetCase struct {
	name string
	path string
	// marker is a string a working copy of the asset must contain --
	// catches the wrong file being embedded at path, not just a missing one.
	marker string
	head   func(staticBase string) template.HTML
}

var assetCases = []assetCase{
	{name: "Lightbox", path: "js/lightbox.js", marker: "data-lightbox", head: ui.LightboxHead},
	{name: "Tracker", path: "js/track.js", marker: "uiTrack", head: ui.TrackerHead},
}

// TestAssetEmbeddedAndServable confirms each authored asset is reachable
// through AssetsFS at the path its Head function points to.
func TestAssetEmbeddedAndServable(t *testing.T) {
	for _, tc := range assetCases {
		t.Run(tc.name, func(t *testing.T) {
			assets := ui.AssetsFS()

			f, err := assets.Open(tc.path)
			if err != nil {
				t.Fatalf("%s asset not embedded at %s: %v", tc.name, tc.path, err)
			}
			body, err := io.ReadAll(f)
			if err != nil {
				t.Fatal(err)
			}
			_ = f.Close()
			if len(body) == 0 {
				t.Fatalf("%s asset is empty", tc.name)
			}
			if !strings.Contains(string(body), tc.marker) {
				t.Fatalf("embedded asset does not look like the %s", tc.name)
			}

			srv := httptest.NewServer(http.StripPrefix("/static/ui/",
				http.FileServer(http.FS(assets))))
			defer srv.Close()
			resp, err := http.Get(srv.URL + "/static/ui/" + tc.path)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("serving %s asset: status %d, want 200", tc.name, resp.StatusCode)
			}
		})
	}
}

func TestAssetHead(t *testing.T) {
	for _, tc := range assetCases {
		t.Run(tc.name, func(t *testing.T) {
			tag := string(tc.head("/static/ui"))
			for _, want := range []string{
				`src="/static/ui/` + tc.path + `"`,
				"defer",
			} {
				if !strings.Contains(tag, want) {
					t.Errorf("%sHead missing %q in: %s", tc.name, want, tag)
				}
			}
			// First-party asset: no SRI (see the Head function's doc).
			if strings.Contains(tag, "integrity=") {
				t.Errorf("%sHead should not carry an SRI hash: %s", tc.name, tag)
			}
			// Trailing slash on the base must not double up in the src.
			if strings.Contains(string(tc.head("/static/ui/")), "ui//js") {
				t.Errorf("%sHead did not trim a trailing slash on staticBase", tc.name)
			}
		})
	}
}
