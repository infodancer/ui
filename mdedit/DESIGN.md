# mdedit — design (v0.1)

> Originally designed as the standalone `github.com/infodancer/mdedit`. Since
> 2026-05-29 it is a nested module in the `infodancer/ui` monorepo
> (`github.com/infodancer/ui/mdedit`), keeping its own `go.mod` and tag. The
> "Placement" decision below records that history.

## Problem

The Go + htmx consumers in the portfolio need a friendly way for
non-technical authors to edit Markdown content, without dragging a JavaScript
framework into the stack. Today osg and faq each carry their own
goldmark + bluemonday setup, and those two policies have already diverged
(faq runs GFM + raw-HTML passthrough; osg runs plain CommonMark) — a
sanitization policy that drifts is a security liability.

## Scope

In scope:

- One audited Markdown → sanitized-HTML pipeline, reused by every consumer.
- A display ↔ edit ↔ (preview) ↔ save UI loop, htmx-native.
- A friendly client editor for non-technical authors, behind a seam so the
  editor choice is swappable.

Out of scope (host owns these): storage, schema, authentication,
authorization, CSRF. Also out of scope: a Hugo variant — editing requires a
backend to store the result, so the component targets Go + htmx only. Hugo
sites consume rendered HTML, never the editor.

## Use cases

Three uses drive the design. They differ along three axes — how permissive
the render policy is, which editor chrome the author sees, and whether the
content is even Markdown — which is exactly why those three are separate,
configurable concerns rather than one fixed editor.

### 1. Comments — constrained Markdown

Small textarea, short content, minimal formatting. Still Markdown, but a
restricted subset: inline emphasis and maybe links, but no headings, no
images, no code blocks; links possibly limited (scheme allowlist, or
disallowed entirely). The point is to keep comments from turning into
full documents.

Implementation (shipped):

- `markdown.Comment(linkPolicy)` — a Restrictive preset built from an empty
  bluemonday policy, allowing only inline emphasis, inline code,
  strikethrough, lists, blockquotes, and fenced code blocks. Headings, images,
  tables, and raw HTML are absent by construction; bare URLs are not
  autolinked. `markdown.Options` gained the element-level controls this needs
  (`Restrictive`, `AllowLists`/`AllowBlockquotes`/`AllowCodeBlocks`) and a
  `LinkPolicy`.
- **Links: relative/same-origin only** (`markdown.LinkRelativeOnly`). Relative
  paths and fragments survive; every scheme-bearing or protocol-relative
  (`//host`) URL is stripped, so comments cannot carry outbound spam.
  (Decision: comment links are allowed but same-origin only.)
- **Toolbar profile on `Field`** (`Field.Toolbar`: `minimal`/`standard`/`full`,
  default `full`). The adapter reads it via `data-mdedit-toolbar`; comments use
  `standard`. The same component is a comment box or a document editor by
  configuration — the toolbar shapes the affordances, the server preset
  enforces the constraints.
- Small `Rows`, low `MaxLength` for the comment field.

### 2. Long-form Markdown — a first-class page

Full formatting: sections (headings), lists, code, tables, embedded images.
The canonical case is **session notes** — the result is a standalone page, not
an inline fragment, authored by non-technical campaign players. This is the
primary case and the reason the component exists in friendly form.

Implications:

- The `Rich()` preset, large textarea. Already supported.
- **Loading a local file** is shipped: `Field.AllowFileLoad` adds a control
  that reads a local `.md` file in the browser and drops it into the editor as
  if typed (never uploaded; same sanitization on Save). See the open-questions
  list.
