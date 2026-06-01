package ui_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/infodancer/ui"
)

// TestTrackerAssetEmbeddedAndServable confirms the authored tracker is reachable
// through AssetsFS at the path TrackerHead points to.
func TestTrackerAssetEmbeddedAndServable(t *testing.T) {
	assets := ui.AssetsFS()

	const path = "js/track.js"
	f, err := assets.Open(path)
	if err != nil {
		t.Fatalf("tracker asset not embedded at %s: %v", path, err)
	}
	body, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if len(body) == 0 {
		t.Fatal("tracker asset is empty")
	}
	if !strings.Contains(string(body), "uiTrack") {
		t.Fatal("embedded asset does not look like the tracker")
	}

	srv := httptest.NewServer(http.StripPrefix("/static/ui/",
		http.FileServer(http.FS(assets))))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/static/ui/" + path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("serving tracker asset: status %d, want 200", resp.StatusCode)
	}
}

func TestTrackerHead(t *testing.T) {
	tag := string(ui.TrackerHead("/static/ui"))
	for _, want := range []string{
		`src="/static/ui/js/track.js"`,
		"defer",
	} {
		if !strings.Contains(tag, want) {
			t.Errorf("TrackerHead missing %q in: %s", want, tag)
		}
	}
	// First-party asset: no SRI (see TrackerHead doc).
	if strings.Contains(tag, "integrity=") {
		t.Errorf("TrackerHead should not carry an SRI hash: %s", tag)
	}
	// Trailing slash on the base must not double up in the src.
	if strings.Contains(string(ui.TrackerHead("/static/ui/")), "ui//js") {
		t.Error("TrackerHead did not trim a trailing slash on staticBase")
	}
}
