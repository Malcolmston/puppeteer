package puppeteer

import (
	"strconv"
	"strings"
	"testing"
)

// This file encodes known-answer vectors taken directly from the upstream
// Node.js Puppeteer test suite (puppeteer/puppeteer), primarily
// test/src/queryselector.test.ts and test/src/page.test.ts, plus the
// test/assets/title.html fixture. Each case asserts the exact value the
// original library asserts, mapped onto this port's standard-library DOM,
// selector engine and page API.
//
// Vectors that require a live browser (script execution, layout/geometry,
// XPath handled by the browser) are recorded as documented t.Skip ceilings.

// pageFrom builds a Page from a raw HTML string, mirroring page.setContent.
func pageFrom(html string) *Page {
	return &Page{doc: Parse(html), body: []byte(html)}
}

// TestParityPageEval mirrors querySelector.test.ts "Page.$eval".
func TestParityPageEval(t *testing.T) {
	// "should work": e => e.id on <section id="testAttribute">.
	p := pageFrom(`<section id="testAttribute">43543</section>`)
	el, err := p.QuerySelector("section")
	if err != nil || el == nil {
		t.Fatalf("QuerySelector(section) = %v, %v", el, err)
	}
	if got := el.ID(); got != "testAttribute" {
		t.Errorf("id = %q, want %q", got, "testAttribute")
	}

	// "should accept arguments": textContent + " world!".
	p = pageFrom(`<section>hello</section>`)
	el, _ = p.QuerySelector("section")
	if got := el.TextContent() + " world!"; got != "hello world!" {
		t.Errorf("text = %q, want %q", got, "hello world!")
	}
}

// TestParityPageEvalAll mirrors querySelector.test.ts "Page.$$eval".
func TestParityPageEvalAll(t *testing.T) {
	// "should work": three divs => length 3.
	p := pageFrom(`<div>hello</div><div>beautiful</div><div>world!</div>`)
	divs, err := p.QuerySelectorAll("div")
	if err != nil {
		t.Fatal(err)
	}
	if len(divs) != 3 {
		t.Errorf("div count = %d, want 3", len(divs))
	}

	// "should accept ElementHandles as arguments":
	// sum of <section> values plus the <div> value == 8.
	p = pageFrom(`<section>2</section><section>2</section><section>1</section><div>3</div>`)
	sections, _ := p.QuerySelectorAll("section")
	sum := 0
	for _, s := range sections {
		n, _ := strconv.Atoi(strings.TrimSpace(s.TextContent()))
		sum += n
	}
	div, _ := p.QuerySelector("div")
	dn, _ := strconv.Atoi(strings.TrimSpace(div.TextContent()))
	if sum+dn != 8 {
		t.Errorf("sum = %d, want 8", sum+dn)
	}
}

// TestParityPageQuery mirrors querySelector.test.ts "Page.$" and "Page.$$".
func TestParityPageQuery(t *testing.T) {
	// "$ should query existing element".
	p := pageFrom(`<section>test</section>`)
	if el, _ := p.QuerySelector("section"); el == nil {
		t.Error("QuerySelector(section) = nil, want element")
	}

	// "$ should return null for non-existing element".
	el, err := p.QuerySelector("non-existing-element")
	if el != nil || err != nil {
		t.Errorf("QuerySelector(non-existing) = %v, %v; want nil, nil", el, err)
	}

	// "$$ should query existing elements": ["A", "B"] in document order.
	p = pageFrom(`<div>A</div><br /><div>B</div>`)
	divs, _ := p.QuerySelectorAll("div")
	if got := texts(divs); !equal(got, []string{"A", "B"}) {
		t.Errorf("$$ texts = %v, want [A B]", got)
	}

	// "$$ should return empty array if nothing is found".
	p = pageFrom(``)
	divs, _ = p.QuerySelectorAll("div")
	if len(divs) != 0 {
		t.Errorf("$$ empty = %d, want 0", len(divs))
	}
}

