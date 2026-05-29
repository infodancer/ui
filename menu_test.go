package ui_test

import (
	"bytes"
	"html/template"
	"strings"
	"testing"

	"github.com/infodancer/ui"
)

// --- Gate evaluation (via Resolve, which is the only public path) -----------

// gateVisible reports whether a single top-level item with gate g survives
// Resolve for viewer v and registry reg.
func gateVisible(g ui.Gate, v ui.Viewer, reg ui.Registry) bool {
	nav := ui.NavData{Items: []ui.MenuItem{{Key: "x", Label: "X", URL: "/x", Gate: g}}}
	return len(ui.Resolve(nav, v, reg).Items) == 1
}

func TestGate_TruthTable(t *testing.T) {
	t.Parallel()

	anon := ui.Viewer{}
	authed := ui.Viewer{Authenticated: true}
	verified := ui.Viewer{Authenticated: true, EmailVerified: true}
	admin := ui.Viewer{Authenticated: true, Roles: []string{"admin"}}
	editorAdmin := ui.Viewer{Authenticated: true, Roles: []string{"editor", "admin"}}

	cases := []struct {
		name string
		gate ui.Gate
		view ui.Viewer
		want bool
	}{
		{"zero gate, anon", ui.Gate{}, anon, true},
		{"requireAuth blocks anon", ui.Gate{RequireAuth: true}, anon, false},
		{"requireAuth allows authed", ui.Gate{RequireAuth: true}, authed, true},
		{"requireAnon allows anon", ui.Gate{RequireAnon: true}, anon, true},
		{"requireAnon blocks authed", ui.Gate{RequireAnon: true}, authed, false},
		{"auth+anon never pass", ui.Gate{RequireAuth: true, RequireAnon: true}, authed, false},
		{"requireVerified blocks unverified", ui.Gate{RequireVerified: true}, authed, false},
		{"requireVerified allows verified", ui.Gate{RequireVerified: true}, verified, true},
		{"role any-of, holder", ui.Gate{RequireRoles: []string{"admin"}}, admin, true},
		{"role any-of, non-holder", ui.Gate{RequireRoles: []string{"admin"}}, authed, false},
		{"role any-of, one of many", ui.Gate{RequireRoles: []string{"admin", "owner"}}, admin, true},
		{"role all-of, has all", ui.Gate{RequireRoles: []string{"editor", "admin"}, RequireAllRoles: true}, editorAdmin, true},
		{"role all-of, missing one", ui.Gate{RequireRoles: []string{"editor", "admin"}, RequireAllRoles: true}, admin, false},
		{"combined AND, all pass", ui.Gate{RequireAuth: true, RequireVerified: true, RequireRoles: []string{"admin"}}, ui.Viewer{Authenticated: true, EmailVerified: true, Roles: []string{"admin"}}, true},
		{"combined AND, one fails", ui.Gate{RequireAuth: true, RequireVerified: true, RequireRoles: []string{"admin"}}, admin, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := gateVisible(tc.gate, tc.view, nil); got != tc.want {
				t.Errorf("gate %+v for viewer %+v: got %v, want %v", tc.gate, tc.view, got, tc.want)
			}
		})
	}
}

func TestGate_CustomGate(t *testing.T) {
	t.Parallel()

	reg := ui.Registry{
		"beta": func(v ui.Viewer) bool { return v.HasRole("beta") },
	}
	beta := ui.Viewer{Authenticated: true, Roles: []string{"beta"}}
	plain := ui.Viewer{Authenticated: true}

	if !gateVisible(ui.Gate{CustomGate: "beta"}, beta, reg) {
		t.Error("custom gate should pass for a beta viewer")
	}
	if gateVisible(ui.Gate{CustomGate: "beta"}, plain, reg) {
		t.Error("custom gate should block a non-beta viewer")
	}
}

func TestGate_MissingCustomGateFailsClosed(t *testing.T) {
	t.Parallel()

	// A gate naming a predicate that isn't registered must hide the item — a
	// missing security gate may never default to visible.
	if gateVisible(ui.Gate{CustomGate: "nope"}, ui.Viewer{Authenticated: true}, ui.Registry{}) {
		t.Error("unregistered custom gate should fail closed (hide the item)")
	}
	if gateVisible(ui.Gate{CustomGate: "nope"}, ui.Viewer{Authenticated: true}, nil) {
		t.Error("nil registry with a custom gate should fail closed")
	}
}

