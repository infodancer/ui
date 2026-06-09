package ui

import (
	"html/template"
	"net/url"
)

// Column describes one table column for [NewTable]. Key is the sort key sent as
// ?sort=; "" marks the column non-sortable (a plain <th> with no link). Label
// is the header text. Align is "", "end", or "center" and maps to a CSS
// alignment class applied to the header and the matching body cells.
type Column struct {
	Key   string
	Label string
	Align string
}

// SortState is a table's active sort: Key matches a [Column].Key and Dir is
// "asc" or "desc". The zero value (both "") means no column is sorted.
type SortState struct {
	Key string
	Dir string
}

// HeaderCell is one precomputed <th> produced by [NewTable]: a plain label when
// SortURL is "", or a sort link otherwise. The partial only prints these, so it
// stays free of URL logic (which html/template cannot do).
type HeaderCell struct {
	Label string
	// SortURL is the link the header points at; "" renders a non-sortable
	// plain <th>. The link flips direction on the active column and sorts
	// ascending on any other.
	SortURL string
	// AriaSort is the th's aria-sort value: "ascending", "descending", or
	// "none".
	AriaSort string
	// Active marks the currently-sorted column (gets the .is-active class).
	Active bool
	// Align mirrors the column's alignment ("", "end", "center").
	Align string
}

// Table backs the ui/table partial: a read-only sortable, paginated data table.
// Build it with [NewTable] so the header links are precomputed and the partial
// stays logic-free. Rows are caller-rendered trusted template.HTML cells (the
// same trusted-body model as [RenderDocument]); each row should carry
// len(columns) cells.
type Table struct {
	// Header is one precomputed cell per column.
	Header []HeaderCell
	// Rows are the body cells, one inner slice per row, each cell trusted HTML.
	Rows [][]template.HTML
	// Align is the per-column alignment class ("", "end", "center"), applied to
	// each body <td> by index so columns line up with their headers.
	Align []string
	// Pager renders below the table; use a prev/next [NewPager] or a numbered
	// [NumberedPager]. The zero value renders an empty nav.
	Pager Pager
	// Caption is an optional <caption> for accessibility; "" omits it.
	Caption string
}

// NewTable precomputes a [Table]'s header cells from the columns and the active
// sort. base is the page path header links point at; q is the current request
// query, whose filters (and current page/offset) are preserved on every header
// link with ?sort= and ?dir= set. sort is the active column + direction. rows
// and pager are passed through unchanged. The caller's q is not mutated.
//
// Clicking the active column's header flips its direction; clicking any other
// sortable header sorts that column ascending. A column with an empty Key is
// not sortable and renders a plain header.
func NewTable(base string, q url.Values, cols []Column, sort SortState, rows [][]template.HTML, pager Pager) Table {
	t := Table{
		Header: make([]HeaderCell, len(cols)),
		Align:  make([]string, len(cols)),
		Rows:   rows,
		Pager:  pager,
	}
	for i, c := range cols {
		t.Align[i] = c.Align
		hc := HeaderCell{Label: c.Label, Align: c.Align, AriaSort: "none"}
		if c.Key != "" {
			if c.Key == sort.Key {
				hc.Active = true
				next := "asc"
				if sort.Dir == "asc" {
					next = "desc"
					hc.AriaSort = "ascending"
				} else {
					hc.AriaSort = "descending"
				}
				hc.SortURL = sortURL(base, q, c.Key, next)
			} else {
				hc.SortURL = sortURL(base, q, c.Key, "asc")
			}
		}
		t.Header[i] = hc
	}
	return t
}

// sortURL clones q, sets sort=key and dir, and joins to base, preserving every
// other param (filters, page/offset) untouched. It deliberately leaves the
// page param alone: a header click keeps you on the current page; a consumer
// that wants header clicks to jump to page 1 strips the page param from q first.
func sortURL(base string, q url.Values, key, dir string) string {
	nq := cloneValues(q)
	nq.Set("sort", key)
	nq.Set("dir", dir)
	return joinQuery(base, nq)
}
