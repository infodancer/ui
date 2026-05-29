package markdown

import (
	"strings"
	"testing"
)

// The XSS corpus is the security contract. Every entry must be neutralized
// under BOTH presets — and especially under Rich, which enables goldmark's
// WithUnsafe (raw HTML passes goldmark untouched, so bluemonday is the only
// thing standing between author input and the rendered page). If a payload
// survives, the sanitizer has a hole.
var xssCorpus = []struct {
	name      string
	in        string
	mustNotIn []string // substrings that must be absent from rendered output
}{
	// Only the executable <script> tag must go; goldmark may leave the
	// inert inner text as escaped plain text, which is harmless.
	{"script tag", "hello <script>alert(1)</script> world", []string{"<script"}},
	{"img onerror", `![x](http://e.x/i.png) <img src=x onerror="alert(1)">`, []string{"onerror", "alert(1)"}},
	{"anchor onclick", `<a href="/" onclick="alert(1)">x</a>`, []string{"onclick", "alert(1)"}},
	{"javascript URI link", "[click](javascript:alert(1))", []string{"javascript:"}},
	{"data text/html URI", "[click](data:text/html;base64,PHNjcmlwdD4=)", []string{"data:text/html"}},
	{"vbscript URI", "[click](vbscript:msgbox(1))", []string{"vbscript:"}},
	{"iframe", `<iframe src="https://evil.example"></iframe>`, []string{"<iframe"}},
	{"object", `<object data="evil.swf"></object>`, []string{"<object"}},
	{"embed", `<embed src="evil.swf">`, []string{"<embed"}},
	{"form", `<form action="/steal"><input name="p"></form>`, []string{"<form", "<input"}},
	{"style tag", `<style>body{display:none}</style>`, []string{"<style"}},
	{"svg onload", `<svg onload="alert(1)"></svg>`, []string{"onload", "alert(1)"}},
	{"meta refresh", `<meta http-equiv="refresh" content="0;url=evil">`, []string{"<meta"}},
}

func TestPresets_XSSCorpusNeutralized(t *testing.T) {
	presets := map[string]*Renderer{
		"Strict":  New(Strict()),
		"Rich":    New(Rich()),
		"Comment": New(Comment(LinkRelativeOnly)),
	}
	for name, r := range presets {
		for _, tc := range xssCorpus {
			t.Run(name+"/"+tc.name, func(t *testing.T) {
				got := r.RenderString(tc.in)
				for _, bad := range tc.mustNotIn {
					if strings.Contains(strings.ToLower(got), strings.ToLower(bad)) {
						t.Errorf("payload survived: %q present in output\ninput:  %s\noutput: %s", bad, tc.in, got)
					}
				}
			})
		}
	}
}

