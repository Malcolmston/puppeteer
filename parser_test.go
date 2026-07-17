package puppeteer

import (
	"strings"
	"testing"
)

func TestParseBasicTree(t *testing.T) {
	doc := Parse(`<!DOCTYPE html><html><head><title>Hi</title></head><body><p>Hello</p></body></html>`)
	if doc.Type != DocumentNode {
		t.Fatalf("root type = %v, want DocumentNode", doc.Type)
	}
	// doctype + html
	var doctype, htmlEl *Node
	for _, c := range doc.Children {
		switch c.Type {
		case DoctypeNode:
			doctype = c
		case ElementNode:
			htmlEl = c
		}
	}
	if doctype == nil || !strings.EqualFold(doctype.Data, "html") {
		t.Fatalf("doctype = %+v", doctype)
	}
	if htmlEl == nil || htmlEl.Data != "html" {
		t.Fatalf("html element missing: %+v", htmlEl)
	}
}

func TestParseAttributes(t *testing.T) {
	doc := Parse(`<a href="/x" data-n=5 class='a b' disabled>link</a>`)
	a := firstElement(doc, "a")
	if a == nil {
		t.Fatal("no <a>")
	}
	if v, _ := a.attr("href"); v != "/x" {
		t.Errorf("href = %q", v)
	}
	if v, _ := a.attr("data-n"); v != "5" {
		t.Errorf("unquoted attr = %q", v)
	}
	if v, _ := a.attr("class"); v != "a b" {
		t.Errorf("single-quoted attr = %q", v)
	}
	if v, ok := a.attr("disabled"); !ok || v != "" {
		t.Errorf("boolean attr = %q ok=%v", v, ok)
	}
}

func TestParseVoidElements(t *testing.T) {
	doc := Parse(`<div><br><img src="a.png"><input name="q"></div>`)
	div := firstElement(doc, "div")
	if div == nil {
		t.Fatal("no div")
	}
	count := 0
	for _, c := range div.Children {
		if c.Type == ElementNode {
			count++
			if len(c.Children) != 0 {
				t.Errorf("void %s has children", c.Data)
			}
		}
	}
	if count != 3 {
		t.Errorf("void children = %d, want 3", count)
	}
}

func TestParseRawTextScript(t *testing.T) {
	doc := Parse(`<script>if (a < b && c > d) { x(); }</script><p>after</p>`)
	s := firstElement(doc, "script")
	if s == nil || len(s.Children) != 1 {
		t.Fatalf("script = %+v", s)
	}
	if got := s.Children[0].Data; got != "if (a < b && c > d) { x(); }" {
		t.Errorf("script text = %q", got)
	}
	if firstElement(doc, "p") == nil {
		t.Error("<p> after script not parsed")
	}
}

func TestParseComment(t *testing.T) {
	doc := Parse(`<div><!-- hi there --></div>`)
	div := firstElement(doc, "div")
	if div == nil || len(div.Children) != 1 || div.Children[0].Type != CommentNode {
		t.Fatalf("comment not parsed: %+v", div)
	}
	if div.Children[0].Data != " hi there " {
		t.Errorf("comment data = %q", div.Children[0].Data)
	}
}

func TestParseImpliedListClose(t *testing.T) {
	doc := Parse(`<ul><li>a<li>b<li>c</ul>`)
	ul := firstElement(doc, "ul")
	if ul == nil {
		t.Fatal("no ul")
	}
	lis := 0
	for _, c := range ul.Children {
		if c.Type == ElementNode && c.Data == "li" {
			lis++
		}
	}
	if lis != 3 {
		t.Errorf("li count = %d, want 3", lis)
	}
}

func TestParseImpliedParagraphClose(t *testing.T) {
	doc := Parse(`<p>one<div>two</div>`)
	body := doc
	var p, div *Node
	var walk func(n *Node)
	walk = func(n *Node) {
		for _, c := range n.Children {
			if c.Data == "p" && p == nil {
				p = c
			}
			if c.Data == "div" && div == nil {
				div = c
			}
			walk(c)
		}
	}
	walk(body)
	if p == nil || div == nil {
		t.Fatalf("p=%v div=%v", p, div)
	}
	if div.Parent == p {
		t.Error("div should not be nested inside p")
	}
}

func TestEntityDecoding(t *testing.T) {
	doc := Parse(`<p>a &amp; b &lt; c &#65; &#x42; &nbsp;&copy;</p>`)
	p := firstElement(doc, "p")
	got := textOf(p)
	want := "a & b < c A B  ©"
	if got != want {
		t.Errorf("decoded = %q, want %q", got, want)
	}
}

func TestRenderRoundTrip(t *testing.T) {
	el := firstElement(Parse(`<div class="x"><span>hi &amp; bye</span></div>`), "div")
	var sb strings.Builder
	el.render(&sb)
	got := sb.String()
	want := `<div class="x"><span>hi &amp; bye</span></div>`
	if got != want {
		t.Errorf("render = %q, want %q", got, want)
	}
}

func TestInnerOuterHTML(t *testing.T) {
	el := wrapElement(firstElement(Parse(`<ul><li>a</li><li>b</li></ul>`), "ul"))
	if got := el.InnerHTML(); got != `<li>a</li><li>b</li>` {
		t.Errorf("InnerHTML = %q", got)
	}
	if got := el.OuterHTML(); got != `<ul><li>a</li><li>b</li></ul>` {
		t.Errorf("OuterHTML = %q", got)
	}
}

// helpers

func firstElement(n *Node, tag string) *Node {
	if n.Type == ElementNode && n.Data == tag {
		return n
	}
	for _, c := range n.Children {
		if r := firstElement(c, tag); r != nil {
			return r
		}
	}
	return nil
}

func textOf(n *Node) string {
	var sb strings.Builder
	n.text(&sb)
	return sb.String()
}
