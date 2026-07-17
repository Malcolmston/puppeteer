package puppeteer

import (
	"fmt"
	"strconv"
	"strings"
)

// selector is a compiled CSS selector list (comma-separated groups).
type selector struct {
	groups []complexSelector
}

// complexSelector is a chain of compound selectors joined by combinators. parts
// is left-to-right; combs[i] is the combinator between parts[i] and parts[i+1].
type complexSelector struct {
	parts []compoundSelector
	combs []byte // ' ', '>', '+', '~'; len == len(parts)-1
}

// compoundSelector is a set of simple selectors that must all match one element.
type compoundSelector struct {
	tag     string // "" or "*" means any tag
	id      string
	classes []string
	attrs   []attrSelector
	pseudos []pseudoSelector
}

// attrSelector matches on an attribute. op is one of "", "=", "^=", "$=", "*=",
// "~=", "|=".
type attrSelector struct {
	name string
	op   string
	val  string
}

// pseudoSelector matches structural pseudo-classes. For nth-child style
// pseudos, the pattern is a*n + b (1-based).
type pseudoSelector struct {
	name string
	a, b int
}

// compileSelector parses a selector string into a reusable compiled selector.
func compileSelector(s string) (*selector, error) {
	p := &selParser{src: s}
	return p.parseSelectorList()
}

type selParser struct {
	src string
	pos int
}

func (p *selParser) parseSelectorList() (*selector, error) {
	sel := &selector{}
	for {
		p.skipSpace()
		cs, err := p.parseComplex()
		if err != nil {
			return nil, err
		}
		if len(cs.parts) == 0 {
			return nil, fmt.Errorf("puppeteer: empty selector in %q", p.src)
		}
		sel.groups = append(sel.groups, cs)
		p.skipSpace()
		if p.pos >= len(p.src) {
			break
		}
		if p.src[p.pos] == ',' {
			p.pos++
			continue
		}
		return nil, fmt.Errorf("puppeteer: unexpected %q in selector %q", p.src[p.pos], p.src)
	}
	return sel, nil
}

func (p *selParser) parseComplex() (complexSelector, error) {
	var cs complexSelector
	for {
		p.skipSpace()
		if p.pos >= len(p.src) || p.src[p.pos] == ',' {
			break
		}
		comp, err := p.parseCompound()
		if err != nil {
			return cs, err
		}
		cs.parts = append(cs.parts, comp)

		// Determine the combinator that follows, if any.
		hadSpace := p.skipSpace()
		if p.pos >= len(p.src) || p.src[p.pos] == ',' {
			break
		}
		switch p.src[p.pos] {
		case '>', '+', '~':
			cs.combs = append(cs.combs, p.src[p.pos])
			p.pos++
		default:
			if hadSpace {
				cs.combs = append(cs.combs, ' ')
			} else {
				return cs, fmt.Errorf("puppeteer: unexpected %q in selector %q", p.src[p.pos], p.src)
			}
		}
	}
	if len(cs.combs) >= len(cs.parts) {
		return cs, fmt.Errorf("puppeteer: trailing combinator in selector %q", p.src)
	}
	return cs, nil
}

func (p *selParser) parseCompound() (compoundSelector, error) {
	var c compoundSelector
	started := false
	for p.pos < len(p.src) {
		ch := p.src[p.pos]
		switch {
		case ch == '*':
			c.tag = "*"
			p.pos++
		case isSelNameStart(ch):
			c.tag = strings.ToLower(p.readName())
		case ch == '#':
			p.pos++
			c.id = p.readName()
		case ch == '.':
			p.pos++
			c.classes = append(c.classes, p.readName())
		case ch == '[':
			a, err := p.parseAttr()
			if err != nil {
				return c, err
			}
			c.attrs = append(c.attrs, a)
		case ch == ':':
			ps, err := p.parsePseudo()
			if err != nil {
				return c, err
			}
			c.pseudos = append(c.pseudos, ps)
		default:
			if !started {
				return c, fmt.Errorf("puppeteer: unexpected %q in selector %q", ch, p.src)
			}
			return c, nil
		}
		started = true
	}
	if !started {
		return c, fmt.Errorf("puppeteer: empty compound selector in %q", p.src)
	}
	return c, nil
}

