package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
)

// Viewer carries everything the menu gate logic needs about the current
// request's subject. The zero value is the anonymous viewer (not
// authenticated, no roles), so an unauthenticated request gates correctly
// with no special-casing.
//
// ui deliberately defines its own minimal viewer rather than importing
// github.com/infodancer/authz: the gate logic needs only role membership plus
// two identity-assurance bits, and keeping the type here leaves ui's
// dependency surface at the standard library. An authz.Principal adapts in one
// line:
//
//	ui.Viewer{Authenticated: true, EmailVerified: p.EmailVerified, Roles: p.Roles}
type Viewer struct {
	Authenticated bool
	EmailVerified bool
	Roles         []string
}

// HasRole reports whether the viewer holds role.
func (v Viewer) HasRole(role string) bool { return slices.Contains(v.Roles, role) }

// Gate is the visibility condition on a [MenuItem]. Every set field is a
// requirement and all of them must pass (logical AND); the zero Gate is
// "always visible". Gates compose with the item tree — see [Resolve].
//
// The fields carry both json and yaml tags so the declarative menu config
// round-trips through either encoding. ui parses JSON itself (see
// [ParseMenu]); a YAML host unmarshals into the same structs with its own
// library, keeping ui free of a YAML dependency.
type Gate struct {
	// RequireAuth shows the item only to authenticated viewers.
	RequireAuth bool `json:"requireAuth,omitempty" yaml:"requireAuth,omitempty"`
	// RequireAnon shows the item only to anonymous viewers (e.g. "Sign in").
	// With RequireAuth it can never pass — a misconfiguration that simply
	// hides the item, consistent with fail-closed gating.
	RequireAnon bool `json:"requireAnon,omitempty" yaml:"requireAnon,omitempty"`
	// RequireVerified shows the item only when the viewer's email is verified.
	RequireVerified bool `json:"requireVerified,omitempty" yaml:"requireVerified,omitempty"`
	// RequireRoles lists role names; the viewer must hold at least one
	// (any-of). Set RequireAllRoles to require every listed role instead.
	RequireRoles []string `json:"requireRoles,omitempty" yaml:"requireRoles,omitempty"`
	// RequireAllRoles flips RequireRoles from any-of to all-of.
	RequireAllRoles bool `json:"requireAllRoles,omitempty" yaml:"requireAllRoles,omitempty"`
	// CustomGate names a predicate registered in the [Registry] passed to
	// [Resolve], for rules the declarative fields can't express. A name with
	// no registered predicate fails closed (the item is hidden) — a missing
	// security gate must never default to visible.
	CustomGate string `json:"customGate,omitempty" yaml:"customGate,omitempty"`
}

// allow reports whether v satisfies the gate. reg supplies custom predicates;
// a referenced-but-absent (or nil) custom gate returns false — fail closed.
func (g Gate) allow(v Viewer, reg Registry) bool {
	if g.RequireAuth && !v.Authenticated {
		return false
	}
	if g.RequireAnon && v.Authenticated {
		return false
	}
	if g.RequireVerified && !v.EmailVerified {
		return false
	}
	if len(g.RequireRoles) > 0 && !rolesPass(v, g.RequireRoles, g.RequireAllRoles) {
		return false
	}
	if g.CustomGate != "" {
		fn, ok := reg[g.CustomGate]
		if !ok || fn == nil || !fn(v) {
			return false
		}
	}
	return true
}

func rolesPass(v Viewer, roles []string, all bool) bool {
	if all {
		for _, r := range roles {
			if !v.HasRole(r) {
				return false
			}
		}
		return true
	}
	for _, r := range roles {
		if v.HasRole(r) {
			return true
		}
	}
	return false
}

// MenuItem kinds. The zero value (KindLink) is a link or a dropdown parent;
// the others select an alternate presentation in the ui/nav partial.
const (
	KindLink      = ""          // a link, or a dropdown parent when it has children
	KindIcon      = "icon"      // an icon affordance (e.g. the notification bell)
	KindSeparator = "separator" // a visual divider within a dropdown; carries no link
	KindSpacer    = "spacer"    // a flex spacer; pushes the items after it to the right
)

