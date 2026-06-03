package ui

import (
	"fmt"
	"net/url"
	"strconv"
)

// Pager backs the ui/pager partial: a prev/next navigation for offset-based
// paging. URLs are precomputed (empty = that side is omitted), so the partial
// stays logic-free. Build one with [NewPager], or set the fields directly.
//
// ui#32 (the sortable table) will add numbered jump-to-page on top of this as
// an additive field; prev/next is the whole of the Tier-1 pager.
type Pager struct {
	// PrevURL links to the previous page; "" omits the prev affordance.
	PrevURL string
	// NextURL links to the next page; "" omits the next affordance.
	NextURL string
	// Position is an optional human label for the current span, e.g. "51–100";
	// "" omits it.
	Position string
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

// pageURL clones q, sets (or drops, at 0) the offset, and joins it to base.
func pageURL(base string, q url.Values, offset int) string {
	nq := make(url.Values, len(q))
	for k, vs := range q {
		nq[k] = append([]string(nil), vs...)
	}
	if offset > 0 {
		nq.Set("offset", strconv.Itoa(offset))
	} else {
		nq.Del("offset")
	}
	if enc := nq.Encode(); enc != "" {
		return base + "?" + enc
	}
	return base
}
