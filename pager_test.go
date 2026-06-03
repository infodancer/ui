package ui_test

import (
	"html/template"
	"net/url"
	"strings"
	"testing"

	ui "github.com/infodancer/ui"
)

func TestNewPager(t *testing.T) {
	const base, size = "/campaign/", 50

	t.Run("first page hides prev, offset omitted in next URL when 0", func(t *testing.T) {
		p := ui.NewPager(base, url.Values{}, 0, size, size, true)
		if p.PrevURL != "" {
			t.Errorf("PrevURL = %q, want empty on first page", p.PrevURL)
		}
		if p.NextURL != "/campaign/?offset=50" {
			t.Errorf("NextURL = %q, want /campaign/?offset=50", p.NextURL)
		}
		if p.Position != "1–50" {
			t.Errorf("Position = %q, want 1–50", p.Position)
		}
	})

	t.Run("middle page preserves filters and pages both ways", func(t *testing.T) {
		q := url.Values{"system": {"osr"}, "tag": {"undead"}}
		p := ui.NewPager(base, q, 50, size, size, true)
		// prev goes to offset 0 → offset param dropped, filters kept
		if !strings.HasPrefix(p.PrevURL, base+"?") ||
			strings.Contains(p.PrevURL, "offset=") ||
			!strings.Contains(p.PrevURL, "system=osr") ||
			!strings.Contains(p.PrevURL, "tag=undead") {
			t.Errorf("PrevURL = %q, want filters but no offset", p.PrevURL)
		}
		if !strings.Contains(p.NextURL, "offset=100") ||
			!strings.Contains(p.NextURL, "system=osr") {
			t.Errorf("NextURL = %q, want offset=100 + filters", p.NextURL)
		}
		// caller's map is not mutated
		if _, ok := q["offset"]; ok {
			t.Errorf("NewPager mutated the caller's query: %v", q)
		}
	})

	t.Run("last page hides next", func(t *testing.T) {
		p := ui.NewPager(base, url.Values{}, 100, size, 17, false)
		if p.NextURL != "" {
			t.Errorf("NextURL = %q, want empty on last page", p.NextURL)
		}
		if p.Position != "101–117" {
			t.Errorf("Position = %q, want 101–117", p.Position)
		}
	})
}

func TestPagerPartial(t *testing.T) {
	tmpl := template.Must(template.New("t").ParseFS(ui.PartialsFS(), "*.gohtml"))
	render := func(p ui.Pager) string {
		var b strings.Builder
		if err := tmpl.ExecuteTemplate(&b, "ui/pager", p); err != nil {
			t.Fatalf("execute ui/pager: %v", err)
		}
		return b.String()
	}

	got := render(ui.Pager{PrevURL: "/x?offset=0", NextURL: "/x?offset=100", Position: "51–100"})
	for _, want := range []string{`class="app-pager"`, `href="/x?offset=0"`, `href="/x?offset=100"`, `app-pager-pos`, "51–100"} {
		if !strings.Contains(got, want) {
			t.Errorf("pager markup missing %q\n%s", want, got)
		}
	}

	// First page: no prev link rendered.
	got = render(ui.Pager{NextURL: "/x?offset=50", Position: "1–50"})
	if strings.Contains(got, "rel=\"prev\"") {
		t.Errorf("first page should render no prev link:\n%s", got)
	}
}
