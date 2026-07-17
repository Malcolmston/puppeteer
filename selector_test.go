package puppeteer

import (
	"testing"
)

const selectorDoc = `
<html><body>
<div id="main" class="container">
  <h1 class="title">Heading</h1>
  <p class="lead first">Lead paragraph</p>
  <p class="body">Second <a href="/a" data-role="link">alpha</a></p>
  <ul class="list">
    <li class="item">one</li>
    <li class="item selected">two</li>
    <li class="item">three</li>
    <li class="item last">four</li>
  </ul>
  <section>
    <span lang="en">english</span>
    <span lang="en-US">american</span>
    <input type="text" name="q" value="hi">
    <input type="checkbox" name="c" checked>
  </section>
</div>
<footer><a href="https://ex.com/">home</a></footer>
</body></html>`

func mustQueryAll(t *testing.T, doc *Node, sel string) []*Node {
	t.Helper()
	s, err := compileSelector(sel)
	if err != nil {
		t.Fatalf("compile %q: %v", sel, err)
	}
	return s.queryAll(doc)
}

func TestSelectorCases(t *testing.T) {
	doc := Parse(selectorDoc)
	cases := []struct {
		sel  string
		want int
	}{
		{"div", 1},
		{"p", 2},
		{"li", 4},
		{"*", countElements(doc)},
		{"#main", 1},
		{".item", 4},
		{".item.selected", 1},
		{"p.body", 1},
		{"[data-role]", 1},
		{`[data-role="link"]`, 1},
		{`[href^="/"]`, 1},
		{`[href$=".com/"]`, 1},
		{`[href*="ex"]`, 1},
		{`[lang~="en"]`, 1},
		{`[lang|="en"]`, 2},
		{"div p", 2},
		{"ul > li", 4},
		{"div > p", 2},
		{"h1 + p", 1},
		{"h1 ~ p", 2},
		{"li.selected + li", 1},
		{"li:first-child", 1},
		{"li:last-child", 1},
		{"li:nth-child(2)", 1},
		{"li:nth-child(odd)", 2},
		{"li:nth-child(even)", 2},
		{"li:nth-child(2n)", 2},
		{"h1, footer a", 2},
		{"section input", 2},
		{`input[type="checkbox"][checked]`, 1},
	}
	for _, c := range cases {
		got := len(mustQueryAll(t, doc, c.sel))
		if got != c.want {
			t.Errorf("selector %q matched %d, want %d", c.sel, got, c.want)
		}
	}
}

func TestSelectorNthAdvanced(t *testing.T) {
	doc := Parse(`<ul><li>1</li><li>2</li><li>3</li><li>4</li><li>5</li></ul>`)
	// -n+3 -> first three
	got := mustQueryAll(t, doc, "li:nth-child(-n+3)")
	if len(got) != 3 {
		t.Errorf("-n+3 matched %d, want 3", len(got))
	}
	// 2n+1 -> 1,3,5
	got = mustQueryAll(t, doc, "li:nth-child(2n+1)")
	if len(got) != 3 {
		t.Errorf("2n+1 matched %d, want 3", len(got))
	}
}

func TestSelectorFirstMatchContent(t *testing.T) {
	doc := Parse(selectorDoc)
	s, _ := compileSelector("a[href]")
	n := s.queryFirst(doc)
	if n == nil {
		t.Fatal("no match")
	}
	if v, _ := n.attr("href"); v != "/a" {
		t.Errorf("first a href = %q, want /a", v)
	}
}

func TestSelectorErrors(t *testing.T) {
	bad := []string{"", "  ", "div >", "[", "[=x]", ":unknown", "div!", ":nth-child(", ":nth-child(z)"}
	for _, s := range bad {
		if _, err := compileSelector(s); err == nil {
			t.Errorf("expected error for %q", s)
		}
	}
}

func TestElementRelativeQuery(t *testing.T) {
	doc := Parse(selectorDoc)
	ul := wrapElement(firstElement(doc, "ul"))
	items, err := ul.QuerySelectorAll(".item")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 4 {
		t.Errorf("relative query matched %d, want 4", len(items))
	}
	// context node itself is not returned
	self, _ := ul.QuerySelector("ul")
	if self != nil {
		t.Error("context element should not match its own descendant query")
	}
}

func TestClosestAndMatches(t *testing.T) {
	doc := Parse(selectorDoc)
	a := wrapElement(firstElement(doc, "a"))
	c, err := a.Closest("div#main")
	if err != nil {
		t.Fatal(err)
	}
	if c == nil || c.ID() != "main" {
		t.Errorf("closest = %v", c)
	}
	if ok, _ := a.Matches("a[data-role]"); !ok {
		t.Error("Matches should be true")
	}
	if ok, _ := a.Matches("span"); ok {
		t.Error("Matches should be false")
	}
}

func TestTraversal(t *testing.T) {
	doc := Parse(selectorDoc)
	items := mustQueryAll(t, doc, "li")
	second := wrapElement(items[1])
	if got := second.TextContent(); got != "two" {
		t.Fatalf("second li text = %q", got)
	}
	if p := second.Prev(); p == nil || p.TextContent() != "one" {
		t.Errorf("prev = %v", p)
	}
	if n := second.Next(); n == nil || n.TextContent() != "three" {
		t.Errorf("next = %v", n)
	}
	if par := second.Parent(); par == nil || par.TagName() != "ul" {
		t.Errorf("parent = %v", par)
	}
	first := wrapElement(items[0])
	if first.Prev() != nil {
		t.Error("first li should have no prev element")
	}
}

func TestClassHelpers(t *testing.T) {
	doc := Parse(selectorDoc)
	sel := wrapElement(mustQueryAll(t, doc, "li.selected")[0])
	if !sel.HasClass("selected") || !sel.HasClass("item") {
		t.Error("HasClass failed")
	}
	if sel.HasClass("nope") {
		t.Error("HasClass false positive")
	}
	if len(sel.ClassList()) != 2 {
		t.Errorf("ClassList = %v", sel.ClassList())
	}
}

func countElements(n *Node) int {
	c := 0
	if n.Type == ElementNode {
		c = 1
	}
	for _, ch := range n.Children {
		c += countElements(ch)
	}
	return c
}
