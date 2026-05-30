# CLAUDE.md

Guidance for Claude Code sessions in the `markdown` module
(`github.com/infodancer/ui/markdown`).

## What this is

The shared, audited Markdown → sanitized-HTML pipeline (goldmark + bluemonday)
for the infodancer / matthewjhunter web portfolio. A library-only module: no
`cmd/`, no JS, no templates. A **nested module** in the
[infodancer/ui](../CLAUDE.md) monorepo with its own `go.mod` and tag
(`markdown/v0.2.x`). Consumers: faq, osg, `github.com/infodancer/ui/mdedit`
(the editor component), and `infodancer/blog`. This module exists so the
sanitization boundary lives in one tested place instead of drifting across
copies.

## What NOT to change without explicit approval

- **The security boundary.** bluemonday is the boundary regardless of the
  goldmark options. Every change must keep the XSS corpus in
  `markdown_test.go` green; new payloads are *added* to the corpus, never
  removed, and the corpus runs against every preset (including the raw-HTML
  `Rich` path).
- **Public API.** `Options`, `Strict()`, `Rich()`, `Comment()`, `New`,
  `Renderer`, `Render`, `RenderString`, `LinkPolicy`, `LinkChecker`,
  `LinkDecision`, `DirectiveFunc`. Renaming or removing is breaking; adding
  `Options` fields, presets, or `LinkPolicy` values is not.
- **The directive sanitization invariant.** A `DirectiveFunc`'s output is *not*
  trusted: it is written into the document and then sanitized by bluemonday
  like all other output. `PolicyCustomize` is how a consumer allowlists that
  markup. Never special-case directive output to bypass the policy.
- **The `LinkChecker` invariant.** It is an *additive* gate that runs after
  bluemonday on already-sanitized links — it must never become the place
  sanitization happens. It may strip/rewrite/annotate links; it does not
  re-open the element or attribute allowlist.

## Conventions

- License: Apache-2.0.
- Go: track the latest patched release per the infodancer org standard.
- TDD: the sanitizer is driven by its XSS corpus. Write the payload test first,
  then the policy change.
- A `Renderer` is immutable and concurrent-safe; build once, reuse. Don't add
  per-call allocation of goldmark/bluemonday objects.

## Build (nested module in a monorepo)

The repo `go.work` shadows the parent infodancer workspace so in-tree builds of
the nested modules resolve. Run module commands from this directory; CI fans
out over the module matrix with `GOWORK=off`.

## Workflow

Per the org standard: GitHub issue before branching (`feature/<n>` or
`bug/<n>`), commits reference the issue, PRs merge to main.

## Presets

- `Strict()` — plain authored Markdown (osg's behavior).
- `Rich()` — full authored content (faq's behavior).
- `Comment(linkPolicy)` — constrained subset for untrusted short content
  (inline + lists/blockquotes/code blocks; no headings/images/tables/raw HTML).
  Built on a Restrictive (empty-base) policy. The comment use case uses
  `LinkRelativeOnly`, which strips scheme-bearing and protocol-relative URLs;
  `relativeURLPattern` guards the protocol-relative `//host` edge that
  bluemonday's relative-URL handling otherwise lets through.

## Roadmap notes

- The faq and osg migrations off their private goldmark+bluemonday copies onto
  this module — the reason it exists — are **done** (faq first, then osg).
- Inline directives (`:name[label]`, v0.1.0) cover osg's `roll` annotation. The
  block/container form (`:::name … :::`) is intentionally deferred until a real
  consumer (faq/blog callouts, figures) needs it; add it as an additive sibling
  to the inline parser, not a rewrite.
- **Image `src` policy** — add an `ImageRelativeOnly`-style policy (relative
  default, external opt-in), mirroring `LinkRelativeOnly`, with its own XSS
  corpus cases. Today UGCPolicy lets external `<img src>` through. This is the
  markdown-module half of the mdedit image work; see
  [infodancer/ui#14](https://github.com/infodancer/ui/issues/14).