func TestRender_PreservesSafeMarkdown(t *testing.T) {
	r := New(Strict())
	got := r.RenderString("# Title\n\nSome **bold** and *italic* and `code` and a [link](https://example.com).\n\n- one\n- two")
	for _, want := range []string{"<h1", "Title", "<strong>bold", "<em>italic", "<code>code", `href="https://example.com"`, "<li>one"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestRender_EmptyInput(t *testing.T) {
	r := New(Strict())
	for _, in := range []string{"", "   ", "\n\t\n"} {
		if got := r.RenderString(in); got != "" {
			t.Errorf("empty/whitespace input %q should render \"\", got %q", in, got)
		}
	}
}

func TestRich_EnablesGFMAndInlineHTML(t *testing.T) {
	r := New(Rich())

	table := r.RenderString("| a | b |\n|---|---|\n| 1 | 2 |")
	if !strings.Contains(table, "<table") {
		t.Errorf("Rich should render GFM tables, got:\n%s", table)
	}

	strike := r.RenderString("~~gone~~")
	if !strings.Contains(strike, "<del>") {
		t.Errorf("Rich should render GFM strikethrough, got:\n%s", strike)
	}

	// kbd/sub/sup/mark are the inline tags faq content relies on; Rich
	// allows raw HTML so an edit round-trips them instead of stripping.
	kbd := r.RenderString("Press <kbd>Ctrl</kbd>")
	if !strings.Contains(kbd, "<kbd>Ctrl</kbd>") {
		t.Errorf("Rich should preserve <kbd>, got:\n%s", kbd)
	}
}

func TestStrict_DropsInlineHTML(t *testing.T) {
	r := New(Strict())
	// Strict does not set WithUnsafe: goldmark elides raw HTML blocks
	// rather than passing them to the sanitizer. The <kbd> tag must not
	// appear as a live element in the output.
	got := r.RenderString("Press <kbd>Ctrl</kbd> now")
	if strings.Contains(got, "<kbd>") {
		t.Errorf("Strict should not emit raw inline HTML tags, got:\n%s", got)
	}
}

func TestRich_StripsEmptyDivs(t *testing.T) {
	r := New(Rich())
	got := r.RenderString("text\n\n<div>  </div>\n\n<div><div></div></div>")
	if strings.Contains(got, "<div>") {
		t.Errorf("Rich should strip empty divs, got:\n%s", got)
	}
}

func TestExtraElements_AreAllowlisted(t *testing.T) {
	r := New(Options{AllowRawHTML: true, ExtraElements: []string{"abbr"}})
	got := r.RenderString(`an <abbr title="x">A</abbr>`)
	if !strings.Contains(got, "<abbr") {
		t.Errorf("configured extra element <abbr> should survive, got:\n%s", got)
	}
}

func TestComment_AllowsLightweightBlocks(t *testing.T) {
	r := New(Comment(LinkRelativeOnly))
	in := "**bold** *italic* ~~strike~~ `code`\n\n- one\n- two\n\n> quoted\n\n```\nfenced\n```"
	got := r.RenderString(in)
	for _, want := range []string{"<strong>bold", "<em>italic", "<del>strike", "<code>code", "<ul>", "<li>one", "<blockquote>", "<pre>"} {
		if !strings.Contains(got, want) {
			t.Errorf("comment should allow %q, got:\n%s", want, got)
		}
	}
}

func TestComment_DeniesHeadingsImagesTablesRawHTML(t *testing.T) {
	r := New(Comment(LinkRelativeOnly))
	cases := []struct {
		name, in, mustNotContain string
	}{
		{"heading", "# Big Heading", "<h1"},
		{"image", "![alt](/pic.png)", "<img"},
		{"table", "| a | b |\n|---|---|\n| 1 | 2 |", "<table"},
		{"raw inline html", "Press <kbd>Ctrl</kbd>", "<kbd>"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := r.RenderString(tc.in); strings.Contains(got, tc.mustNotContain) {
				t.Errorf("comment should not emit %q, got:\n%s", tc.mustNotContain, got)
			}
		})
	}
}

func TestComment_RelativeOnlyLinks(t *testing.T) {
	r := New(Comment(LinkRelativeOnly))
	cases := []struct {
		name, in string
		wantHref bool // whether an <a href=...> should survive
	}{
		{"relative path", "[x](/internal/page)", true},
		{"fragment", "[x](#section)", true},
		{"absolute https", "[x](https://evil.example/spam)", false},
		{"mailto", "[x](mailto:a@b.com)", false},
		{"protocol-relative", "[x](//evil.example/spam)", false},
		{"javascript", "[x](javascript:alert(1))", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := r.RenderString(tc.in)
			hasHref := strings.Contains(got, "href=")
			if hasHref != tc.wantHref {
				t.Errorf("link %q: got href=%v want %v\noutput: %s", tc.in, hasHref, tc.wantHref, got)
			}
			// The visible link text must survive regardless.
			if !strings.Contains(got, ">x<") && !strings.Contains(got, "x</a>") && !strings.Contains(got, "x") {
				t.Errorf("link text dropped for %q: %s", tc.in, got)
			}
			// No outbound host should ever appear in an href.
			if strings.Contains(got, "evil.example") && hasHref {
				t.Errorf("outbound host leaked into a live link for %q: %s", tc.in, got)
			}
		})
	}
}

