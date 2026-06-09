package ui

import (
	"fmt"
	"net/url"
	"strconv"
)

// Pager backs the ui/pager partial: a prev/next navigation for paged lists.
// URLs are precomputed (empty = that side is omitted), so the partial stays
// logic-free. Build a prev/next pager with [NewPager] (offset-based, no count),
// or a numbered jump-to-page pager with [NumberedPager] (1-based, needs a
// count). Or set the fields directly.
type Pager struct {
	// PrevURL links to the previous page; "" omits the prev affordance.
	PrevURL string
	// NextURL links to the next page; "" omits the next affordance.
	NextURL string
	// Position is an optional human label for the current span, e.g. "51–100"
	// (offset pager) or "Page 7 of 23" (numbered pager); "" omits it.
	Position string
	// Pages, when non-empty, renders a windowed numbered jump-to-page strip in
	// addition to prev/next (1 … 5 6 [7] 8 9 … 23). Empty means prev/next only,
	// which costs no COUNT(*). Populated by [NumberedPager].
	Pages []PageLink
}

// PageLink is one cell of a numbered pager strip. A URL of "" marks a non-link
// cell: the current page (Current true) or an ellipsis gap (Current false,
// Label "…").
type PageLink struct {
	Label   string
	URL     string
	Current bool
}

// NewPager builds a prev/next [Pager] for offset paging. base is the path the
// page links to (e.g. "/campaign/"); q is the current request query, whose
// filters are preserved on every page link with ?offset= set (and omitted at 0
// for a clean first-page URL). pageSize is the page length; itemsThisPage (the
// number of rows actually on this page) drives Position; hasNext is whether a
// further page exists (typically from a limit+1 probe). The caller's q is not
// mutated.
func NewPager(base string, q url.Values, offset, pageSize, itemsThisPage int, hasNext bool) Pager {
	var p Pager
	if offset > 0 {
		prev := offset - pageSize
		if prev < 0 {
			prev = 0
		}
		p.PrevURL = pageURL(base, q, prev)
	}
	if hasNext {
		p.NextURL = pageURL(base, q, offset+pageSize)
	}
	if itemsThisPage > 0 {
		p.Position = fmt.Sprintf("%d–%d", offset+1, offset+itemsThisPage)
	}
	return p
}

// NumberedPager builds a [Pager] with prev/next AND a windowed numbered
// jump-to-page strip. page is 1-based; total is the COUNT(*) the caller
// supplies (the numbered strip is what makes the count worth paying for). q's
// filters are preserved on every link, with the page written as ?page=N and
// omitted at page 1 for a clean first-page URL. The caller's q is not mutated.
//
// The window keeps the first and last page plus the current page +/- 2, with a
// single ellipsis bridging each gap: page 7 of 23 -> 1 … 5 6 [7] 8 9 … 23. A
// single page yields no strip (Pages is nil); a consumer can then omit the
// pager entirely.
func NumberedPager(base string, q url.Values, page, pageSize, total int) Pager {
	if pageSize < 1 {
		pageSize = 1
	}
	pageCount := (total + pageSize - 1) / pageSize
	if pageCount < 1 {
		pageCount = 1
	}
	if page < 1 {
		page = 1
	}
	if page > pageCount {
		page = pageCount
	}

	var p Pager
	if page > 1 {
		p.PrevURL = pageNumURL(base, q, page-1)
	}
	if page < pageCount {
		p.NextURL = pageNumURL(base, q, page+1)
	}
	p.Position = fmt.Sprintf("Page %d of %d", page, pageCount)

	// A lone page needs no numbered strip.
	if pageCount <= 1 {
		return p
	}

	// The visible set: first, last, and a radius-2 window around the current
	// page (clamped). Walk 1..pageCount in order, emitting a link for each
	// member and one ellipsis whenever consecutive members leave a numeric gap.
	const radius = 2
	visible := func(n int) bool {
		return n == 1 || n == pageCount || (n >= page-radius && n <= page+radius)
	}
	prevEmitted := 0
	for n := 1; n <= pageCount; n++ {
		if !visible(n) {
			continue
		}
		if prevEmitted != 0 && n-prevEmitted > 1 {
			p.Pages = append(p.Pages, PageLink{Label: "…"})
		}
		link := PageLink{Label: strconv.Itoa(n)}
		if n == page {
			link.Current = true
		} else {
			link.URL = pageNumURL(base, q, n)
		}
		p.Pages = append(p.Pages, link)
		prevEmitted = n
	}
	return p
}

// pageURL clones q, sets (or drops, at 0) the offset, and joins it to base.
func pageURL(base string, q url.Values, offset int) string {
	nq := cloneValues(q)
	if offset > 0 {
		nq.Set("offset", strconv.Itoa(offset))
	} else {
		nq.Del("offset")
	}
	return joinQuery(base, nq)
}

// pageNumURL clones q, sets (or drops, at page 1) the 1-based page, and joins
// it to base. Page 1 omits ?page= for a clean first-page URL.
func pageNumURL(base string, q url.Values, page int) string {
	nq := cloneValues(q)
	if page > 1 {
		nq.Set("page", strconv.Itoa(page))
	} else {
		nq.Del("page")
	}
	return joinQuery(base, nq)
}

// cloneValues returns a deep copy of q, so a builder can Set/Del without
// mutating the caller's request query. Shared by every URL-building helper
// here and in table.go.
func cloneValues(q url.Values) url.Values {
	nq := make(url.Values, len(q))
	for k, vs := range q {
		nq[k] = append([]string(nil), vs...)
	}
	return nq
}

// joinQuery appends the encoded query to base, dropping the "?" when empty.
func joinQuery(base string, q url.Values) string {
	if enc := q.Encode(); enc != "" {
		return base + "?" + enc
	}
	return base
}
