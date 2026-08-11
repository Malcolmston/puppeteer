package puppeteer

// Parse turns an HTML document string into a DOM tree rooted at a DocumentNode.
// It never returns an error: malformed input is recovered pragmatically (the
// way browsers do) rather than rejected.
func Parse(html string) *Node {
	p := &parser{tk: newTokenizer(html)}
	return p.parse()
}

// autoCloseOnOpen maps an opening tag to the set of currently-open sibling tags
// that it implicitly closes. This reproduces the most common HTML "optional end
// tag" behavior (e.g. <li> closing a previous <li>) without a full HTML5 tree
// construction algorithm.
var autoCloseOnOpen = map[string]map[string]bool{
	"li":       {"li": true},
	"option":   {"option": true},
	"optgroup": {"optgroup": true, "option": true},
	"dd":       {"dd": true, "dt": true},
	"dt":       {"dd": true, "dt": true},
}

// tableSections are the row-group and heading elements that live directly under
// <table>. Opening one clears the stack back to the table.
var tableSections = map[string]bool{
	"thead": true, "tbody": true, "tfoot": true, "caption": true, "colgroup": true,
}

// tableContexts are the elements whose "current node" triggers foster parenting
// of a non-table element (a table cell is deliberately excluded — its content is
// ordinary flow content).
var tableContexts = map[string]bool{
	"table": true, "thead": true, "tbody": true, "tfoot": true, "tr": true,
}

// tableStructural is the set of elements that are legal in table context and are
// therefore never foster-parented.
var tableStructural = map[string]bool{
	"table": true, "caption": true, "colgroup": true, "col": true,
	"thead": true, "tbody": true, "tfoot": true, "tr": true, "td": true, "th": true,
	"script": true, "style": true, "template": true,
}

// blockElements close an open <p> when they start.
var blockElements = map[string]bool{
	"address": true, "article": true, "aside": true, "blockquote": true,
	"div": true, "dl": true, "fieldset": true, "figure": true, "footer": true,
	"form": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true,
	"h6": true, "header": true, "hr": true, "main": true, "nav": true,
	"ol": true, "p": true, "pre": true, "section": true, "table": true,
	"ul": true,
}

type parser struct {
	tk    *tokenizer
	doc   *Node
	stack []*Node // open element stack; stack[0] is the document
}

func (p *parser) top() *Node { return p.stack[len(p.stack)-1] }

func (p *parser) parse() *Node {
	p.doc = &Node{Type: DocumentNode}
	p.stack = []*Node{p.doc}

	for {
		tok, ok := p.tk.next()
		if !ok {
			break
		}
		switch tok.kind {
		case tokText:
			if tok.data == "" {
				continue
			}
			p.top().AppendChild(&Node{Type: TextNode, Data: tok.data})
		case tokComment:
			p.top().AppendChild(&Node{Type: CommentNode, Data: tok.data})
		case tokDoctype:
			p.doc.AppendChild(&Node{Type: DoctypeNode, Data: tok.data})
		case tokStartTag:
			p.startTag(tok)
		case tokEndTag:
			p.endTag(tok.data)
		}
	}
	return p.doc
}

func (p *parser) startTag(tok token) {
	name := tok.data

	// A stray <col> outside a table is ignored by the HTML tree constructor
	// (it is meaningful only inside a <colgroup>/<table>).
	if name == "col" && !p.inTable() {
		return
	}

	// Table tree construction: clear the stack to the right context and, where
	// the markup omits them, synthesise the implied row-group/row wrappers.
	p.prepareTableInsertion(name)

	// Implicit end tags for sibling elements (e.g. <li> after <li>).
	if closes, ok := autoCloseOnOpen[name]; ok {
		for len(p.stack) > 1 && closes[p.top().Data] {
			p.pop()
		}
	}
	// Block elements implicitly close an open <p>.
	if blockElements[name] && p.top().Type == ElementNode && p.top().Data == "p" {
		p.pop()
	}

	el := &Node{Type: ElementNode, Data: name, Attr: tok.attr}

	// Foster parenting: a non-table element started while the current node is a
	// table (but not a cell) is inserted immediately before that table instead
	// of inside it, then opened as usual.
	if tableContexts[p.top().Data] && !tableStructural[name] {
		p.fosterInsert(el)
	} else {
		p.top().AppendChild(el)
	}

	// Void elements never have children or an end tag. A trailing "/" on a
	// non-void, non-foreign HTML start tag is ignored per the spec: <span/> is
	// an ordinary open <span>, not a self-closed one, so following content
	// nests inside it.
	if voidElements[name] {
		return
	}

	if rawTextElements[name] || escapableRawText[name] {
		raw := p.tk.readRawText(name)
		if escapableRawText[name] {
			raw = decodeEntities(raw)
		}
		if raw != "" {
			el.AppendChild(&Node{Type: TextNode, Data: raw})
		}
		// Consume the matching end tag if present.
		if tok2, ok := p.tk.next(); ok {
			if tok2.kind != tokEndTag || tok2.data != name {
				// Not the expected end tag; reprocess generically.
				p.stack = append(p.stack, el)
				p.reprocess(tok2)
			}
		}
		return
	}

	p.stack = append(p.stack, el)
}

