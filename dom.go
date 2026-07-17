package puppeteer

import (
	"strings"
)

// NodeType identifies the kind of a DOM Node.
type NodeType int

const (
	// DocumentNode is the synthetic root of a parsed document.
	DocumentNode NodeType = iota
	// ElementNode is an HTML element such as <div>.
	ElementNode
	// TextNode is a run of character data.
	TextNode
	// CommentNode is an HTML comment (<!-- ... -->).
	CommentNode
	// DoctypeNode is a <!DOCTYPE ...> declaration.
	DoctypeNode
)

// Attribute is a single name/value pair on an element. Attribute names are
// normalized to lower case during parsing.
type Attribute struct {
	Name  string
	Value string
}

// Node is a single node in the DOM tree produced by the parser. The tree is a
// simple parent/children model: every Node (except the document root) has a
// Parent, and Children preserves document order.
//
// For an ElementNode, Data holds the (lower-cased) tag name. For a TextNode,
// Data holds the raw text. For a CommentNode, Data holds the comment body
// (without the <!-- --> delimiters). For a DoctypeNode, Data holds the doctype
// name.
type Node struct {
	Type     NodeType
	Data     string
	Attr     []Attribute
	Parent   *Node
	Children []*Node
}

// AppendChild adds c as the last child of n and sets c.Parent.
func (n *Node) AppendChild(c *Node) {
	c.Parent = n
	n.Children = append(n.Children, c)
}

// attr returns the value of the named attribute and whether it was present.
func (n *Node) attr(name string) (string, bool) {
	name = strings.ToLower(name)
	for _, a := range n.Attr {
		if a.Name == name {
			return a.Value, true
		}
	}
	return "", false
}

// classes returns the whitespace-separated tokens of the class attribute.
func (n *Node) classes() []string {
	v, ok := n.attr("class")
	if !ok {
		return nil
	}
	return strings.Fields(v)
}

// text recursively concatenates the text content of n and its descendants.
func (n *Node) text(sb *strings.Builder) {
	switch n.Type {
	case TextNode:
		sb.WriteString(n.Data)
	case ElementNode, DocumentNode:
		for _, c := range n.Children {
			c.text(sb)
		}
	}
}

// voidElements are HTML elements that never have children or an end tag.
var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true,
	"param": true, "source": true, "track": true, "wbr": true,
}

// rawTextElements have their content parsed literally (no nested markup).
var rawTextElements = map[string]bool{
	"script": true, "style": true,
}

// escapableRawText elements decode character references but do not parse tags.
var escapableRawText = map[string]bool{
	"textarea": true, "title": true,
}

// render serializes n and its subtree back into HTML, appending to sb.
func (n *Node) render(sb *strings.Builder) {
	switch n.Type {
	case DocumentNode:
		for _, c := range n.Children {
			c.render(sb)
		}
	case TextNode:
		if p := n.Parent; p != nil && p.Type == ElementNode &&
			(rawTextElements[p.Data] || escapableRawText[p.Data]) {
			sb.WriteString(n.Data)
		} else {
			sb.WriteString(escapeText(n.Data))
		}
	case CommentNode:
		sb.WriteString("<!--")
		sb.WriteString(n.Data)
		sb.WriteString("-->")
	case DoctypeNode:
		sb.WriteString("<!DOCTYPE ")
		sb.WriteString(n.Data)
		sb.WriteString(">")
	case ElementNode:
		sb.WriteByte('<')
		sb.WriteString(n.Data)
		for _, a := range n.Attr {
			sb.WriteByte(' ')
			sb.WriteString(a.Name)
			sb.WriteString(`="`)
			sb.WriteString(escapeAttr(a.Value))
			sb.WriteByte('"')
		}
		sb.WriteByte('>')
		if voidElements[n.Data] {
			return
		}
		for _, c := range n.Children {
			c.render(sb)
		}
		sb.WriteString("</")
		sb.WriteString(n.Data)
		sb.WriteByte('>')
	}
}

// renderChildren serializes only the children of n (inner HTML).
func (n *Node) renderChildren(sb *strings.Builder) {
	for _, c := range n.Children {
		c.render(sb)
	}
}

var textEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

func escapeText(s string) string { return textEscaper.Replace(s) }

var attrEscaper = strings.NewReplacer("&", "&amp;", `"`, "&quot;")

func escapeAttr(s string) string { return attrEscaper.Replace(s) }

// attributesMap returns the element's attributes as a sorted-key map.
func (n *Node) attributesMap() map[string]string {
	m := make(map[string]string, len(n.Attr))
	for _, a := range n.Attr {
		m[a.Name] = a.Value
	}
	return m
}