// --- Resolve: tree pruning --------------------------------------------------

func TestResolve_PrunesHiddenItemsAndChildren(t *testing.T) {
	t.Parallel()

	nav := ui.NavData{Items: []ui.MenuItem{
		{Key: "home", Label: "Home", URL: "/"},
		{Key: "admin", Label: "Admin", URL: "/admin", Gate: ui.Gate{RequireRoles: []string{"admin"}}},
		{Key: "acct", Label: "Account", Children: []ui.MenuItem{
			{Key: "prefs", Label: "Prefs", URL: "/prefs"},
			{Key: "billing", Label: "Billing", URL: "/billing", Gate: ui.Gate{RequireRoles: []string{"admin"}}},
		}},
	}}

	got := ui.Resolve(nav, ui.Viewer{Authenticated: true}, nil).Items
	keys := itemKeys(got)
	if want := []string{"home", "acct"}; !equalStrings(keys, want) {
		t.Fatalf("top-level keys = %v, want %v", keys, want)
	}
	// The "acct" dropdown keeps only the ungated child.
	acct := findItem(got, "acct")
	if acct == nil {
		t.Fatal("acct item missing")
	}
	if ck := itemKeys(acct.Children); !equalStrings(ck, []string{"prefs"}) {
		t.Errorf("acct children = %v, want [prefs]", ck)
	}
}

func TestResolve_EmptyDropdownDropped(t *testing.T) {
	t.Parallel()

	// A dropdown parent (no own URL) whose every child gates out disappears.
	nav := ui.NavData{Items: []ui.MenuItem{
		{Key: "admin-menu", Label: "Admin", Children: []ui.MenuItem{
			{Key: "users", Label: "Users", URL: "/users", Gate: ui.Gate{RequireRoles: []string{"admin"}}},
		}},
	}}
	got := ui.Resolve(nav, ui.Viewer{Authenticated: true}, nil).Items
	if len(got) != 0 {
		t.Errorf("empty dropdown should be dropped, got %v", itemKeys(got))
	}
}

func TestResolve_IconWithoutSurvivingChildrenIsMuted(t *testing.T) {
	t.Parallel()

	// The notification bell: visible to any signed-in viewer, links to the
	// admin page only for admins, muted (inert) for everyone else.
	bell := ui.MenuItem{
		Key: "notifications", Kind: ui.KindIcon, Icon: "bell",
		Gate: ui.Gate{RequireAuth: true},
		Children: []ui.MenuItem{
			{Key: "admin", Label: "Notifications", URL: "/admin/notifications/", Gate: ui.Gate{RequireRoles: []string{"admin"}}},
		},
	}
	nav := ui.NavData{Items: []ui.MenuItem{bell}}

	// Admin: bell present, interactive (child survives), not muted.
	admin := findItem(ui.Resolve(nav, ui.Viewer{Authenticated: true, Roles: []string{"admin"}}, nil).Items, "notifications")
	if admin == nil || admin.Muted || len(admin.Children) != 1 {
		t.Errorf("admin bell should be present, interactive, unmuted; got %+v", admin)
	}

	// Signed-in non-admin: bell present but muted, no children.
	user := findItem(ui.Resolve(nav, ui.Viewer{Authenticated: true}, nil).Items, "notifications")
	if user == nil || !user.Muted || len(user.Children) != 0 {
		t.Errorf("non-admin bell should be present and muted; got %+v", user)
	}

	// Anonymous: bell absent (gate requires auth).
	if anon := findItem(ui.Resolve(nav, ui.Viewer{}, nil).Items, "notifications"); anon != nil {
		t.Errorf("anonymous bell should be absent; got %+v", anon)
	}
}

