package puppeteer

import "strings"

// headElements are the tag names that belong in <head> during the HTML tree
// construction "in head" phase. When a document omits an explicit <head>, a
// leading run of these elements is hoisted into a synthesized head, mirroring
// how a browser builds the DOM.
var headElements = map[string]bool{
	"base": true, "basefont": true, "bgsound": true, "link": true,
	"meta": true, "noscript": true, "script": true, "style": true,
	"template": true, "title": true,
}

// NormalizedHTML serializes the page's document the way a browser would report
// document.documentElement.outerHTML — i.e. what Puppeteer's page.content()
// returns. Any leading <!DOCTYPE> is preserved, a single <html> element is
// guaranteed, and missing <head>/<body> elements are synthesized so that, for
// example, a document set from the fragment "<div>hello</div>" serializes as
// "<html><head></head><body><div>hello</div></body></html>".
//
// Unlike HTML, which serializes the parse tree verbatim, NormalizedHTML applies
// the head/body structural normalization that a live DOM performs. It does not
// mutate the underlying document. When the page has no document it returns "".
func (p *Page) NormalizedHTML() string {
	if p.doc == nil {
		return ""
	}
	var sb strings.Builder

	var htmlEl *Node
	for _, c := range p.doc.Children {
		switch {
		case c.Type == DoctypeNode:
			c.render(&sb)
		case c.Type == ElementNode && c.Data == "html" && htmlEl == nil:
			htmlEl = c
		}
	}

	head := &Node{Type: ElementNode, Data: "head"}
	body := &Node{Type: ElementNode, Data: "body"}

	var htmlAttr []Attribute
	var source []*Node
	if htmlEl != nil {
		htmlAttr = htmlEl.Attr
		for _, c := range htmlEl.Children {
			switch {
			case c.Type == ElementNode && c.Data == "head":
				head.Children = append(head.Children, c.Children...)
			case c.Type == ElementNode && c.Data == "body":
				body.Children = append(body.Children, c.Children...)
			default:
				source = append(source, c)
			}
		}
	} else {
		for _, c := range p.doc.Children {
			if c.Type == DoctypeNode {
				continue
			}
			source = append(source, c)
		}
	}

	inHead := len(head.Children) == 0 && len(body.Children) == 0
	for _, c := range source {
		if inHead && c.Type == ElementNode && headElements[c.Data] {
			head.Children = append(head.Children, c)
			continue
		}
		if inHead && c.Type == TextNode && strings.TrimSpace(c.Data) == "" {
			// Inter-element whitespace before body content is dropped in head.
			continue
		}
		inHead = false
		body.Children = append(body.Children, c)
	}

	htmlNode := &Node{Type: ElementNode, Data: "html", Attr: htmlAttr}
	htmlNode.Children = []*Node{head, body}
	htmlNode.render(&sb)
	return sb.String()
}