// MenuItem is one entry in a menu (see [NavData.Items]). An item with Children
// and no URL is a pure dropdown parent; with both a URL and Children it is a
// link that also opens a submenu. Kind selects an alternate presentation.
//
// The struct carries json and yaml tags for the declarative config; the
// per-request fields (Badge, Muted) are populated in code and never
// serialized.
type MenuItem struct {
	Key      string     `json:"key" yaml:"key"`
	Label    string     `json:"label,omitempty" yaml:"label,omitempty"`
	URL      string     `json:"url,omitempty" yaml:"url,omitempty"`
	Icon     string     `json:"icon,omitempty" yaml:"icon,omitempty"`
	Kind     string     `json:"kind,omitempty" yaml:"kind,omitempty"`
	Gate     Gate       `json:"gate,omitzero" yaml:"gate,omitempty"`
	Children []MenuItem `json:"children,omitempty" yaml:"children,omitempty"`

	// Badge is live, per-request state (an unread count, a status dot) the
	// host attaches after [Resolve] — never part of static config, hence no
	// struct tags. A nil Badge renders no badge.
	Badge *Badge `json:"-" yaml:"-"`

	// Muted is set by [Resolve] on a KindIcon item that passes its own gate
	// but has no surviving URL or children: the affordance still renders, but
	// inert (the bell a signed-in non-admin sees). The partial styles it with
	// .app-nav-bell--muted and emits no link.
	Muted bool `json:"-" yaml:"-"`
}

// Badge is the detectable state on an item, typically a KindIcon. Count is the
// headline number (0 renders no count and the empty badge collapses via CSS);
// Label is accessible text for the affordance; State is a render hint
// (data-state, e.g. "active" to light the icon); PollURL, when set, is an htmx
// endpoint the partial wraps the badge in so it refreshes live
// (hx-get=PollURL, hx-trigger="load, every 30s").
type Badge struct {
	Count   int
	Label   string
	State   string
	PollURL string
}

// Registry maps [Gate.CustomGate] names to predicates. Build it once at
// startup and pass it to [Resolve] on each request. A nil or empty Registry is
// fine for menus that use no custom gates.
type Registry map[string]func(Viewer) bool

// Resolve returns a copy of nav containing only the items viewer may see, with
// every item's Children filtered recursively. It evaluates each [Gate] against
// viewer (custom gates via reg, fail-closed on a missing name) and applies the
// empty-parent rules:
//
//   - A KindLink item whose own gate passes but which has no URL and no
//     surviving children is dropped — an empty dropdown carries no meaning.
//   - A KindIcon item whose own gate passes but which has no URL and no
//     surviving children is kept with Muted set: the affordance renders inert.
//     This is the notification bell a signed-in non-admin sees.
//   - Separators that end up leading, trailing, or doubled after pruning are
//     removed, so a divider never dangles around gated-out neighbours. Spacers
//     are collapsed when consecutive and dropped when trailing (a leading
//     spacer — everything right-aligned — is kept).
//
// Resolve never mutates nav or its items.
func Resolve(nav NavData, viewer Viewer, reg Registry) NavData {
	out := nav
	out.Items = resolveItems(nav.Items, viewer, reg)
	return out
}

func resolveItems(items []MenuItem, v Viewer, reg Registry) []MenuItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]MenuItem, 0, len(items))
	for _, it := range items {
		if !it.Gate.allow(v, reg) {
			continue
		}
		it.Children = resolveItems(it.Children, v, reg)
		if !isStructural(it.Kind) && it.URL == "" && len(it.Children) == 0 {
			if it.Kind == KindIcon {
				it.Muted = true
			} else {
				continue // empty dropdown parent
			}
		}
		out = append(out, it)
	}
	return tidyStructural(out)
}

// isStructural reports whether a kind is a contentless layout element
// (separator, spacer) rather than something that needs a link or children.
func isStructural(kind string) bool {
	return kind == KindSeparator || kind == KindSpacer
}

// tidyStructural cleans up the contentless layout items left by gating:
// separators dropped when leading, trailing, or doubled (a divider needs
// content on both sides); spacers collapsed when consecutive and dropped when
// trailing (a leading spacer right-aligns everything and is kept).
func tidyStructural(items []MenuItem) []MenuItem {
	out := make([]MenuItem, 0, len(items))
	for _, it := range items {
		switch it.Kind {
		case KindSeparator:
			if len(out) == 0 || out[len(out)-1].Kind == KindSeparator {
				continue
			}
		case KindSpacer:
			if len(out) > 0 && out[len(out)-1].Kind == KindSpacer {
				continue
			}
		}
		out = append(out, it)
	}
	for len(out) > 0 && isStructural(out[len(out)-1].Kind) {
		out = out[:len(out)-1]
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ParseMenu decodes a JSON array of [MenuItem]s — the declarative menu config —
// from r. Unknown fields are tolerated so a newer config stays loadable by an
// older binary. YAML consumers unmarshal into []MenuItem with their own
// library (the structs carry yaml tags), which keeps ui's dependency surface
// at the standard library.
func ParseMenu(r io.Reader) ([]MenuItem, error) {
	var items []MenuItem
	if err := json.NewDecoder(r).Decode(&items); err != nil {
		return nil, fmt.Errorf("ui: parse menu: %w", err)
	}
	return items, nil
}
