package markdown

import (
	"html/template"
	"strings"
	"testing"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// sampleRoll is a well-behaved DirectiveFunc that escapes its label before
// interpolating it — the pattern a real consumer (osg's roll) follows.
func sampleRoll(label string) template.HTML {
	esc := template.HTMLEscapeString(label)
	//nolint:gosec // label is escaped above; output is also re-sanitized by bluemonday.
	return template.HTML(`<span class="roll" title="` + esc + `">🎲</span>`)
}

// allowRoll is the PolicyCustomize that opens the bluemonday gate for the
// markup sampleRoll emits.
func allowRoll(p *bluemonday.Policy) {
	p.AllowAttrs("class", "title").OnElements("span")
}

func rollOptions() Options {
	return Options{
		Directives:      map[string]DirectiveFunc{"roll": sampleRoll},
		PolicyCustomize: allowRoll,
	}
}

func TestDirective_RendersRegistered(t *testing.T) {
	r := New(rollOptions())
	got := r.RenderString(":roll[1d20+3 >= 11]")
	for _, want := range []string{`<span`, `class="roll"`, `title="1d20+3 &gt;= 11"`, "🎲"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestDirective_StaysInline(t *testing.T) {
	r := New(rollOptions())
	got := r.RenderString("Heinrich cast it :roll[1d20+3 >= 11] and it worked.")
	if n := strings.Count(got, "<p>"); n != 1 {
		t.Errorf("directive should stay inline within one paragraph, got %d <p>:\n%s", n, got)
	}
	for _, want := range []string{"Heinrich cast it", "and it worked.", `class="roll"`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got:\n%s", want, got)
		}
	}
}

func TestDirective_StrippedWithoutPolicyCustomize(t *testing.T) {
	// Directive registered, but its markup is NOT allowlisted: bluemonday must
	// strip the distinguishing class/title (UGCPolicy keeps a bare <span>, but
	// not the attributes that drive styling/behavior). Fail-safe by default.
	r := New(Options{Directives: map[string]DirectiveFunc{"roll": sampleRoll}})
	got := r.RenderString(":roll[1d20+3 >= 11]")
	if strings.Contains(got, `class="roll"`) {
		t.Errorf("class should be stripped without PolicyCustomize, got:\n%s", got)
	}
	if strings.Contains(got, "title=") {
		t.Errorf("title should be stripped without PolicyCustomize, got:\n%s", got)
	}
	// The inner glyph survives as text.
	if !strings.Contains(got, "🎲") {
		t.Errorf("inner text should survive sanitization, got:\n%s", got)
	}
}

func TestDirective_Unregistered_LeftAsText(t *testing.T) {
	r := New(rollOptions())
	got := r.RenderString("see :foo[bar] here")
	if !strings.Contains(got, ":foo[bar]") {
		t.Errorf("unregistered directive name should pass through as literal text, got:\n%s", got)
	}
}

func TestDirective_NoBracketNotConsumed(t *testing.T) {
	r := New(rollOptions())
	got := r.RenderString("the :roll was good")
	if !strings.Contains(got, ":roll") {
		t.Errorf(":name without a bracket should be literal text, got:\n%s", got)
	}
}

func TestDirective_OrdinaryColonsUntouched(t *testing.T) {
	r := New(rollOptions())
	got := r.RenderString("ratio 3:1 and http://example.com/x stay literal")
	for _, want := range []string{"3:1", "http://example.com/x"} {
		if !strings.Contains(got, want) {
			t.Errorf("ordinary colon usage %q should be untouched, got:\n%s", want, got)
		}
	}
}

func TestDirective_EmptyLabel(t *testing.T) {
	r := New(rollOptions())
	got := r.RenderString(":roll[]")
	if !strings.Contains(got, `title=""`) {
		t.Errorf("empty label should yield empty title, got:\n%s", got)
	}
}

func TestDirective_SpecialCharsInLabel(t *testing.T) {
	r := New(rollOptions())
	got := r.RenderString(`:roll[Heinrich casts Sleep: 1d20+3 >= 11 succeeded with 14]`)
	// The colon, plus, and >= inside the label must all survive (escaped).
	if !strings.Contains(got, "Heinrich casts Sleep: 1d20+3 &gt;= 11 succeeded with 14") {
		t.Errorf("mechanics string should round-trip into the title, got:\n%s", got)
	}
}

func TestDirective_EscapedBracket(t *testing.T) {
	r := New(rollOptions())
	got := r.RenderString(`:roll[a\]b]`)
	if !strings.Contains(got, `title="a]b"`) {
		t.Errorf("escaped bracket should be unescaped into the label, got:\n%s", got)
	}
}

func TestDirective_XSS_EscapingRenderer(t *testing.T) {
	r := New(rollOptions())
	got := r.RenderString(`:roll[">` + "<script>alert(1)</script>" + `]`)
	// The label is escaped into an attribute value: no executable <script> tag
	// survives. (The inert escaped text &lt;script&gt; is harmless.)
	if strings.Contains(strings.ToLower(got), "<script") {
		t.Errorf("an executable <script> survived a label, got:\n%s", got)
	}
}

func TestDirective_XSS_UnsafeRendererStillSanitized(t *testing.T) {
	// A deliberately unsafe DirectiveFunc that does NOT escape its label.
	// bluemonday is the boundary regardless: the script must still be gone.
	unsafe := func(label string) template.HTML {
		//nolint:gosec // intentionally unescaped to prove bluemonday is the boundary.
		return template.HTML(`<span class="roll" title="` + label + `">x</span>`)
	}
	r := New(Options{
		Directives:      map[string]DirectiveFunc{"roll": unsafe},
		PolicyCustomize: allowRoll,
	})
	got := r.RenderString(`:roll[">` + "<script>alert(1)</script>" + `]`)
	if strings.Contains(strings.ToLower(got), "<script") {
		t.Errorf("bluemonday must neutralize script even from an unescaping renderer, got:\n%s", got)
	}
}

func TestExtensions_Passthrough(t *testing.T) {
	// A goldmark extension the presets don't expose, supplied via Options.Extensions.
	r := New(Options{Extensions: []goldmark.Extender{extension.Strikethrough}})
	got := r.RenderString("~~gone~~")
	if !strings.Contains(got, "<del>") {
		t.Errorf("Options.Extensions should enable the passed extender, got:\n%s", got)
	}
}

func TestPolicyCustomize_AllowlistsElement(t *testing.T) {
	// UGCPolicy permits <span> but strips a data-* attribute from it; allowing
	// that attribute is precisely what PolicyCustomize is for. (A made-up
	// element name won't do — goldmark drops tags it doesn't recognize before
	// bluemonday ever sees them.)
	const in = `an <span data-roll="x">y</span> here`
	with := New(Options{
		AllowRawHTML:    true,
		PolicyCustomize: func(p *bluemonday.Policy) { p.AllowAttrs("data-roll").OnElements("span") },
	})
	if got := with.RenderString(in); !strings.Contains(got, `data-roll="x"`) {
		t.Errorf("PolicyCustomize should allowlist the data-roll attribute, got:\n%s", got)
	}

	without := New(Options{AllowRawHTML: true})
	if got := without.RenderString(in); strings.Contains(got, "data-roll") {
		t.Errorf("without PolicyCustomize, data-roll should be stripped, got:\n%s", got)
	}
}