// TestParityElementHandleQuery mirrors querySelector.test.ts "ElementHandle.$"
// and "ElementHandle.$$", including scoped subtree resolution.
func TestParityElementHandleQuery(t *testing.T) {
	// "ElementHandle.$ should query existing element": html > .second > .inner.
	p := pageFrom(`<html><body><div class="second"><div class="inner">A</div></div></body></html>`)
	htmlEl, _ := p.QuerySelector("html")
	second, _ := htmlEl.QuerySelector(".second")
	if second == nil {
		t.Fatal(".second = nil")
	}
	inner, _ := second.QuerySelector(".inner")
	if inner == nil || inner.TextContent() != "A" {
		t.Errorf("inner text = %v, want A", inner)
	}

	// "ElementHandle.$ should return null for non-existing element".
	p = pageFrom(`<html><body><div class="second"><div class="inner">B</div></div></body></html>`)
	htmlEl, _ = p.QuerySelector("html")
	if third, _ := htmlEl.QuerySelector(".third"); third != nil {
		t.Error(".third != nil, want nil")
	}

	// "ElementHandle.$$ should query existing elements": ["A","B"].
	p = pageFrom(`<html><body><div>A</div><br /><div>B</div></body></html>`)
	htmlEl, _ = p.QuerySelector("html")
	els, _ := htmlEl.QuerySelectorAll("div")
	if got := texts(els); !equal(got, []string{"A", "B"}) {
		t.Errorf("scoped $$ = %v, want [A B]", got)
	}

	// "ElementHandle.$$ should return empty array for non-existing elements".
	p = pageFrom(`<html><body><span>A</span><br /><span>B</span></body></html>`)
	htmlEl, _ = p.QuerySelector("html")
	els, _ = htmlEl.QuerySelectorAll("div")
	if len(els) != 0 {
		t.Errorf("scoped $$ empty = %d, want 0", len(els))
	}
}

// TestParityElementHandleEval mirrors querySelector.test.ts
// "ElementHandle.$eval" / "ElementHandle.$$eval", which resolve selectors
// against the element's subtree only (not the whole document).
func TestParityElementHandleEval(t *testing.T) {
	// "$eval should work": .tweet .like => "100".
	p := pageFrom(`<html><body><div class="tweet"><div class="like">100</div><div class="retweets">10</div></div></body></html>`)
	tweet, _ := p.QuerySelector(".tweet")
	like, _ := tweet.QuerySelector(".like")
	if like == nil || like.TextContent() != "100" {
		t.Errorf("like = %v, want 100", like)
	}

	// "$eval should retrieve content from subtree": #myId .a is the child div,
	// not the earlier not-a-child-div sibling.
	p = pageFrom(`<div class="a">not-a-child-div</div><div id="myId"><div class="a">a-child-div</div></div>`)
	myID, _ := p.QuerySelector("#myId")
	a, _ := myID.QuerySelector(".a")
	if a == nil || a.TextContent() != "a-child-div" {
		t.Errorf("subtree .a = %v, want a-child-div", a)
	}

	// "$$eval should work": .tweet .like => ["100","10"].
	p = pageFrom(`<html><body><div class="tweet"><div class="like">100</div><div class="like">10</div></div></body></html>`)
	tweet, _ = p.QuerySelector(".tweet")
	likes, _ := tweet.QuerySelectorAll(".like")
	if got := texts(likes); !equal(got, []string{"100", "10"}) {
		t.Errorf("$$eval = %v, want [100 10]", got)
	}

	// "$$eval should retrieve content from subtree".
	p = pageFrom(`<div class="a">not-a-child-div</div><div id="myId"><div class="a">a1-child-div</div><div class="a">a2-child-div</div></div>`)
	myID, _ = p.QuerySelector("#myId")
	as, _ := myID.QuerySelectorAll(".a")
	if got := texts(as); !equal(got, []string{"a1-child-div", "a2-child-div"}) {
		t.Errorf("subtree $$ = %v, want [a1-child-div a2-child-div]", got)
	}

	// "$$eval should not throw in case of missing selector": length 0.
	p = pageFrom(`<div class="a">not-a-child-div</div><div id="myId"></div>`)
	myID, _ = p.QuerySelector("#myId")
	as, err := myID.QuerySelectorAll(".a")
	if err != nil || len(as) != 0 {
		t.Errorf("missing subtree $$ = %d, %v; want 0, nil", len(as), err)
	}
}