func (p *selParser) parseAttr() (attrSelector, error) {
	var a attrSelector
	p.pos++ // consume '['
	p.skipSpace()
	a.name = strings.ToLower(p.readName())
	if a.name == "" {
		return a, fmt.Errorf("puppeteer: empty attribute name in selector %q", p.src)
	}
	p.skipSpace()
	if p.pos >= len(p.src) {
		return a, fmt.Errorf("puppeteer: unterminated attribute selector in %q", p.src)
	}
	if p.src[p.pos] == ']' {
		p.pos++
		return a, nil
	}
	// Operator.
	switch p.src[p.pos] {
	case '=':
		a.op = "="
		p.pos++
	case '^', '$', '*', '~', '|':
		if p.pos+1 < len(p.src) && p.src[p.pos+1] == '=' {
			a.op = string(p.src[p.pos]) + "="
			p.pos += 2
		} else {
			return a, fmt.Errorf("puppeteer: invalid attribute operator in %q", p.src)
		}
	default:
		return a, fmt.Errorf("puppeteer: invalid attribute selector in %q", p.src)
	}
	p.skipSpace()
	a.val = p.readAttrValue()
	p.skipSpace()
	if p.pos >= len(p.src) || p.src[p.pos] != ']' {
		return a, fmt.Errorf("puppeteer: unterminated attribute selector in %q", p.src)
	}
	p.pos++ // consume ']'
	return a, nil
}

func (p *selParser) readAttrValue() string {
	if p.pos < len(p.src) && (p.src[p.pos] == '"' || p.src[p.pos] == '\'') {
		quote := p.src[p.pos]
		p.pos++
		start := p.pos
		for p.pos < len(p.src) && p.src[p.pos] != quote {
			p.pos++
		}
		v := p.src[start:p.pos]
		if p.pos < len(p.src) {
			p.pos++
		}
		return v
	}
	start := p.pos
	for p.pos < len(p.src) && p.src[p.pos] != ']' && !isSpace(p.src[p.pos]) {
		p.pos++
	}
	return p.src[start:p.pos]
}

func (p *selParser) parsePseudo() (pseudoSelector, error) {
	p.pos++ // consume ':'
	if p.pos < len(p.src) && p.src[p.pos] == ':' {
		p.pos++ // tolerate ::pseudo-element syntax
	}
	name := strings.ToLower(p.readName())
	ps := pseudoSelector{name: name}
	switch name {
	case "first-child":
		ps.a, ps.b = 0, 1
		ps.name = "nth-child"
	case "last-child":
		ps.name = "last-child"
	case "nth-child":
		if p.pos >= len(p.src) || p.src[p.pos] != '(' {
			return ps, fmt.Errorf("puppeteer: :nth-child requires an argument in %q", p.src)
		}
		p.pos++ // '('
		start := p.pos
		for p.pos < len(p.src) && p.src[p.pos] != ')' {
			p.pos++
		}
		arg := strings.TrimSpace(p.src[start:p.pos])
		if p.pos >= len(p.src) {
			return ps, fmt.Errorf("puppeteer: unterminated :nth-child in %q", p.src)
		}
		p.pos++ // ')'
		a, b, err := parseNth(arg)
		if err != nil {
			return ps, err
		}
		ps.a, ps.b = a, b
	default:
		return ps, fmt.Errorf("puppeteer: unsupported pseudo-class :%s", name)
	}
	return ps, nil
}

// parseNth parses an An+B microsyntax value (e.g. "odd", "2n+1", "-n+3", "4").
func parseNth(s string) (int, int, error) {
	s = strings.ToLower(strings.ReplaceAll(s, " ", ""))
	switch s {
	case "odd":
		return 2, 1, nil
	case "even":
		return 2, 0, nil
	case "":
		return 0, 0, fmt.Errorf("puppeteer: empty :nth-child argument")
	}
	nIdx := strings.IndexByte(s, 'n')
	if nIdx < 0 {
		b, err := strconv.Atoi(s)
		if err != nil {
			return 0, 0, fmt.Errorf("puppeteer: invalid :nth-child argument %q", s)
		}
		return 0, b, nil
	}
	// a coefficient.
	aStr := s[:nIdx]
	var a int
	switch aStr {
	case "", "+":
		a = 1
	case "-":
		a = -1
	default:
		v, err := strconv.Atoi(aStr)
		if err != nil {
			return 0, 0, fmt.Errorf("puppeteer: invalid :nth-child coefficient %q", s)
		}
		a = v
	}
	// b constant.
	bStr := s[nIdx+1:]
	if bStr == "" {
		return a, 0, nil
	}
	b, err := strconv.Atoi(bStr)
	if err != nil {
		return 0, 0, fmt.Errorf("puppeteer: invalid :nth-child constant %q", s)
	}
	return a, b, nil
}

func (p *selParser) readName() string {
	start := p.pos
	for p.pos < len(p.src) && isSelNameChar(p.src[p.pos]) {
		p.pos++
	}
	return p.src[start:p.pos]
}

// skipSpace advances over whitespace and reports whether any was skipped.
func (p *selParser) skipSpace() bool {
	start := p.pos
	for p.pos < len(p.src) && isSpace(p.src[p.pos]) {
		p.pos++
	}
	return p.pos > start
}

func isSelNameStart(c byte) bool {
	return isASCIILetter(c) || c == '_' || c == '-' || c >= 0x80
}

func isSelNameChar(c byte) bool {
	return isASCIILetter(c) || (c >= '0' && c <= '9') || c == '_' || c == '-' || c >= 0x80
}

