# Vendored editor assets

These files are third-party, pinned, and checked in deliberately so the
module is self-contained — no runtime CDN, no npm, no bundler. Treat them
as read-only: to upgrade, re-run the fetch below, recompute the hashes,
and review the diff.

## EasyMDE 2.20.0

Source: https://www.npmjs.com/package/easymde (MIT, © Jeroen Akkerman et al.)
Bundles CodeMirror 5 + marked.js. We disable marked.js preview at mount
time (see `assets/adapters/easymde.js`); the authoritative renderer is the
server-side `markdown` package (goldmark + bluemonday).

Fetched from jsDelivr:

```
curl -fsSL -o easymde.min.js  https://cdn.jsdelivr.net/npm/easymde@2.20.0/dist/easymde.min.js
curl -fsSL -o easymde.min.css https://cdn.jsdelivr.net/npm/easymde@2.20.0/dist/easymde.min.css
```

Subresource Integrity (sha384), for the `integrity=` attribute on the
`<script>`/`<link>` tags the module emits:

```
easymde.min.js   sha384-YDXeUfPZ4SP6vJpnF+ZMmf4B1bax6yd4Q/aNbkvLidRD843hPG5RE67M0IYT4LOq
easymde.min.css  sha384-3AvV7152TgYAMYdGZPqG9BpmSH2ZW6ewTDL0QV5PyNkl19KMI+yLMdJz183N8A2d
```

Verify a local copy:

```
openssl dgst -sha384 -binary easymde.min.js | openssl base64 -A
```