func TestComment_RelativeLinkNoFlavorOfOutbound(t *testing.T) {
	// A relative link survives with its href intact.
	r := New(Comment(LinkRelativeOnly))
	got := r.RenderString("see [the rules](/rules)")
	if !strings.Contains(got, `href="/rules"`) {
		t.Errorf("relative href should be preserved, got: %s", got)
	}
}

func TestLinkChecker_AllowlistsExternalHosts(t *testing.T) {
	// Permissive scheme policy, then a checker that allows only one host.
	r := New(Options{
		Links: LinkSafeSchemes,
		LinkChecker: func(href string) LinkDecision {
			if strings.HasPrefix(href, "https://trusted.example/") {
				return LinkDecision{Allow: true}
			}
			return LinkDecision{Allow: false}
		},
	})
	got := r.RenderString("[ok](https://trusted.example/page) and [no](https://spam.example/x)")
	if !strings.Contains(got, `href="https://trusted.example/page"`) {
		t.Errorf("allowlisted host should survive, got:\n%s", got)
	}
	if strings.Contains(got, "spam.example") && strings.Contains(got, "href=") {
		t.Errorf("non-allowlisted host should be stripped to text, got:\n%s", got)
	}
	if !strings.Contains(got, "no") {
		t.Errorf("stripped link must keep its text, got:\n%s", got)
	}
}

func TestLinkChecker_RewritesHrefAndRel(t *testing.T) {
	r := New(Options{
		Links: LinkSafeSchemes,
		LinkChecker: func(href string) LinkDecision {
			return LinkDecision{Allow: true, Href: "/redirect?to=" + href, Rel: "nofollow noopener"}
		},
	})
	got := r.RenderString("[x](https://example.com/p)")
	if !strings.Contains(got, `href="/redirect?to=https://example.com/p"`) {
		t.Errorf("href rewrite not applied, got:\n%s", got)
	}
	if !strings.Contains(got, `rel="nofollow noopener"`) {
		t.Errorf("rel not set, got:\n%s", got)
	}
}

func TestLinkChecker_RunsAfterPolicy(t *testing.T) {
	// Under relative-only, an outbound link is gone before the checker runs,
	// so the checker is never asked about it. The checker can still further
	// restrict the relative links that survive.
	var sawHrefs []string
	r := New(Options{
		Restrictive: true,
		Links:       LinkRelativeOnly,
		LinkChecker: func(href string) LinkDecision {
			sawHrefs = append(sawHrefs, href)
			return LinkDecision{Allow: href == "/keep"} // strip /drop
		},
	})
	got := r.RenderString("[a](/keep) [b](/drop) [c](https://evil.example)")
	if !strings.Contains(got, `href="/keep"`) {
		t.Errorf("/keep should survive, got:\n%s", got)
	}
	if strings.Contains(got, `href="/drop"`) {
		t.Errorf("/drop should be stripped by the checker, got:\n%s", got)
	}
	for _, h := range sawHrefs {
		if strings.Contains(h, "evil.example") {
			t.Errorf("checker should never see the policy-stripped outbound link, saw %q", h)
		}
	}
}

func TestLinkChecker_NotInvokedWhenNil(t *testing.T) {
	// Sanity: a nil checker leaves the relative-only behavior unchanged.
	r := New(Comment(LinkRelativeOnly))
	got := r.RenderString("[x](/page)")
	if !strings.Contains(got, `href="/page"`) {
		t.Errorf("nil checker should not alter output, got:\n%s", got)
	}
}

func TestRenderer_ConcurrentSafe(t *testing.T) {
	r := New(Rich())
	const n = 50
	done := make(chan string, n)
	for i := 0; i < n; i++ {
		go func() { done <- r.RenderString("# h\n\n**b** <script>x</script>") }()
	}
	for i := 0; i < n; i++ {
		if out := <-done; strings.Contains(out, "<script") {
			t.Fatal("concurrent render leaked a script tag")
		}
	}
}