- **Embedded images** are the remaining open piece. Today the renderer accepts
  Markdown `![](url)`; real upload (paste/drag an image → POST to a host
  endpoint → insert the returned URL) is deferred and split across three
  responsibilities — see the open-questions list and
  [infodancer/ui#14](https://github.com/infodancer/ui/issues/14).
- The display partial already serves both inline and full-page rendering; the
  page shell is the consumer's concern.

### 3. Code editing — non-Markdown (speculative)

Editing source/config, not Markdown: syntax highlighting, no sanitizing
render, no toolbar. Speculative; the goal here is only to **not foreclose it**.

Implications, and why it's already mostly free:

- The adapter seam is content-agnostic — it enhances a `<textarea>` and keeps
  its value authoritative; it never assumes Markdown. So a code mode is a
  different adapter (CodeMirror 6 with a language mode), which is the single
  strongest argument for CM6 as the next adapter after EasyMDE.
- The render step is per-`Field` and host-supplied (`Field.HTML` is whatever
  the host produced). For code there is no Markdown render: the host stores raw
  and displays a syntax-highlighted `<pre><code>`, or validates instead of
  sanitizing. The `markdown` module simply isn't used in this mode.
- Consequence for the core: keep "Markdown" out of the seam's contract and
  keep the render step pluggable, so code mode needs a new adapter and a
  display choice — not changes to the loop. The `mdedit` name stays even if a
  future field edits code; it's the Markdown editor with an extensible seam.

## Decisions

### Placement: a nested module in infodancer/ui

`ui`'s original charter was tokens + base CSS + nav/footer partials, explicitly
*no JS, interactivity is the consumer's problem* — so mdedit (an editor with a
backend save loop) began life as a **separate** sibling module that merely
*depends on* `ui` for `--app-*` tokens. In the 2026-05-29 consolidation,
`ui` became a three-module monorepo: the root CSS/template library plus two
nested Go modules, `ui/markdown` (the sanitizer) and `ui/mdedit` (this
component), each with its own `go.mod` and tag. The boundary that mattered —
mdedit and markdown version independently and carry no Go dependency from the
root `ui` module — is preserved; they simply share a repository now. (Decision:
infodancer org; nested module under `ui`, not a standalone repo.)

### One audited render policy — its own module

The render/sanitize pipeline lives in the **`ui/markdown`** module
(`github.com/infodancer/ui/markdown`), which `mdedit` and every other consumer
(faq, osg, blog) depend on. It is the single sanitization boundary: goldmark
parses; bluemonday filters against an allowlist; bluemonday is the boundary
regardless of the goldmark options chosen. Two presets cover the existing
consumers:

- `Strict()` — CommonMark, no raw HTML, UGCPolicy. Matches osg's behavior.
- `Rich()` — GFM, raw inline HTML (filtered by bluemonday), kbd/sub/sup/mark,
  empty-div stripping. Matches faq's content needs.

Its XSS corpus runs against **every** preset, including the raw-HTML-enabled
one, because that is the dangerous path. osg and faq have migrated onto it,
deleting their copies.

Why a separate module rather than a subpackage of `mdedit`: the sanitizer is a
security boundary that should version on its own cadence, and consumers that
only display stored Markdown (or render Hugo-side in Go) need it without the
editor's embedded assets. `mdedit` itself carries no Go dependency on the root
`ui` module; the host calls `markdown` and passes the result in as `Field.HTML`.

### The editor seam

The client editor sits behind a one-function contract (`assets/mdedit.js`):
an adapter enhances a plain `<textarea>` and keeps the textarea's value
authoritative. The textarea is what htmx/forms serialize, so the rest of the
stack neither knows nor cares which editor is mounted. Swapping editors is
choosing a different adapter (`Field.Adapter`) — no server, template, or htmx
change. The loader owns enhancement on htmx swaps, value flushing before
requests, teardown on element removal, file-load wiring, and optional debounced
live preview. The controller shape is
`getValue`/`setValue`/`sync`/`destroy`/`onChange` (`setValue` was added for the
file-load feature; adding a controller method is additive).

### Editor: EasyMDE first

