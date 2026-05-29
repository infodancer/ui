// Package markdown is the single audited Markdown→HTML pipeline for the
// infodancer/matthewjhunter web stack. It exists so that every consumer
// (osg, faq, mdedit, future blog/timeline) sanitizes through one policy
// with one test corpus, instead of each carrying a near-duplicate goldmark +
// bluemonday setup that drifts over time.
//
// Pipeline:
//
//  1. goldmark parses the Markdown. GFM extensions and raw-HTML
//     passthrough (goldmark's WithUnsafe) are opt-in via [Options].
//  2. bluemonday filters the resulting HTML against an allowlist. This is
//     the security boundary — it holds regardless of the goldmark
//     options, which is why the XSS corpus in markdown_test.go is run
//     against every preset, including the raw-HTML-enabled one.
//
// The contract: only the sanitized output of a [Renderer] may be cast to
// template.HTML for user-authored fields.
package markdown

import (
	"bytes"
	"html"
	"html/template"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// LinkPolicy selects how <a href> values are treated by the sanitizer.
type LinkPolicy int

const (
	// LinkDefault keeps the base allowlist's own link handling: for the
	// UGCPolicy base (Strict/Rich) that means http/https/mailto and
	// relative URLs with rel="nofollow". For a Restrictive base, no <a> is
	// allowed at all unless another policy value opts links back in.
	LinkDefault LinkPolicy = iota
	// LinkNone strips every <a> tag (link text is kept). With a Restrictive
	// base this is also the effect of LinkDefault, since that base allows
	// no links to begin with.
	LinkNone
	// LinkRelativeOnly permits relative / same-origin links (e.g. "/page",
	// "#frag") and strips every link that carries a scheme (http:, https:,
	// mailto:, javascript:) or is protocol-relative ("//host"). For
	// untrusted comments this blocks outbound spam while still letting
	// authors reference other pages on the site.
	LinkRelativeOnly
	// LinkSafeSchemes permits http/https/mailto and relative URLs, adds
	// rel="nofollow", and rejects every other scheme.
	LinkSafeSchemes
)

// LinkChecker is a per-link scrutiny/rewriting hook. When set on [Options], it
// is invoked for every surviving <a href> as the last step of rendering — after
// bluemonday and after the [LinkPolicy] gate — so it sees only already-sanitized
// hrefs. Use it for logic the coarse policy can't express: allowlisting
// specific external hosts, forcing https, routing links through an interstitial,
// or consulting a reputation service.
//
// It composes with Links rather than replacing it: the policy runs first, so a
// link a stricter policy already removed never reaches the checker. To make
// decisions about links a policy would drop (e.g. external links under the
// default relative-only comment policy), pair the checker with a more permissive
// LinkPolicy such as [LinkSafeSchemes] or [LinkDefault].
//
// The checker is consuming-application code and is trusted: a [LinkDecision.Href]
// it returns is used verbatim and is NOT re-sanitized, so a rewrite to a
// dangerous scheme is the caller's responsibility.
type LinkChecker func(href string) LinkDecision

// LinkDecision is what a [LinkChecker] returns for one link. The zero value
// (Allow false) strips the link, so a checker that forgets a case fails closed.
type LinkDecision struct {
	// Allow keeps the link. When false, the <a> tag is removed and its text
	// content is preserved.
	Allow bool
	// Href, when non-empty, replaces the link target (e.g. forced to https or
	// rewritten to an interstitial). Ignored when Allow is false.
	Href string
	// Rel, when non-empty, sets the link's rel attribute (e.g.
	// "nofollow noopener"). Ignored when Allow is false.
	Rel string
}

// Options configures a [Renderer]. The zero value is CommonMark parsed with
// no raw HTML, sanitized against bluemonday's UGCPolicy — i.e. [Strict].
//
// There are two allowlist bases. The default base is UGCPolicy: permissive,
// for authored content you mostly trust (headings, images, tables, links).
// Setting Restrictive instead builds the allowlist from nothing, permitting
// only the inline set plus the block constructs you explicitly enable — for
// untrusted short content like comments.
type Options struct {
	// --- goldmark parse extensions (all off by default = CommonMark) ---

	Tables        bool // GFM tables
	Strikethrough bool // GFM ~~deleted~~ → <del>
	Autolink      bool // GFM: turn bare URLs into links
	TaskList      bool // GFM task-list checkboxes

	// AllowRawHTML sets goldmark's WithUnsafe, letting raw HTML in the
	// source pass goldmark untouched so bluemonday can filter it. Enable
	// only when authors legitimately embed inline HTML (faq uses
	// kbd/sub/sup/mark); bluemonday remains the boundary either way.
	AllowRawHTML bool

	// --- sanitize allowlist ---

	// Restrictive builds the allowlist from an empty policy (only what is
	// explicitly permitted) instead of the permissive UGCPolicy. Use it for
	// untrusted content. Under a Restrictive base the always-on set is
	// inline only: p, br, strong, em, code (and del when Strikethrough is
	// set); the block fields below add to it.
	Restrictive bool

	// These add block constructs to a Restrictive base. They have no effect
	// on the UGCPolicy base, which already allows them.
	AllowLists       bool // ul, ol, li
	AllowBlockquotes bool // blockquote
	AllowCodeBlocks  bool // pre (fenced code blocks)

	// ExtraElements are tag names allowed in addition to the base
	// (UGCPolicy, or the Restrictive inline set). e.g. kbd, sub, sup, mark.
	ExtraElements []string

	// Directives registers inline directive renderers by name. When non-empty,
	// the :name[label] inline syntax is enabled for those names and each parsed
	// directive is rendered by its [DirectiveFunc]. The label is raw free-text
	// (it may carry punctuation Markdown would otherwise interpret); only
	// registered names activate, so other uses of ':' are left as text. The
	// emitted HTML is still sanitized by bluemonday — pair this with
	// PolicyCustomize to allowlist the markup the funcs produce. See
	// [DirectiveFunc].
	Directives map[string]DirectiveFunc

	// Extensions are extra goldmark extenders applied verbatim, after the
	// preset-derived ones — a general escape hatch for goldmark features the
	// presets don't expose (e.g. footnotes, definition lists). bluemonday
	// remains the boundary regardless of what they emit.
	Extensions []goldmark.Extender

	// PolicyCustomize, when non-nil, is applied to the bluemonday policy after
	// the base allowlist (and ExtraElements/Links) are built. It is the one
	// hook for allowlisting custom markup — typically the output of a
	// [DirectiveFunc] or an element an [Extensions] extender emits. It runs
	// inside the module so the security boundary stays in one audited place.
	PolicyCustomize func(*bluemonday.Policy)

	// Links selects href handling; see [LinkPolicy].
	Links LinkPolicy

	// LinkChecker, when non-nil, is a final per-link scrutiny/rewriting hook
	// applied to links that survive Links. See [LinkChecker]. nil disables it
	// (zero post-render cost).
	LinkChecker LinkChecker

	// StripEmptyDivs removes whitespace-only <div></div> blocks left in the
	// sanitized output. Useful for imported HTML; harmless on clean Markdown.
	StripEmptyDivs bool
}

// Strict returns options for plain authored Markdown: CommonMark, no raw
// HTML, UGCPolicy allowlist. Matches osg's historical render behavior.
func Strict() Options { return Options{} }

// Rich returns options for full authored content (faq's needs): GFM
// extensions, raw inline HTML passthrough (filtered by bluemonday), the
// kbd/sub/sup/mark elements faq relies on, and empty-div stripping.
func Rich() Options {
	return Options{
		Tables:         true,
		Strikethrough:  true,
		Autolink:       true,
		TaskList:       true,
		AllowRawHTML:   true,
		ExtraElements:  []string{"kbd", "sub", "sup", "mark"},
		StripEmptyDivs: true,
	}
}

// Comment returns options for untrusted short content: a restricted subset
// with inline emphasis, inline code, strikethrough, lists, blockquotes, and
// fenced code blocks — but no headings, images, tables, or raw HTML. Links
// are governed by linkPolicy (the comment use case uses [LinkRelativeOnly]
// to block outbound spam). No raw HTML, and bare URLs are not autolinked, so
// every link is an explicit one the policy can vet.
func Comment(linkPolicy LinkPolicy) Options {
	return Options{
		Strikethrough:    true,
		Restrictive:      true,
		AllowLists:       true,
		AllowBlockquotes: true,
		AllowCodeBlocks:  true,
		Links:            linkPolicy,
	}
}

// Renderer is an immutable, concurrent-safe Markdown renderer. Build one
// per distinct policy at startup with [New] and reuse it across requests.
type Renderer struct {
	md             goldmark.Markdown
	policy         *bluemonday.Policy
	stripEmptyDivs bool
	linkChecker    LinkChecker
}

// New builds a Renderer for the given options. The underlying goldmark and
// bluemonday objects are constructed once here and are safe for concurrent
// reads thereafter.
func New(opts Options) *Renderer {
	var exts []goldmark.Extender
	if opts.Tables {
		exts = append(exts, extension.Table)
	}
	if opts.Strikethrough {
		exts = append(exts, extension.Strikethrough)
	}
	if opts.Autolink {
		exts = append(exts, extension.Linkify)
	}
	if opts.TaskList {
		exts = append(exts, extension.TaskList)
	}
	if len(opts.Directives) > 0 {
		exts = append(exts, &directiveExtension{funcs: opts.Directives})
	}
	exts = append(exts, opts.Extensions...)

	var gmOpts []goldmark.Option
	if len(exts) > 0 {
		gmOpts = append(gmOpts, goldmark.WithExtensions(exts...))
	}
	if opts.AllowRawHTML {
		gmOpts = append(gmOpts, goldmark.WithRendererOptions(gmhtml.WithUnsafe()))
	}

	return &Renderer{
		md:             goldmark.New(gmOpts...),
		policy:         buildPolicy(opts),
		stripEmptyDivs: opts.StripEmptyDivs,
		linkChecker:    opts.LinkChecker,
	}
}

// buildPolicy assembles the bluemonday allowlist for opts. bluemonday is the
// security boundary; goldmark's own IsDangerousURL additionally rejects
// javascript:/vbscript:/data:text/html/file: link targets upstream.
func buildPolicy(opts Options) *bluemonday.Policy {
	var p *bluemonday.Policy
	if opts.Restrictive {
		// Built from nothing: only the inline set plus explicitly enabled
		// blocks. Headings, images, and tables are absent by construction.
		p = bluemonday.NewPolicy()
		p.AllowElements("p", "br", "strong", "em", "code")
		if opts.Strikethrough {
			p.AllowElements("del")
		}
		if opts.AllowLists {
			p.AllowElements("ul", "ol", "li")
		}
		if opts.AllowBlockquotes {
			p.AllowElements("blockquote")
		}
		if opts.AllowCodeBlocks {
			p.AllowElements("pre")
		}
	} else {
		// The canonical user-generated-content allowlist: permits the markup
		// goldmark emits and strips script/style/iframe/object/embed/form
		// plus every on* event-handler attribute.
		p = bluemonday.UGCPolicy()
	}
	if len(opts.ExtraElements) > 0 {
		p.AllowElements(opts.ExtraElements...)
	}
	applyLinkPolicy(p, opts.Links)
	if opts.PolicyCustomize != nil {
		opts.PolicyCustomize(p)
	}
	return p
}

// applyLinkPolicy configures href handling on p per lp. LinkDefault leaves the
// base policy untouched.
func applyLinkPolicy(p *bluemonday.Policy, lp LinkPolicy) {
	switch lp {
	case LinkDefault, LinkNone:
		// LinkDefault: keep the base as-is. LinkNone: rely on not allowing
		// <a> (the Restrictive base allows none; for the UGC base, callers
		// wanting LinkNone is out of the documented use cases).
	case LinkRelativeOnly:
		// bluemonday treats a protocol-relative "//host" target as a
		// relative URL and would let it through, so a scheme allowlist alone
		// is not enough to keep links same-origin. Constrain the href value
		// itself with a positive allow-pattern: fragment, query, a rooted
		// path that is not "//…", or a scheme-less relative path (no colon).
		// Go's RE2 has no negative lookahead, hence the explicit alternation.
		p.AllowAttrs("href").Matching(relativeURLPattern).OnElements("a")
		p.AllowRelativeURLs(true)
	case LinkSafeSchemes:
		p.AllowAttrs("href").OnElements("a")
		p.RequireParseableURLs(true)
		p.AllowRelativeURLs(true)
		p.AllowURLSchemes("http", "https", "mailto")
		p.RequireNoFollowOnLinks(true)
	}
}

// applyLinkChecker re-parses the already-sanitized fragment, runs check on
// every <a href>, and applies the decision: strip (unwrap, keeping link text),
// rewrite the href, and/or set rel. It operates on bluemonday's output, so it
// is an additional gate, never a replacement for sanitization.
func applyLinkChecker(fragment string, check LinkChecker) string {
	// Parse in a <div> context so the fragment is treated as flow content.
	ctx := &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div}
	nodes, err := xhtml.ParseFragment(strings.NewReader(fragment), ctx)
	if err != nil {
		// The input is well-formed sanitized HTML, so this is unexpected.
		// Return it unchanged: bluemonday has already gated it; failing to
		// run the (additive) checker must not corrupt the output.
		return fragment
	}
	for _, n := range nodes {
		processLinks(n, check)
	}
	var b strings.Builder
	for _, n := range nodes {
		if err := xhtml.Render(&b, n); err != nil {
			return fragment
		}
	}
	return b.String()
}

