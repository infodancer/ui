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

	// Prev/next-only pager (Pages nil) renders no numbered strip — guards the
	// additive change against regressing the offset pager.
	got = render(ui.Pager{PrevURL: "/x?offset=0", NextURL: "/x?offset=100", Position: "51–100"})
	if strings.Contains(got, "app-pager-pages") {
		t.Errorf("prev/next-only pager should render no numbered strip:\n%s", got)
	}

	// Numbered pager renders the strip with a current cell and an ellipsis.
	got = render(ui.NumberedPager("/x", url.Values{}, 7, 50, 50*23))
	for _, want := range []string{
		`app-pager-pages`,
		`class="app-pager-page is-current" aria-current="page"`,
		`app-pager-ellipsis`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("numbered pager markup missing %q\n%s", want, got)
		}
	}
}

func TestNumberedPager(t *testing.T) {
	const base, size = "/bestiary/", 50

	// labels flattens a Pager's numbered strip to "label" / "[label]" (current)
	// / "…" (ellipsis), so a test reads as the visible strip.
	labels := func(p ui.Pager) []string {
		out := make([]string, 0, len(p.Pages))
		for _, pl := range p.Pages {
			switch {
			case pl.Current:
				out = append(out, "["+pl.Label+"]")
			case pl.URL == "":
				out = append(out, "…")
			default:
				out = append(out, pl.Label)
			}
		}
		return out
	}
	eq := func(t *testing.T, got, want []string) {
		t.Helper()
		if strings.Join(got, " ") != strings.Join(want, " ") {
			t.Errorf("strip = %v, want %v", got, want)
		}
	}

	t.Run("mid range windows with two ellipses", func(t *testing.T) {
		p := ui.NumberedPager(base, url.Values{}, 7, size, size*23)
		eq(t, labels(p), []string{"1", "…", "5", "6", "[7]", "8", "9", "…", "23"})
		if p.Position != "Page 7 of 23" {
			t.Errorf("Position = %q", p.Position)
		}
		if !strings.Contains(p.PrevURL, "page=6") || !strings.Contains(p.NextURL, "page=8") {
			t.Errorf("prev=%q next=%q, want page 6 / page 8", p.PrevURL, p.NextURL)
		}
	})

	t.Run("few pages: no ellipsis, prev hidden on page 1", func(t *testing.T) {
		p := ui.NumberedPager(base, url.Values{}, 1, size, size*3)
		eq(t, labels(p), []string{"[1]", "2", "3"})
		if p.PrevURL != "" {
			t.Errorf("PrevURL = %q, want empty on page 1", p.PrevURL)
		}
	})

	t.Run("near start: single trailing ellipsis", func(t *testing.T) {
		p := ui.NumberedPager(base, url.Values{}, 2, size, size*23)
		eq(t, labels(p), []string{"1", "[2]", "3", "4", "…", "23"})
	})

	t.Run("near end: single leading ellipsis", func(t *testing.T) {
		p := ui.NumberedPager(base, url.Values{}, 22, size, size*23)
		eq(t, labels(p), []string{"1", "…", "20", "21", "[22]", "23"})
	})

	t.Run("single page: no strip, both URLs empty", func(t *testing.T) {
		p := ui.NumberedPager(base, url.Values{}, 1, size, 10)
		if len(p.Pages) != 0 {
			t.Errorf("Pages = %v, want none for a single page", p.Pages)
		}
		if p.PrevURL != "" || p.NextURL != "" {
			t.Errorf("single page should have no prev/next: prev=%q next=%q", p.PrevURL, p.NextURL)
		}
		if p.Position != "Page 1 of 1" {
			t.Errorf("Position = %q", p.Position)
		}
	})

	t.Run("preserves filters; page 1 link omits page param; q not mutated", func(t *testing.T) {
		q := url.Values{"tag": {"undead"}}
		p := ui.NumberedPager(base, q, 3, size, size*5)
		for _, pl := range p.Pages {
			if pl.URL == "" {
				continue
			}
			if !strings.Contains(pl.URL, "tag=undead") {
				t.Errorf("page link %q dropped the filter", pl.URL)
			}
			if pl.Label == "1" && strings.Contains(pl.URL, "page=") {
				t.Errorf("page-1 link should omit page=: %q", pl.URL)
			}
		}
		if _, ok := q["page"]; ok {
			t.Errorf("NumberedPager mutated the caller's query: %v", q)
		}
	})

	t.Run("out-of-range page clamps into bounds", func(t *testing.T) {
		p := ui.NumberedPager(base, url.Values{}, 999, size, size*3)
		if p.Position != "Page 3 of 3" {
			t.Errorf("Position = %q, want clamp to last page", p.Position)
		}
		if p.NextURL != "" {
			t.Errorf("clamped-to-last should hide next: %q", p.NextURL)
		}
	})
}
