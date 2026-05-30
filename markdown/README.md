# markdown

The single audited Markdown → sanitized-HTML pipeline for the infodancer /
matthewjhunter web stack. One goldmark + bluemonday policy, one XSS test
corpus, reused by every consumer (faq, osg, mdedit, blog) instead of each
carrying a near-duplicate setup that drifts.

A nested module in the [infodancer/ui](../README.md) monorepo, versioned and
imported on its own path:

```
import "github.com/infodancer/ui/markdown"
```

```go
r := markdown.New(markdown.Rich())
html := r.Render(userMarkdown) // template.HTML, sanitized
```

## Why a shared module

A Markdown sanitizer is a security boundary. When several services each keep
their own copy of the goldmark configuration and bluemonday policy, those
copies drift — and a drifted allowlist is exactly where an XSS slips in. This
module is the one place that boundary is defined and tested, so every consumer
sanitizes identically.

## Presets

| Preset | goldmark | allowlist | use |
|--------|----------|-----------|-----|
| `Strict()` | CommonMark, no raw HTML | UGCPolicy | plain Markdown fields (osg's behavior) |
| `Rich()` | GFM, raw inline HTML | UGCPolicy + kbd/sub/sup/mark, empty-div stripping | full content (faq's behavior) |
| `Comment(linkPolicy)` | inline + strikethrough, no raw HTML | restrictive: emphasis, inline code, lists, blockquotes, code blocks — no headings/images/tables | untrusted comments |

`Comment` takes a `LinkPolicy`. `LinkRelativeOnly` keeps relative/same-origin
links and strips every scheme-bearing or protocol-relative (`//host`) URL —
the right default for public comments. Other policies: `LinkNone`,
`LinkSafeSchemes` (http/https/mailto + nofollow), `LinkDefault`.

Or build your own with `Options`. Whatever the goldmark settings, **bluemonday
is the boundary** — raw-HTML passthrough still strips script/style/iframe/
object/embed/form and every `on*` handler, and goldmark rejects
`javascript:`/`vbscript:`/`data:text/html`/`file:` URLs. The XSS corpus in
`markdown_test.go` runs against every preset, including the raw-HTML one.

## Custom link scrutiny

`LinkPolicy` is a coarse gate. For logic it can't express — allowlisting
specific external hosts, forcing https, routing through an interstitial,
consulting a reputation service — set `Options.LinkChecker`:

```go
r := markdown.New(markdown.Options{
    Links: markdown.LinkSafeSchemes, // coarse pre-filter to http/https/mailto
    LinkChecker: func(href string) markdown.LinkDecision {
        if allowedHost(href) {
            return markdown.LinkDecision{Allow: true, Rel: "nofollow noopener"}
        }
        return markdown.LinkDecision{Allow: false} // strip; link text is kept
    },
})
```

The checker runs as the last render step, after bluemonday and after the
`LinkPolicy` gate, so it only sees already-sanitized hrefs. It **composes**
with `Links`: a link a stricter policy already removed never reaches the
checker, so to scrutinize links the policy would drop, pair the checker with a
permissive policy (`LinkSafeSchemes` or `LinkDefault`). The decision can strip
the link (keeping its text), rewrite the href, and/or set `rel`. The zero
`LinkDecision` strips, so a checker that misses a case fails closed.

The checker is *your* code and is trusted: a `Href` it returns is used verbatim
and is not re-sanitized.

## Inline directives

A consumer can render a custom inline construct from Markdown without forking
the pipeline. Register a `DirectiveFunc` per name; the `:name[label]` syntax is
then recognized, where `label` is **raw free-text** (it may contain `:`, `=`,
`+`, parentheses, quotes — punctuation Markdown would otherwise interpret).
Only registered names activate, so ordinary colons (`3:1`, `http://…`) and
unregistered `:foo[…]` pass through as text.

```go
r := markdown.New(markdown.Options{
    Directives: map[string]markdown.DirectiveFunc{
        "roll": func(label string) template.HTML {
            esc := template.HTMLEscapeString(label)
            return template.HTML(`<span class="roll" title="` + esc + `">🎲</span>`)
        },
    },
    // Allowlist the markup the directive emits — see below.
    PolicyCustomize: func(p *bluemonday.Policy) {
        p.AllowAttrs("class", "title").OnElements("span")
    },
})
r.Render(":roll[1d20+3 >= 11 succeeded with 14]")
```

The label is carried verbatim (with `\]` and `\\` unescaped) and is never
parsed as Markdown. A directive node is a leaf, so it renders inline within the
surrounding paragraph. Block/container directives (`:::name … :::`) are not
implemented yet — they'll be added when a consumer needs them.

**The output is still sanitized.** A `DirectiveFunc`'s HTML is written into the
document and then filtered by bluemonday like everything else, so it is
fail-safe: without a matching `PolicyCustomize` allowlist entry, the directive's
distinguishing markup (e.g. `class`, `title`) is stripped. The func should also
escape `label` itself — bluemonday neutralizes a script payload regardless, but
a forgotten escape can still mangle output.

## Custom goldmark extensions and policy

`Options.Extensions []goldmark.Extender` is a general escape hatch for goldmark
features the presets don't expose (footnotes, definition lists, …). They run
after the preset-derived extensions; bluemonday remains the boundary regardless
of what they emit.

`Options.PolicyCustomize func(*bluemonday.Policy)` runs inside the module after
the base allowlist is built — the one hook for allowlisting custom markup (a
directive's output, or an element an extender emits). Keeping it inside the
module is deliberate: the security boundary stays in one audited place.

## Contract

Only the sanitized output of a `Renderer` may be cast to `template.HTML` for
user-authored fields. A `Renderer` is immutable and safe for concurrent use;
build one per policy at startup and reuse it.

## Status

Pre-1.0 (tag `markdown/v0.2.x`). `Options`, the presets, and `Renderer` are the
public surface. Consumers pin to a tag until v1.0. Versioned independently of
the `ui` root and `mdedit` modules in this repo.

## License

Apache-2.0. See [the repository LICENSE](../LICENSE).