EasyMDE was chosen for v0.1: a prebuilt bundle vendored as two pinned,
SRI-hashed files (no bundler, no npm), a text-label toolbar friendly to
non-technical authors, and Markdown-source output. Tradeoff accepted: it rides
on legacy CodeMirror 5 and has sporadic upstream maintenance. CodeMirror 6 is
the planned successor adapter once a small build step is acceptable; Toast UI
is a candidate only if true rich-text WYSIWYG becomes a requirement (it fights
a canonical-Markdown pipeline).

### Single renderer; preview is server-side

EasyMDE's bundled client preview (marked.js) is disabled. The only renderer is
the server's `markdown` package, so a preview is an honest picture of what Save
will store — no client/server divergence. The cost is a round-trip for preview
instead of instant split-screen; for occasional editing by non-technical users
that is the right trade.

The dual-render question (instant client preview vs. server-only) is left as a
**toggle** (`Field.LivePreview` plus the adapter's `onChange`) so it can be
experimented with rather than decided up front.

## Public surface (treat with versioning discipline)

- `markdown.Options` / `Strict` / `Rich` / `Renderer` (in `ui/markdown`)
- `AssetsFS()`, `PartialsFS()`, `HeadTags()`
- `Field` (field names) and the partial names `mdedit/{display,edit,preview}`
- The `assets/mdedit.js` adapter contract

Adding fields/options is non-breaking; renaming or removing is breaking. Until
v1.0, consumers pin to a tag.

## Open questions / follow-ups

Driven by the use cases above:

- ~~`Comment()` preset + element-level `markdown.Options`~~ — **shipped.**
  Links resolved to same-origin only (`LinkRelativeOnly`).
- ~~Toolbar/profile knob on `Field`~~ — **shipped** (`minimal`/`standard`/`full`).
- ~~Load a local Markdown file into the editor~~ — **shipped**
  (`Field.AllowFileLoad`, [infodancer/ui#12](https://github.com/infodancer/ui/issues/12) /
  [#13](https://github.com/infodancer/ui/pull/13)). Read client-side, dropped
  into the textarea, never uploaded; same render/sanitize path on Save, so no
  new server surface.
- **Image support** — deferred, split three ways
  ([infodancer/ui#14](https://github.com/infodancer/ui/issues/14)):
  - **What `<img>`/`src` survives** is the `ui/markdown` module's job, not
    mdedit's. Decision: a relative/same-origin-by-default image policy
    (`ImageRelativeOnly`, mirroring `LinkRelativeOnly`) with external behind an
    explicit opt-in — UGCPolicy currently lets external `<img src>` through, a
    tracking/referer-leak vector. Tracked as a separate `markdown` task.
  - **Inserting a reference** is mdedit's small part: a future `Field.UploadURL`
    plus EasyMDE's `imageUploadFunction` (paste/drag → POST → insert
    `![alt](url)` at the cursor). Plain Markdown image syntax, **no shortcode**
    (captions/`srcset` would be a goldmark extension in `markdown`, not an
    mdedit feature). The upload-endpoint contract: `POST {UploadURL}`
    multipart field `image` → `2xx` JSON `{url, alt?}` → adapter inserts the
    reference. mdedit does not validate, store, scan, or strip metadata.
  - **Storing/serving the bytes** is a separate future media module the host
    mounts (magic-byte sniff, EXIF strip, SVG reject, `nosniff`, ideally a
    separate origin). mdedit never touches storage.
  - Sequencing: build the mdedit wiring when the first real consumer needs it;
    until then it is untestable dead code.
- **CodeMirror 6 adapter** (use case 3, and the EasyMDE successor): non-Markdown
  content with a language mode, once a build step is acceptable. Keep Markdown
  out of the seam contract so this needs no core change.

Standing:

- Vendor an SVG icon set (or FontAwesome subset) for a prettier EasyMDE toolbar
  without a CDN; today the toolbar uses text labels.
- Whether `ui/markdown` should expose a `MarkdownFirstParagraph`-style excerpt
  helper (osg has one) as a shared utility.
