package puppeteer

import (
	"strings"
)

// Element is a handle to an ElementNode in a parsed document. It offers the
// convenience accessors and traversal helpers familiar from Puppeteer's
// ElementHandle, adapted to a static (non-JavaScript) DOM.
type Element struct {
	node *Node
}

// wrapElement returns an *Element for n, or nil if n is nil or not an element.
func wrapElement(n *Node) *Element {
	if n == nil || n.Type != ElementNode {
		return nil
	}
	return &Element{node: n}
}

// wrapElements wraps a slice of nodes, skipping non-elements.
func wrapElements(nodes []*Node) []*Element {
	out := make([]*Element, 0, len(nodes))
	for _, n := range nodes {
		if e := wrapElement(n); e != nil {
			out = append(out, e)
		}
	}
	return out
}

// Node returns the underlying DOM node.
func (e *Element) Node() *Node { return e.node }

// TagName returns the element's lower-cased tag name.
func (e *Element) TagName() string { return e.node.Data }

// TextContent returns the concatenated text of the element and its descendants.
func (e *Element) TextContent() string {
	var sb strings.Builder
	e.node.text(&sb)
	return sb.String()
}

// InnerText is a convenience alias for TextContent with surrounding whitespace
// trimmed and internal runs collapsed to single spaces.
func (e *Element) InnerText() string {
	return strings.Join(strings.Fields(e.TextContent()), " ")
}

// InnerHTML returns the serialized markup of the element's children.
func (e *Element) InnerHTML() string {
	var sb strings.Builder
	e.node.renderChildren(&sb)
	return sb.String()
}

// OuterHTML returns the serialized markup of the element including itself.
func (e *Element) OuterHTML() string {
	var sb strings.Builder
	e.node.render(&sb)
	return sb.String()
}

// Attr returns the value of the named attribute and whether it exists.
func (e *Element) Attr(name string) (string, bool) { return e.node.attr(name) }

// AttrOr returns the named attribute's value or def when it is absent.
func (e *Element) AttrOr(name, def string) string {
	if v, ok := e.node.attr(name); ok {
		return v
	}
	return def
}

// Attributes returns all attributes as a name/value map.
func (e *Element) Attributes() map[string]string { return e.node.attributesMap() }

// ID returns the element's id attribute.
func (e *Element) ID() string { return e.AttrOr("id", "") }

// ClassList returns the element's class tokens.
func (e *Element) ClassList() []string { return e.node.classes() }

// HasClass reports whether the element carries the given class token.
func (e *Element) HasClass(class string) bool {
	for _, c := range e.node.classes() {
		if c == class {
			return true
		}
	}
	return false
}

// Children returns the element's child elements (text and comments excluded).
func (e *Element) Children() []*Element { return wrapElements(e.node.Children) }

// Parent returns the element's parent element, or nil.
func (e *Element) Parent() *Element { return wrapElement(e.node.Parent) }

// Next returns the next sibling element, or nil.
func (e *Element) Next() *Element { return wrapElement(nextElement(e.node)) }

// Prev returns the previous sibling element, or nil.
func (e *Element) Prev() *Element { return wrapElement(prevElement(e.node)) }

// Closest returns the nearest ancestor (including the element itself) that
// matches the selector, or nil.
func (e *Element) Closest(selector string) (*Element, error) {
	sel, err := compileSelector(selector)
	if err != nil {
		return nil, err
	}
	for n := e.node; n != nil && n.Type == ElementNode; n = n.Parent {
		if sel.matchNode(n) {
			return wrapElement(n), nil
		}
	}
	return nil, nil
}

// Matches reports whether the element matches the selector.
func (e *Element) Matches(selector string) (bool, error) {
	sel, err := compileSelector(selector)
	if err != nil {
		return false, err
	}
	return sel.matchNode(e.node), nil
}

// QuerySelector returns the first descendant matching the selector, or nil.
func (e *Element) QuerySelector(selector string) (*Element, error) {
	sel, err := compileSelector(selector)
	if err != nil {
		return nil, err
	}
	if n := sel.queryFirst(e.node); n != nil {
		return wrapElement(n), nil
	}
	return nil, nil
}

// QuerySelectorAll returns all descendants matching the selector.
func (e *Element) QuerySelectorAll(selector string) ([]*Element, error) {
	sel, err := compileSelector(selector)
	if err != nil {
		return nil, err
	}
	return wrapElements(sel.queryAll(e.node)), nil
}

// nextElement returns the following sibling element of n, or nil.
func nextElement(n *Node) *Node {
	if n == nil || n.Parent == nil {
		return nil
	}
	sibs := n.Parent.Children
	idx := indexOf(sibs, n)
	for i := idx + 1; i < len(sibs); i++ {
		if sibs[i].Type == ElementNode {
			return sibs[i]
		}
	}
	return nil
}

// prevElement returns the preceding sibling element of n, or nil.
func prevElement(n *Node) *Node {
	if n == nil || n.Parent == nil {
		return nil
	}
	sibs := n.Parent.Children
	idx := indexOf(sibs, n)
	for i := idx - 1; i >= 0; i-- {
		if sibs[i].Type == ElementNode {
			return sibs[i]
		}
	}
	return nil
}

func indexOf(nodes []*Node, target *Node) int {
	for i, n := range nodes {
		if n == target {
			return i
		}
	}
	return -1
}
