# Vendored demo assets

These files back the runnable spike only (`examples/spike`); they are not part
of the published `mdedit` module. They are vendored so the demo runs offline,
with no runtime CDN. Treat them as read-only: to upgrade, re-run the fetch,
recompute the hash, and review the diff.

## htmx 2.0.4

Source: https://htmx.org (BSD Zero Clause License, © Big Sky Software)

Fetched from jsDelivr:

```
curl -fsSL -o htmx.min.js https://cdn.jsdelivr.net/npm/htmx.org@2.0.4/dist/htmx.min.js
```

Subresource Integrity (sha384):

```
htmx.min.js  sha384-HGfztofotfshcF7+8n44JQL2oJmowVChPTg48S+jvZoztPfvwD79OC/LTtG6dMp+
```

Verify a local copy:

```
openssl dgst -sha384 -binary htmx.min.js | openssl base64 -A
```