func TestResolve_TidiesSeparators(t *testing.T) {
	t.Parallel()

	sep := ui.MenuItem{Kind: ui.KindSeparator}
	nav := ui.NavData{Items: []ui.MenuItem{
		sep, // leading -> dropped
		{Key: "a", Label: "A", URL: "/a"},
		sep,
		{Key: "gated", Label: "G", URL: "/g", Gate: ui.Gate{RequireRoles: []string{"admin"}}}, // removed, leaving a doubled sep
		sep,
		{Key: "b", Label: "B", URL: "/b"},
		sep, // trailing -> dropped
	}}
	got := ui.Resolve(nav, ui.Viewer{Authenticated: true}, nil).Items
	// Expect: a, <one sep>, b — no leading/trailing/doubled separators.
	if len(got) != 3 || got[0].Key != "a" || got[1].Kind != ui.KindSeparator || got[2].Key != "b" {
		t.Errorf("separator tidy failed: %v", describe(got))
	}
}

func TestResolve_DoesNotMutateInput(t *testing.T) {
	t.Parallel()

	nav := ui.NavData{Items: []ui.MenuItem{
		{Key: "bell", Kind: ui.KindIcon, Gate: ui.Gate{RequireAuth: true}},
	}}
	_ = ui.Resolve(nav, ui.Viewer{Authenticated: true}, nil)
	if nav.Items[0].Muted {
		t.Error("Resolve mutated the caller's input item (set Muted)")
	}
}

// --- Render -----------------------------------------------------------------

func renderNav(t *testing.T, nav ui.NavData) string {
	t.Helper()
	tmpl, err := template.ParseFS(ui.PartialsFS(), "*.gohtml")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "ui/nav", nav); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestNav_RendersDropdown(t *testing.T) {
	t.Parallel()
	out := renderNav(t, ui.NavData{
		BrandText: "Example",
		Items: []ui.MenuItem{
			{Key: "home", Label: "Home", URL: "/"},
			{Key: "tools", Label: "Tools", Children: []ui.MenuItem{
				{Key: "ng", Label: "Namegen", URL: "/tools/namegen/"},
			}},
		},
	})
	for _, want := range []string{
		`<details class="app-nav-dropdown">`,
		`<summary>Tools</summary>`,
		`<ul class="app-nav-menu" role="list">`,
		`href="/tools/namegen/"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("dropdown render missing %q\n--- got ---\n%s", want, out)
		}
	}
}

func TestNav_RendersBellStates(t *testing.T) {
	t.Parallel()

	bellCfg := ui.MenuItem{
		Key: "notifications", Kind: ui.KindIcon, Icon: "bell",
		Gate: ui.Gate{RequireAuth: true},
		Children: []ui.MenuItem{
			{Key: "admin", Label: "Notifications", URL: "/admin/notifications/", Gate: ui.Gate{RequireRoles: []string{"admin"}}},
		},
	}

	// Admin sees a clickable bell linking to the admin page.
	adminNav := ui.Resolve(ui.NavData{Items: []ui.MenuItem{bellCfg}}, ui.Viewer{Authenticated: true, Roles: []string{"admin"}}, nil)
	adminOut := renderNav(t, adminNav)
	if !strings.Contains(adminOut, `data-icon="bell"`) {
		t.Errorf("admin bell missing glyph slot\n%s", adminOut)
	}
	if !strings.Contains(adminOut, `href="/admin/notifications/"`) {
		t.Errorf("admin bell should link to the admin page\n%s", adminOut)
	}

	// Non-admin sees a muted, non-interactive bell.
	userNav := ui.Resolve(ui.NavData{Items: []ui.MenuItem{bellCfg}}, ui.Viewer{Authenticated: true}, nil)
	userOut := renderNav(t, userNav)
	if !strings.Contains(userOut, "app-nav-bell--muted") {
		t.Errorf("non-admin bell should be muted\n%s", userOut)
	}
	if strings.Contains(userOut, "/admin/notifications/") {
		t.Errorf("non-admin bell must not link to the admin page\n%s", userOut)
	}

	// Anonymous: no bell at all.
	anonOut := renderNav(t, ui.Resolve(ui.NavData{Items: []ui.MenuItem{bellCfg}}, ui.Viewer{}, nil))
	if strings.Contains(anonOut, "app-nav-bell") {
		t.Errorf("anonymous viewer should see no bell\n%s", anonOut)
	}
}

func TestNav_BadgeWithPollURL(t *testing.T) {
	t.Parallel()
	nav := ui.NavData{Items: []ui.MenuItem{
		{Key: "b", Kind: ui.KindIcon, Icon: "bell", URL: "/n",
			Badge: &ui.Badge{Count: 3, Label: "3 unread", State: "active", PollURL: "/n/bell"}},
	}}
	out := renderNav(t, nav)
	for _, want := range []string{
		`hx-get="/n/bell"`,
		`hx-trigger="load, every 30s"`,
		`data-state="active"`,
		`aria-label="3 unread"`,
		`>3</span>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("badge render missing %q\n--- got ---\n%s", want, out)
		}
	}
}