// ---- Matching ----------------------------------------------------------

// matchNode reports whether n matches any group of the selector.
func (s *selector) matchNode(n *Node) bool {
	if n == nil || n.Type != ElementNode {
		return false
	}
	for i := range s.groups {
		if matchComplex(s.groups[i], len(s.groups[i].parts)-1, n) {
			return true
		}
	}
	return false
}

// matchComplex matches parts[0..idx] of a complex selector against n, where n
// must match parts[idx].
func matchComplex(cs complexSelector, idx int, n *Node) bool {
	if !matchCompound(cs.parts[idx], n) {
		return false
	}
	if idx == 0 {
		return true
	}
	comb := cs.combs[idx-1]
	switch comb {
	case ' ':
		for a := n.Parent; a != nil && a.Type == ElementNode; a = a.Parent {
			if matchComplex(cs, idx-1, a) {
				return true
			}
		}
	case '>':
		if p := n.Parent; p != nil && p.Type == ElementNode && matchComplex(cs, idx-1, p) {
			return true
		}
	case '+':
		if prev := prevElement(n); prev != nil && matchComplex(cs, idx-1, prev) {
			return true
		}
	case '~':
		for prev := prevElement(n); prev != nil; prev = prevElement(prev) {
			if matchComplex(cs, idx-1, prev) {
				return true
			}
		}
	}
	return false
}

// matchCompound tests a single compound selector against one element node.
func matchCompound(c compoundSelector, n *Node) bool {
	if n.Type != ElementNode {
		return false
	}
	if c.tag != "" && c.tag != "*" && c.tag != n.Data {
		return false
	}
	if c.id != "" {
		if id, _ := n.attr("id"); id != c.id {
			return false
		}
	}
	for _, cl := range c.classes {
		if !hasClassNode(n, cl) {
			return false
		}
	}
	for _, a := range c.attrs {
		if !matchAttr(a, n) {
			return false
		}
	}
	for _, ps := range c.pseudos {
		if !matchPseudo(ps, n) {
			return false
		}
	}
	return true
}

func hasClassNode(n *Node, class string) bool {
	for _, c := range n.classes() {
		if c == class {
			return true
		}
	}
	return false
}

func matchAttr(a attrSelector, n *Node) bool {
	v, ok := n.attr(a.name)
	if !ok {
		return false
	}
	switch a.op {
	case "":
		return true
	case "=":
		return v == a.val
	case "^=":
		return a.val != "" && strings.HasPrefix(v, a.val)
	case "$=":
		return a.val != "" && strings.HasSuffix(v, a.val)
	case "*=":
		return a.val != "" && strings.Contains(v, a.val)
	case "~=":
		if a.val == "" {
			return false
		}
		for _, f := range strings.Fields(v) {
			if f == a.val {
				return true
			}
		}
		return false
	case "|=":
		return v == a.val || strings.HasPrefix(v, a.val+"-")
	}
	return false
}

func matchPseudo(ps pseudoSelector, n *Node) bool {
	switch ps.name {
	case "nth-child":
		idx := elementIndex(n) // 1-based
		if idx == 0 {
			return false
		}
		return nthMatch(ps.a, ps.b, idx)
	case "last-child":
		return nextElement(n) == nil && n.Parent != nil
	}
	return false
}

// elementIndex returns the 1-based position of n among its element siblings.
func elementIndex(n *Node) int {
	if n.Parent == nil {
		return 0
	}
	i := 0
	for _, s := range n.Parent.Children {
		if s.Type != ElementNode {
			continue
		}
		i++
		if s == n {
			return i
		}
	}
	return 0
}

// nthMatch reports whether index (1-based) satisfies a*k + b for some k >= 0.
func nthMatch(a, b, index int) bool {
	if a == 0 {
		return index == b
	}
	k := index - b
	if k%a != 0 {
		return false
	}
	return k/a >= 0
}

// ---- Querying ----------------------------------------------------------

// queryAll returns all element descendants of root that match the selector, in
// document order. root itself is not tested.
func (s *selector) queryAll(root *Node) []*Node {
	var out []*Node
	var walk func(n *Node)
	walk = func(n *Node) {
		for _, c := range n.Children {
			if c.Type == ElementNode {
				if s.matchNode(c) {
					out = append(out, c)
				}
				walk(c)
			}
		}
	}
	walk(root)
	return out
}

// queryFirst returns the first matching descendant of root, or nil.
func (s *selector) queryFirst(root *Node) *Node {
	var found *Node
	var walk func(n *Node) bool
	walk = func(n *Node) bool {
		for _, c := range n.Children {
			if c.Type != ElementNode {
				continue
			}
			if s.matchNode(c) {
				found = c
				return true
			}
			if walk(c) {
				return true
			}
		}
		return false
	}
	walk(root)
	return found
}
