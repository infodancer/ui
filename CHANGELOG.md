# Changelog

All notable changes to `infodancer/ui` are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Until `v1.0`, breaking changes can land in any release; consumers pin to a specific tag and treat upgrades deliberately. See the Versioning section in [DESIGN.md](DESIGN.md) for what counts as breaking.

## Unreleased

### Added
- `FooterData.BrandURL` (Go) and `.Site.Params.ui.brand_url` (Hugo) so the footer brand mark can link somewhere other than `/`. Defaults to `/` when empty, matching prior behavior.

### Changed
- DESIGN.md no longer states a fixed token count in prose; the canonical roster is the `tokens.css` file and the assertion list in `ui_test.go`. This avoids drift when tokens are added.
- `NavUser` data shape in DESIGN.md reduced to `DisplayName` to match the shipped Go struct and the rendered partial output. The earlier `MenuURL` / `SignOutURL` fields were spec-only and never implemented.