func TestNav_ZeroCountBadgeRendersEmpty(t *testing.T) {
	t.Parallel()
	nav := ui.NavData{Items: []ui.MenuItem{
		{Key: "b", Kind: ui.KindIcon, Icon: "bell", URL: "/n", Badge: &ui.Badge{Count: 0}},
	}}
	out := renderNav(t, nav)
	if !strings.Contains(out, `<span class="app-nav-badge"></span>`) {
		t.Errorf("zero-count badge should render empty (CSS collapses it)\n--- got ---\n%s", out)
	}
}

func TestNav_LegacyLinksStillRender(t *testing.T) {
	t.Parallel()
	// A consumer not yet migrated to Items keeps working via Links.
	out := renderNav(t, ui.NavData{
		BrandText: "Example",
		Links:     []ui.NavLink{{Label: "Browse", URL: "/browse"}},
	})
	if !strings.Contains(out, `href="/browse"`) || !strings.Contains(out, "Browse") {
		t.Errorf("legacy Links should still render when Items is empty\n--- got ---\n%s", out)
	}
}

func TestNav_ItemsTakePrecedenceOverLinks(t *testing.T) {
	t.Parallel()
	out := renderNav(t, ui.NavData{
		Items: []ui.MenuItem{{Key: "n", Label: "New", URL: "/new"}},
		Links: []ui.NavLink{{Label: "Old", URL: "/old"}},
	})
	if !strings.Contains(out, `href="/new"`) {
		t.Errorf("Items should render\n%s", out)
	}
	if strings.Contains(out, `href="/old"`) {
		t.Errorf("Links should be ignored when Items is set\n%s", out)
	}
}

// --- Config round-trip ------------------------------------------------------

func TestParseMenu_JSON(t *testing.T) {
	t.Parallel()
	const cfg = `[
	  {"key":"home","label":"Home","url":"/"},
	  {"key":"notifications","kind":"icon","icon":"bell",
	   "gate":{"requireAuth":true},
	   "children":[
	     {"key":"admin","label":"Notifications","url":"/admin/notifications/",
	      "gate":{"requireRoles":["admin"]}}
	   ]}
	]`
	items, err := ui.ParseMenu(strings.NewReader(cfg))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	bell := items[1]
	if bell.Kind != ui.KindIcon || bell.Icon != "bell" || !bell.Gate.RequireAuth {
		t.Errorf("bell decoded wrong: %+v", bell)
	}
	if len(bell.Children) != 1 || !equalStrings(bell.Children[0].Gate.RequireRoles, []string{"admin"}) {
		t.Errorf("bell child decoded wrong: %+v", bell.Children)
	}
}

func TestParseMenu_ToleratesUnknownFields(t *testing.T) {
	t.Parallel()
	// Forward-compat: an older binary loads a newer config without choking.
	items, err := ui.ParseMenu(strings.NewReader(`[{"key":"x","label":"X","url":"/x","futureField":true}]`))
	if err != nil {
		t.Fatalf("unknown fields should be tolerated: %v", err)
	}
	if len(items) != 1 || items[0].Key != "x" {
		t.Errorf("decode lost data: %+v", items)
	}
}

// --- helpers ----------------------------------------------------------------

func itemKeys(items []ui.MenuItem) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		if it.Kind == ui.KindSeparator {
			out = append(out, "|")
			continue
		}
		out = append(out, it.Key)
	}
	return out
}

func findItem(items []ui.MenuItem, key string) *ui.MenuItem {
	for i := range items {
		if items[i].Key == key {
			return &items[i]
		}
	}
	return nil
}

func describe(items []ui.MenuItem) string {
	parts := make([]string, len(items))
	for i, it := range items {
		if it.Kind == ui.KindSeparator {
			parts[i] = "sep"
		} else {
			parts[i] = it.Key
		}
	}
	return strings.Join(parts, ",")
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
