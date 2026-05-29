package markdown

import (
	"html/template"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// DirectiveFunc renders one inline directive's label to an HTML fragment.
//
// Directives are written as :name[label] in the Markdown source, where name
// identifies the registered func and label is raw free-text (it may contain
// punctuation Markdown would otherwise interpret — :, =, +, parentheses,
// quotes). The func receives label verbatim (with \] and \\ unescaped) and
// returns the HTML to emit in its place.
//
// Security: the returned HTML is written into the document and then sanitized
// by bluemonday exactly like all other output, so a DirectiveFunc must
//
//  1. be paired with [Options.PolicyCustomize] to allowlist the elements and
//     attributes it emits — otherwise that markup is stripped (fail-safe), and
//  2. escape label when interpolating it into attributes or text. bluemonday
//     is still the boundary if it forgets (a script payload is neutralized
//     regardless), but a forgotten escape can still mangle the output.
type DirectiveFunc func(label string) template.HTML

// kindDirective is the AST node kind for a parsed inline directive.
var kindDirective = ast.NewNodeKind("Directive")

// directiveNode is the inline AST node produced for a matched :name[label].
// It is a leaf: the label is carried verbatim and is never parsed as Markdown.
type directiveNode struct {
	ast.BaseInline
	name  string
	label string
}

func (n *directiveNode) Kind() ast.NodeKind { return kindDirective }

func (n *directiveNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"name": n.name, "label": n.label}, nil)
}

// directiveParser is the inline parser for :name[label]. It only consumes the
// construct when name is one of the registered directive names; any other use
// of ':' (ratios, URLs, an unregistered :foo[…]) is left untouched as text.
type directiveParser struct {
	names map[string]struct{}
}

func (p *directiveParser) Trigger() []byte { return []byte{':'} }

func (p *directiveParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()
	// line[0] is the ':' trigger. Need at least ":x[" plus a closer.
	if len(line) < 4 || line[0] != ':' {
		return nil
	}

	// Parse the name: [A-Za-z][A-Za-z0-9_-]*
	i := 1
	if !isNameStart(line[i]) {
		return nil
	}
	for i < len(line) && isNameChar(line[i]) {
		i++
	}
	name := string(line[1:i])
	if i >= len(line) || line[i] != '[' {
		return nil
	}
	if _, ok := p.names[name]; !ok {
		return nil
	}
	i++ // consume '['

	// Scan the raw label to the matching unescaped ']', honoring \] and \\.
	var label []byte
	closed := false
	for i < len(line) {
		c := line[i]
		if c == '\\' && i+1 < len(line) {
			if next := line[i+1]; next == ']' || next == '\\' {
				label = append(label, next)
				i += 2
				continue
			}
		}
		if c == ']' {
			closed = true
			i++ // consume ']'
			break
		}
		label = append(label, c)
		i++
	}
	if !closed {
		return nil
	}

	block.Advance(i)
	return &directiveNode{name: name, label: string(label)}
}

func isNameStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isNameChar(c byte) bool {
	return isNameStart(c) || (c >= '0' && c <= '9') || c == '-' || c == '_'
}

// directiveHTMLRenderer renders directiveNode by dispatching to the registered
// DirectiveFunc for its name. The parser only emits nodes for registered
// names, so a missing func is a no-op rather than a panic.
type directiveHTMLRenderer struct {
	funcs map[string]DirectiveFunc
}

func (r *directiveHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindDirective, r.render)
}

func (r *directiveHTMLRenderer) render(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*directiveNode)
	if fn, ok := r.funcs[n.name]; ok {
		_, _ = w.WriteString(string(fn(n.label)))
	}
	return ast.WalkSkipChildren, nil
}

// directiveExtension wires the inline directive parser and renderer into a
// goldmark instance for the configured set of named DirectiveFuncs.
type directiveExtension struct {
	funcs map[string]DirectiveFunc
}

func (e *directiveExtension) Extend(m goldmark.Markdown) {
	names := make(map[string]struct{}, len(e.funcs))
	for name := range e.funcs {
		names[name] = struct{}{}
	}
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(&directiveParser{names: names}, 100),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&directiveHTMLRenderer{funcs: e.funcs}, 100),
	))
}
