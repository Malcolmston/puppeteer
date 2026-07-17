package puppeteer

import (
	"testing"
)

func TestElementHelpers(t *testing.T) {
	doc := Parse(`<div id="wrap" class="a b"><span title="t">  Hello   world  </span></div>`)
	div := wrapElement(firstElement(doc, "div"))

	if div.Node() == nil || div.Node().Data != "div" {
		t.Error("Node() failed")
	}
	if div.ID() != "wrap" {
		t.Errorf("ID = %q", div.ID())
	}
	attrs := div.Attributes()
	if attrs["id"] != "wrap" || attrs["class"] != "a b" {
		t.Errorf("Attributes = %v", attrs)
	}
	if div.AttrOr("missing", "def") != "def" {
		t.Error("AttrOr default failed")
	}
	kids := div.Children()
	if len(kids) != 1 || kids[0].TagName() != "span" {
		t.Errorf("Children = %v", kids)
	}
	span := kids[0]
	if span.InnerText() != "Hello world" {
		t.Errorf("InnerText = %q", span.InnerText())
	}
	if v, ok := span.Attr("title"); !ok || v != "t" {
		t.Errorf("Attr title = %q %v", v, ok)
	}
}

func TestDocumentAccessor(t *testing.T) {
	doc := Parse(`<html><body><p>x</p></body></html>`)
	p := &Page{doc: doc, body: []byte("<html></html>")}
	if p.Document() != doc {
		t.Error("Document() mismatch")
	}
}

func TestFieldNamesAndFallbackForm(t *testing.T) {
	doc := Parse(`<section id="host"><form action="/x"><input name="a"><input name="b"></form></section>`)
	p := &Page{doc: doc}
	// Selector matches a wrapper element; FormBySelector should descend into
	// the inner form.
	f, err := p.FormBySelector("#host")
	if err != nil {
		t.Fatal(err)
	}
	names := f.FieldNames()
	if len(names) != 2 || names[0] != "a" || names[1] != "b" {
		t.Errorf("FieldNames = %v", names)
	}
}

func TestFormBySelectorNotAForm(t *testing.T) {
	doc := Parse(`<div id="d"><p>no form here</p></div>`)
	p := &Page{doc: doc}
	if _, err := p.FormBySelector("#d"); err == nil {
		t.Error("expected error when element is not and contains no form")
	}
}

func TestBogusCommentAndDoctypeVariants(t *testing.T) {
	doc := Parse(`<?xml version="1.0"?><![CDATA[data]]><p>ok</p>`)
	// The processing instruction and CDATA become comment nodes; <p> still parses.
	if firstElement(doc, "p") == nil {
		t.Error("<p> after bogus comments not parsed")
	}
	comments := 0
	var walk func(n *Node)
	walk = func(n *Node) {
		if n.Type == CommentNode {
			comments++
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(doc)
	if comments < 1 {
		t.Errorf("expected bogus comment nodes, got %d", comments)
	}
}

func TestPageQuerySelectorAllRelativeError(t *testing.T) {
	doc := Parse(`<div><span>x</span></div>`)
	p := &Page{doc: doc}
	if _, err := p.QuerySelectorAll("<<bad"); err == nil {
		t.Error("expected selector compile error")
	}
	if _, err := p.QuerySelector(":bogus"); err == nil {
		t.Error("expected selector compile error")
	}
}
