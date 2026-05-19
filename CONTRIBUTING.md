# Contributing

This is a personal-portfolio utility, not a general-purpose OSS framework. PRs are welcome but the design scope is intentionally narrow — please read [DESIGN.md](DESIGN.md) first.

## Local development

```bash
task check    # vet + fmt + lint (when implementation lands)
task test     # tests
```

CSS files are hand-written. There's no build pipeline; what's in `assets/css/` is what ships.

## Conventions

- Apache-2.0 license; new files don't need a per-file header (the repo-level LICENSE covers them).
- Token names are the public API; treat additions/changes with version discipline (see DESIGN.md → Versioning).
- The parallel Hugo + Go partial variants are deliberate. Don't consolidate without revisiting the design decision.
- No JavaScript dependencies. No CSS preprocessors.

## Reporting issues

Use GitHub Issues. For security-relevant reports, see [SECURITY.md](SECURITY.md).
