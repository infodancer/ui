# Security Policy

## Reporting a Vulnerability

Use [GitHub's private vulnerability reporting](https://github.com/infodancer/ui/security/advisories/new) for any security-relevant issue. Public issue tracking is fine for non-security bugs.

## Scope

This repository ships HTML + CSS only. Security-relevant areas:

- CSS that affects content-injection safety in consumer sites (e.g., a selector that breaks a consumer's escaping assumptions).
- Hugo or Go partials that emit consumer-controlled content without proper escaping (the partials use the standard Hugo / Go html/template auto-escaping; an issue here would be a regression).

Out of scope: client-side JavaScript (we don't ship any), server-side code in consumers (their responsibility).
