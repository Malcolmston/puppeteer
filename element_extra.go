package puppeteer

import "strings"

// This file adds DOM-traversal and attribute conveniences to Element that the
// Node.js DOM (and thus Puppeteer's ElementHandle) exposes but the base handle
// here did not: attribute introspection, the data-* dataset, and the jQuery-like
// sibling/ancestor collection helpers.

// HasAttribute reports whether the element carries the named attribute,
// regardless of its value. Names are matched case-insensitively.
func (e *Element) HasAttribute(name string) bool {
	_, ok := e.node.attr(name)
	return ok
}

// AttributeNames returns the names of the element's attributes in source order.
func (e *Element) AttributeNames() []string {
	names := make([]string, 0, len(e.node.Attr))
	for _, a := range e.node.Attr {
		names = append(names, a.Name)
	}
	return names
}

// Dataset returns the element's data-* attributes as a map keyed by the DOM
// dataset name: the "data-" prefix is stripped and each "-x" sequence becomes an
// upper-cased letter, so data-user-id becomes userId. This mirrors the
// HTMLElement.dataset property. Attributes without the data- prefix are omitted.
func (e *Element) Dataset() map[string]string {
	out := map[string]string{}
	for _, a := range e.node.Attr {
		if !strings.HasPrefix(a.Name, "data-") {
			continue
		}
		key := ppDatasetKey(strings.TrimPrefix(a.Name, "data-"))
		if key == "" {
			continue
		}
		out[key] = a.Value
	}
	return out
}

// ppDatasetKey converts a data-* attribute suffix into its DOM dataset camelCase
// form: "user-id" -> "userId", "" -> "".
func ppDatasetKey(s string) string {
	var b strings.Builder
	upper := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '-' {
			upper = true
			continue
		}
		if upper && c >= 'a' && c <= 'z' {
			b.WriteByte(c - ('a' - 'A'))
		} else {
			b.WriteByte(c)
		}
		upper = false
	}
	return b.String()
}

// Siblings returns the element's sibling elements in document order, excluding
// the element itself. Text and comment nodes are skipped.
func (e *Element) Siblings() []*Element {
	if e.node.Parent == nil {
		return nil
	}
	var out []*Element
	for _, s := range e.node.Parent.Children {
		if s.Type == ElementNode && s != e.node {
			out = append(out, wrapElement(s))
		}
	}
	return out
}

// NextAll returns every following sibling element in document order.
func (e *Element) NextAll() []*Element {
	var out []*Element
	for n := nextElement(e.node); n != nil; n = nextElement(n) {
		out = append(out, wrapElement(n))
	}
	return out
}

// PrevAll returns every preceding sibling element in document order (the
// nearest preceding sibling last).
func (e *Element) PrevAll() []*Element {
	var rev []*Element
	for n := prevElement(e.node); n != nil; n = prevElement(n) {
		rev = append(rev, wrapElement(n))
	}
	// rev is nearest-first; reverse to document order.
	out := make([]*Element, len(rev))
	for i, el := range rev {
		out[len(rev)-1-i] = el
	}
	return out
}

// Ancestors returns the element's ancestor elements from the immediate parent up
// to the document root's outermost element, nearest first. The document node
// itself is not included.
func (e *Element) Ancestors() []*Element {
	var out []*Element
	for n := e.node.Parent; n != nil && n.Type == ElementNode; n = n.Parent {
		out = append(out, wrapElement(n))
	}
	return out
}

// IsEmpty reports whether the element has no child elements, no comments and no
// non-whitespace text, approximating the CSS :empty pseudo-class with the
// Selectors Level 4 relaxation that whitespace-only text is treated as empty.
func (e *Element) IsEmpty() bool {
	for _, c := range e.node.Children {
		switch c.Type {
		case ElementNode, CommentNode:
			return false
		case TextNode:
			if strings.TrimSpace(c.Data) != "" {
				return false
			}
		}
	}
	return true
}