// processLinks walks n's descendants, applying check to each <a> that has an
// href. It iterates with a saved next pointer because unwrapping mutates the
// child list.
func processLinks(n *xhtml.Node, check LinkChecker) {
	for child := n.FirstChild; child != nil; {
		next := child.NextSibling
		if child.Type == xhtml.ElementNode && child.DataAtom == atom.A {
			processLinks(child, check) // descend first (defensive; links don't nest)
			if href, ok := getAttr(child, "href"); ok {
				switch d := check(href); {
				case !d.Allow:
					unwrap(n, child) // drop <a>, keep its text
				default:
					if d.Href != "" {
						setAttr(child, "href", d.Href)
					}
					if d.Rel != "" {
						setAttr(child, "rel", d.Rel)
					}
				}
			}
		} else {
			processLinks(child, check)
		}
		child = next
	}
}

func getAttr(n *xhtml.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

func setAttr(n *xhtml.Node, key, val string) {
	for i := range n.Attr {
		if n.Attr[i].Key == key {
			n.Attr[i].Val = val
			return
		}
	}
	n.Attr = append(n.Attr, xhtml.Attribute{Key: key, Val: val})
}

// unwrap replaces node with its children in parent, preserving link text.
func unwrap(parent, node *xhtml.Node) {
	for c := node.FirstChild; c != nil; {
		next := c.NextSibling
		node.RemoveChild(c)
		parent.InsertBefore(c, node)
		c = next
	}
	parent.RemoveChild(node)
}

// Render converts Markdown source to sanitized HTML safe to drop into a
// template. Empty or whitespace-only input returns "" so callers can guard
// with {{if}} on the original Markdown and skip rendering entirely.
func (r *Renderer) Render(md string) template.HTML {
	if strings.TrimSpace(md) == "" {
		return ""
	}
	var buf bytes.Buffer
	if err := r.md.Convert([]byte(md), &buf); err != nil {
		// goldmark.Convert only errors on writer failure; bytes.Buffer
		// never fails to write. Fall back to escaped plain text so a
		// programming error here can never leak raw Markdown as HTML.
		return template.HTML(html.EscapeString(md)) //nolint:gosec // escaped above
	}
	clean := r.policy.SanitizeBytes(buf.Bytes())
	out := string(clean)
	if r.stripEmptyDivs {
		out = stripEmptyDivs(out)
	}
	if r.linkChecker != nil {
		out = applyLinkChecker(out, r.linkChecker)
	}
	return template.HTML(out) //nolint:gosec // sanitized by bluemonday above
}

// RenderString is [Renderer.Render] for callers that store the result as a
// plain string (e.g. a body_html database column) rather than template.HTML.
func (r *Renderer) RenderString(md string) string {
	return string(r.Render(md))
}

// relativeURLPattern matches same-origin href values for [LinkRelativeOnly]:
// a fragment (#…), a query (?…), a rooted path that is not protocol-relative
// (/… but not //…), or a scheme-less relative path (first char is not a
// slash/colon/query/hash, and the value contains no colon). Any scheme-bearing
// or protocol-relative URL fails to match and is dropped.
var relativeURLPattern = regexp.MustCompile(`^(?:#.*|\?.*|/(?:[^/].*)?|[^/:?#][^:]*)$`)

// emptyDivPattern matches a <div> with optional attributes containing only
// whitespace. The [^>]* in the open tag and the lack of nested-tag handling
// is adequate because bluemonday guarantees well-formed output; stripEmptyDivs
// loops to collapse nesting.
var emptyDivPattern = regexp.MustCompile(`<div\b[^>]*>\s*</div>`)

func stripEmptyDivs(s string) string {
	for {
		next := emptyDivPattern.ReplaceAllString(s, "")
		if next == s {
			return s
		}
		s = next
	}
}