// reprocess handles a token that was read speculatively but not consumed.
func (p *parser) reprocess(tok token) {
	switch tok.kind {
	case tokText:
		if tok.data != "" {
			p.top().AppendChild(&Node{Type: TextNode, Data: tok.data})
		}
	case tokComment:
		p.top().AppendChild(&Node{Type: CommentNode, Data: tok.data})
	case tokStartTag:
		p.startTag(tok)
	case tokEndTag:
		p.endTag(tok.data)
	}
}

func (p *parser) endTag(name string) {
	// Find the nearest matching open element and pop up to it.
	for i := len(p.stack) - 1; i >= 1; i-- {
		if p.stack[i].Data == name {
			p.stack = p.stack[:i]
			return
		}
	}
	// A </p> with no <p> open is a special case: the HTML tree constructor
	// inserts (and immediately closes) an empty <p>. Matching parse5/cheerio,
	// this only surfaces once the insertion point already holds an element —
	// a run of bare text before the stray </p> produces nothing.
	if name == "p" && p.topHasElementChild() {
		p.top().AppendChild(&Node{Type: ElementNode, Data: "p"})
		return
	}
	// No matching open element: ignore the stray end tag.
}

// prepareTableInsertion applies the table-specific stack manipulations before a
// start tag is inserted: it clears the stack to the appropriate table context
// and synthesises an implied <tbody>/<tr> when the markup omits them.
func (p *parser) prepareTableInsertion(name string) {
	switch {
	case tableSections[name]:
		// A row group / caption / colgroup clears back to the table.
		p.clearToTable()
	case name == "tr":
		// A row clears any open cells/rows, then needs a row-group parent.
		p.clearOpen("td", "th", "tr")
		if p.top().Type == ElementNode && p.top().Data == "table" {
			p.pushSynthesized("tbody")
		}
	case name == "td" || name == "th":
		// A cell clears any open cell, then needs a row parent.
		p.clearOpen("td", "th")
		if p.top().Type == ElementNode && p.top().Data != "tr" && p.inTable() {
			p.clearToTable()
			if p.top().Data == "table" {
				p.pushSynthesized("tbody")
			}
			p.pushSynthesized("tr")
		}
	}
}

// clearOpen pops the stack while the current node is one of the named elements.
func (p *parser) clearOpen(names ...string) {
	for len(p.stack) > 1 {
		d := p.top().Data
		hit := false
		for _, n := range names {
			if d == n {
				hit = true
				break
			}
		}
		if !hit {
			return
		}
		p.pop()
	}
}

// clearToTable pops the stack until the current node is a <table> (or the stack
// bottom), removing any open rows, cells and row groups.
func (p *parser) clearToTable() {
	for len(p.stack) > 1 {
		d := p.top().Data
		if d == "table" {
			return
		}
		if d == "td" || d == "th" || d == "tr" || tableSections[d] {
			p.pop()
			continue
		}
		return
	}
}

// pushSynthesized appends a parser-inserted element of the given name to the
// current node and opens it.
func (p *parser) pushSynthesized(name string) {
	el := &Node{Type: ElementNode, Data: name}
	p.top().AppendChild(el)
	p.stack = append(p.stack, el)
}

// fosterInsert places el immediately before the nearest open <table> in that
// table's parent, then opens el, implementing foster parenting.
func (p *parser) fosterInsert(el *Node) {
	for i := len(p.stack) - 1; i >= 1; i-- {
		n := p.stack[i]
		if n.Type == ElementNode && n.Data == "table" && n.Parent != nil {
			parent := n.Parent
			idx := indexOf(parent.Children, n)
			el.Parent = parent
			parent.Children = append(parent.Children, nil)
			copy(parent.Children[idx+1:], parent.Children[idx:])
			parent.Children[idx] = el
			return
		}
	}
	p.top().AppendChild(el)
}

// inTable reports whether a <table> is currently open on the stack.
func (p *parser) inTable() bool {
	for i := len(p.stack) - 1; i >= 1; i-- {
		if p.stack[i].Type == ElementNode && p.stack[i].Data == "table" {
			return true
		}
	}
	return false
}

// topHasElementChild reports whether the current node already has at least one
// element child.
func (p *parser) topHasElementChild() bool {
	for _, c := range p.top().Children {
		if c.Type == ElementNode {
			return true
		}
	}
	return false
}

func (p *parser) pop() {
	if len(p.stack) > 1 {
		p.stack = p.stack[:len(p.stack)-1]
	}
}
