package ui_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/infodancer/ui"
)

// TestLightboxAssetEmbeddedAndServable confirms the authored lightbox is
// reachable through AssetsFS at the path LightboxHead points to.
func TestLightboxAssetEmbeddedAndServable(t *testing.T) {
	assets := ui.AssetsFS()

	const path = "js/lightbox.js"
	f, err := assets.Open(path)
	if err != nil {
		t.Fatalf("lightbox asset not embedded at %s: %v", path, err)
	}
	body, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if len(body) == 0 {
		t.Fatal("lightbox asset is empty")
	}
	if !strings.Contains(string(body), "data-lightbox") {
		t.Fatal("embedded asset does not look like the lightbox")
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
		t.Fatalf("serving lightbox asset: status %d, want 200", resp.StatusCode)
	}
}

func TestLightboxHead(t *testing.T) {
	tag := string(ui.LightboxHead("/static/ui"))
	for _, want := range []string{
		`src="/static/ui/js/lightbox.js"`,
		"defer",
	} {
		if !strings.Contains(tag, want) {
			t.Errorf("LightboxHead missing %q in: %s", want, tag)
		}
	}
	// First-party asset: no SRI (see LightboxHead doc).
	if strings.Contains(tag, "integrity=") {
		t.Errorf("LightboxHead should not carry an SRI hash: %s", tag)
	}
	// Trailing slash on the base must not double up in the src.
	if strings.Contains(string(ui.LightboxHead("/static/ui/")), "ui//js") {
		t.Error("LightboxHead did not trim a trailing slash on staticBase")
	}
}
