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
	"tr":       {"tr": true, "td": true, "th": true},
	"td":       {"td": true, "th": true},
	"th":       {"td": true, "th": true},
	"thead":    {"td": true, "th": true, "tr": true},
	"tbody":    {"td": true, "th": true, "tr": true},
	"tfoot":    {"td": true, "th": true, "tr": true},
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
	p.top().AppendChild(el)

	if voidElements[name] || tok.selfClosing {
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
	// No matching open element: ignore the stray end tag.
}

func (p *parser) pop() {
	if len(p.stack) > 1 {
		p.stack = p.stack[:len(p.stack)-1]
	}
}
