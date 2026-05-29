package ui_test

import (
	"crypto/sha512"
	"encoding/base64"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/infodancer/ui"
)

// TestHTMXAssetEmbeddedAndServable confirms the vendored htmx file is reachable
// through AssetsFS at the path HeadTags points to, and that its SRI hash in the
// emitted tag actually matches the bytes — so the browser won't reject it.
func TestHTMXAssetEmbeddedAndServable(t *testing.T) {
	assets := ui.AssetsFS()

	path := "js/htmx-" + ui.HTMXVersion + ".min.js"
	f, err := assets.Open(path)
	if err != nil {
		t.Fatalf("htmx asset not embedded at %s: %v", path, err)
	}
	body, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if len(body) == 0 {
		t.Fatal("htmx asset is empty")
	}
	if !strings.Contains(string(body), "htmx") {
		t.Fatal("embedded asset does not look like htmx")
	}

	// The integrity attribute in HeadTags must match the embedded bytes.
	sum := sha512.Sum384(body)
	want := "sha384-" + base64.StdEncoding.EncodeToString(sum[:])
	tag := string(ui.HeadTags("/static/ui"))
	if !strings.Contains(tag, want) {
		t.Fatalf("HeadTags SRI does not match embedded asset.\n got tag: %s\n want hash: %s", tag, want)
	}

	// And it must actually serve through an http.FileServer over AssetsFS.
	srv := httptest.NewServer(http.StripPrefix("/static/ui/",
		http.FileServer(http.FS(assets))))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/static/ui/" + path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("serving htmx asset: status %d, want 200", resp.StatusCode)
	}
}

func TestHeadTags(t *testing.T) {
	tag := string(ui.HeadTags("/static/ui"))
	for _, want := range []string{
		`src="/static/ui/js/htmx-` + ui.HTMXVersion + `.min.js"`,
		`integrity="sha384-`,
		`crossorigin="anonymous"`,
		"defer",
	} {
		if !strings.Contains(tag, want) {
			t.Errorf("HeadTags missing %q in: %s", want, tag)
		}
	}

	// Trailing slash on the base must not double up in the src.
	if strings.Contains(string(ui.HeadTags("/static/ui/")), "ui//js") {
		t.Error("HeadTags did not trim a trailing slash on staticBase")
	}
}

func TestRequestHelpers(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if ui.IsRequest(r) {
		t.Error("plain request should not be detected as htmx")
	}
	r.Header.Set("HX-Request", "true")
	if !ui.IsRequest(r) {
		t.Error("HX-Request: true should be detected")
	}
	if ui.IsBoosted(r) {
		t.Error("not boosted")
	}
	r.Header.Set("HX-Boosted", "true")
	if !ui.IsBoosted(r) {
		t.Error("HX-Boosted: true should be detected")
	}
	r.Header.Set("HX-Target", "note-body")
	if got := ui.Target(r); got != "note-body" {
		t.Errorf("Target = %q, want note-body", got)
	}
}

func TestResponseHelpers(t *testing.T) {
	cases := []struct {
		name   string
		apply  func(http.ResponseWriter)
		header string
		want   string
	}{
		{"redirect", func(w http.ResponseWriter) { ui.Redirect(w, "/x") }, "HX-Redirect", "/x"},
		{"refresh", func(w http.ResponseWriter) { ui.Refresh(w) }, "HX-Refresh", "true"},
		{"pushurl", func(w http.ResponseWriter) { ui.PushURL(w, "/y") }, "HX-Push-Url", "/y"},
		{"retarget", func(w http.ResponseWriter) { ui.Retarget(w, "#z") }, "HX-Retarget", "#z"},
		{"reswap", func(w http.ResponseWriter) { ui.Reswap(w, "innerHTML") }, "HX-Reswap", "innerHTML"},
		{"trigger", func(w http.ResponseWriter) { ui.Trigger(w, "saved") }, "HX-Trigger", "saved"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c.apply(rec)
			if got := rec.Header().Get(c.header); got != c.want {
				t.Errorf("%s = %q, want %q", c.header, got, c.want)
			}
		})
	}
}

// TestHTMXIsOnlyAsset guards the Layer-1/Layer-2 independence invariant from the
// asset side: shipping htmx must not have dragged in a base document template.
// (Layer 2, when it lands, is a partial — it must stay optional.)
func TestHTMXIsOnlyAsset(t *testing.T) {
	_, err := fs.Stat(ui.PartialsFS(), "base.gohtml")
	if err == nil {
		t.Fatal("a base.gohtml partial appeared; Layer 2 must stay separate/optional " +
			"and this test updated deliberately when it lands")
	}
}
