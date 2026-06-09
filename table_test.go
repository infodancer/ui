package ui_test

import (
	"html/template"
	"net/url"
	"strings"
	"testing"

	ui "github.com/infodancer/ui"
)

func cols() []ui.Column {
	return []ui.Column{
		{Key: "name", Label: "Name"},
		{Key: "", Label: "System"}, // display-only
		{Key: "cr", Label: "CR", Align: "end"},
	}
}

func TestNewTable(t *testing.T) {
	const base = "/bestiary/"

	t.Run("active column flips dir; aria-sort reflects current", func(t *testing.T) {
		tbl := ui.NewTable(base, url.Values{}, cols(),
			ui.SortState{Key: "name", Dir: "asc"}, nil, ui.Pager{})
		name := tbl.Header[0]
		if !name.Active {
			t.Error("name column should be Active")
		}
		if name.AriaSort != "ascending" {
			t.Errorf("AriaSort = %q, want ascending", name.AriaSort)
		}
		if !strings.Contains(name.SortURL, "sort=name") || !strings.Contains(name.SortURL, "dir=desc") {
			t.Errorf("active link should flip to desc: %q", name.SortURL)
		}
	})

	t.Run("inactive sortable column requests asc, aria none", func(t *testing.T) {
		tbl := ui.NewTable(base, url.Values{}, cols(),
			ui.SortState{Key: "name", Dir: "asc"}, nil, ui.Pager{})
		cr := tbl.Header[2]
		if cr.Active {
			t.Error("cr column should not be Active")
		}
		if cr.AriaSort != "none" {
			t.Errorf("AriaSort = %q, want none", cr.AriaSort)
		}
		if !strings.Contains(cr.SortURL, "sort=cr") || !strings.Contains(cr.SortURL, "dir=asc") {
			t.Errorf("inactive link should request asc: %q", cr.SortURL)
		}
		if cr.Align != "end" {
			t.Errorf("Align = %q, want end (passthrough)", cr.Align)
		}
	})

	t.Run("non-sortable column has no link", func(t *testing.T) {
		tbl := ui.NewTable(base, url.Values{}, cols(),
			ui.SortState{Key: "name", Dir: "asc"}, nil, ui.Pager{})
		sys := tbl.Header[1]
		if sys.SortURL != "" {
			t.Errorf("display-only column should have no SortURL: %q", sys.SortURL)
		}
		if sys.AriaSort != "none" {
			t.Errorf("AriaSort = %q, want none", sys.AriaSort)
		}
	})

	t.Run("preserves filters and current page; does not mutate q", func(t *testing.T) {
		q := url.Values{"tag": {"undead"}, "system": {"osr"}, "page": {"4"}}
		tbl := ui.NewTable(base, q, cols(),
			ui.SortState{Key: "name", Dir: "desc"}, nil, ui.Pager{})
		u := tbl.Header[0].SortURL
		for _, want := range []string{"tag=undead", "system=osr", "page=4", "sort=name", "dir=asc"} {
			if !strings.Contains(u, want) {
				t.Errorf("header URL %q missing %q", u, want)
			}
		}
		if _, ok := q["sort"]; ok {
			t.Errorf("NewTable mutated the caller's query: %v", q)
		}
	})

	t.Run("align passthrough into body", func(t *testing.T) {
		tbl := ui.NewTable(base, url.Values{}, cols(), ui.SortState{}, nil, ui.Pager{})
		if tbl.Align[2] != "end" {
			t.Errorf("Align[2] = %q, want end", tbl.Align[2])
		}
	})
}

func TestTablePartial(t *testing.T) {
	tmpl := template.Must(template.New("t").ParseFS(ui.PartialsFS(), "*.gohtml"))
	render := func(tbl ui.Table) string {
		var b strings.Builder
		if err := tmpl.ExecuteTemplate(&b, "ui/table", tbl); err != nil {
			t.Fatalf("execute ui/table: %v", err)
		}
		return b.String()
	}

	rows := [][]template.HTML{
		{`<a href="/bestiary/goblin">Goblin</a>`, "OSR", "1"},
		{`<a href="/bestiary/orc">Orc</a>`, "OSR", "1"},
	}
	tbl := ui.NewTable("/bestiary/", url.Values{}, cols(),
		ui.SortState{Key: "name", Dir: "asc"},
		rows, ui.NumberedPager("/bestiary/", url.Values{}, 1, 50, 100))
	tbl.Caption = "Creatures"
	got := render(tbl)

	for _, want := range []string{
		`class="app-table"`,
		`<caption>Creatures</caption>`,
		`class="app-sort is-active"`, // active sort header
		`aria-sort="ascending"`,
		`<a href="/bestiary/goblin">Goblin</a>`, // trusted cell HTML passes through
		`class="app-cell-end"`,                  // align on the CR column
		`class="app-pager"`,                     // pager renders below
	} {
		if !strings.Contains(got, want) {
			t.Errorf("table markup missing %q\n%s", want, got)
		}
	}

	// Two columns are sortable (name, cr) so exactly two anchors carry app-sort;
	// the display-only System column renders a plain <th> with no sort link.
	if n := strings.Count(got, `class="app-sort`); n != 2 {
		t.Errorf(`app-sort anchor count = %d, want 2 (name, cr); System must be plain`, n)
	}
	if !strings.Contains(got, "System") {
		t.Errorf("display-only System header label missing:\n%s", got)
	}
}