// TestParityPageTitle mirrors page.test.ts "Page.title" using the upstream
// test/assets/title.html fixture (<!DOCTYPE html><title>Woof-Woof</title>).
func TestParityPageTitle(t *testing.T) {
	p := pageFrom("<!DOCTYPE html>\n<title>Woof-Woof</title>\n")
	if got := p.Title(); got != "Woof-Woof" {
		t.Errorf("title = %q, want %q", got, "Woof-Woof")
	}
}

// TestParitySetContent mirrors page.test.ts "Page.setContent", asserting the
// browser-normalized serialization returned by page.content().
func TestParitySetContent(t *testing.T) {
	const expected = "<html><head></head><body><div>hello</div></body></html>"
	cases := []struct{ in, want string }{
		// "should work".
		{`<div>hello</div>`, expected},
		// "should work with doctype".
		{`<!DOCTYPE html><div>hello</div>`, "<!DOCTYPE html>" + expected},
	}
	for _, c := range cases {
		if got := pageFrom(c.in).NormalizedHTML(); got != c.want {
			t.Errorf("NormalizedHTML(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestParitySetContentText mirrors page.test.ts setContent text cases
// ("with accents", "with emojis", "with newline", "with tricky content").
func TestParitySetContentText(t *testing.T) {
	cases := []struct{ in, want string }{
		{`<div>aberración</div>`, "aberración"},
		{`<div>🐥</div>`, "🐥"},
		{"<div>\n</div>", "\n"},
		// "with tricky content": trailing \x7F lives outside the div.
		{"<div>hello world</div>\x7f", "hello world"},
	}
	for _, c := range cases {
		p := pageFrom(c.in)
		el, _ := p.QuerySelector("div")
		if el == nil || el.TextContent() != c.want {
			t.Errorf("textContent(%q) = %v, want %q", c.in, el, c.want)
		}
	}
}

// TestParitySelectorEngine asserts standard CSS Selectors Level 4 semantics
// that Puppeteer's page.$/$$ inherit by delegating to the browser's native
// querySelectorAll. These exercise the pseudo-classes and case-insensitive
// attribute flag the browser engine supports.
func TestParitySelectorEngine(t *testing.T) {
	doc := `<ul><li class="a">1</li><li>2</li><li class="a">3</li></ul>` +
		`<p>x</p><span></span><em>e</em>`
	p := pageFrom(doc)
	cases := []struct {
		sel  string
		want int
	}{
		{"li:not(.a)", 1},
		{":not(li)", 4}, // ul, p, span, em
		{"li:first-of-type", 1},
		{"li:last-of-type", 1},
		{"li:nth-of-type(2)", 1},
		{"li:nth-last-child(1)", 1},
		{"li:only-of-type", 0},
		{"em:only-of-type", 1},
		{"span:empty", 1},
		{"p:empty", 0},
		{"li:only-child", 0},
		{"em:only-child", 0},
		{"[class=A i]", 2}, // case-insensitive attribute flag
		{"[class=A]", 0},   // case-sensitive default
		{"li:not(:first-child)", 2},
	}
	for _, c := range cases {
		got, err := p.QuerySelectorAll(c.sel)
		if err != nil {
			t.Errorf("QuerySelectorAll(%q) error: %v", c.sel, err)
			continue
		}
		if len(got) != c.want {
			t.Errorf("QuerySelectorAll(%q) = %d, want %d", c.sel, len(got), c.want)
		}
	}
}

// TestParityXPathCeiling records that XPath selectors ("xpath/...") from
// querySelector.test.ts are a browser-backed ceiling: this port ships a CSS
// selector engine only and has no XPath evaluator.
func TestParityXPathCeiling(t *testing.T) {
	t.Skip("upstream xpath/ selectors require the browser's XPath engine; " +
		"this standard-library port implements CSS selectors only")
}

func texts(els []*Element) []string {
	out := make([]string, len(els))
	for i, e := range els {
		out[i] = e.TextContent()
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
