package puppeteer

import (
	"reflect"
	"testing"
)

// firstElement returns the first element matching selector in html, failing the
// test if none is found.
func ppFirstEl(t *testing.T, html, selector string) *Element {
	t.Helper()
	doc := Parse(html)
	sel, err := compileSelector(selector)
	if err != nil {
		t.Fatalf("compileSelector(%q): %v", selector, err)
	}
	n := sel.queryFirst(doc)
	if n == nil {
		t.Fatalf("no element matched %q", selector)
	}
	return wrapElement(n)
}

func TestElementHasAttributeAndNames(t *testing.T) {
	el := ppFirstEl(t, `<a href="/x" class="c" data-id="7" hidden>hi</a>`, "a")
	if !el.HasAttribute("HREF") {
		t.Error("HasAttribute should be case-insensitive for href")
	}
	if !el.HasAttribute("hidden") {
		t.Error("valueless attribute hidden should be present")
	}
	if el.HasAttribute("nope") {
		t.Error("absent attribute reported present")
	}
	got := el.AttributeNames()
	want := []string{"href", "class", "data-id", "hidden"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("AttributeNames = %v, want %v", got, want)
	}
}

func TestElementDataset(t *testing.T) {
	el := ppFirstEl(t, `<div id="d" data-user-id="42" data-role="admin" data-="skip" class="x">x</div>`, "#d")
	got := el.Dataset()
	want := map[string]string{"userId": "42", "role": "admin"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Dataset = %v, want %v", got, want)
	}
}

func TestPPDatasetKey(t *testing.T) {
	cases := []struct{ in, want string }{
		{"user-id", "userId"},
		{"role", "role"},
		{"a-b-c", "aBC"},
		{"", ""},
		{"-leading", "Leading"},
		{"trailing-", "trailing"},
	}
	for _, c := range cases {
		if got := ppDatasetKey(c.in); got != c.want {
			t.Errorf("ppDatasetKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

const ppSiblingHTML = `<ul><li id="a">A</li><li id="b">B</li>text<li id="c">C</li><li id="d">D</li></ul>`

func ppIDs(els []*Element) []string {
	out := make([]string, 0, len(els))
	for _, e := range els {
		out = append(out, e.ID())
	}
	return out
}

func TestElementSiblings(t *testing.T) {
	el := ppFirstEl(t, ppSiblingHTML, "#b")
	if got, want := ppIDs(el.Siblings()), []string{"a", "c", "d"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Siblings = %v, want %v", got, want)
	}
	if got, want := ppIDs(el.NextAll()), []string{"c", "d"}; !reflect.DeepEqual(got, want) {
		t.Errorf("NextAll = %v, want %v", got, want)
	}
	if got, want := ppIDs(el.PrevAll()), []string{"a"}; !reflect.DeepEqual(got, want) {
		t.Errorf("PrevAll = %v, want %v", got, want)
	}
	c := ppFirstEl(t, ppSiblingHTML, "#c")
	if got, want := ppIDs(c.PrevAll()), []string{"a", "b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("PrevAll(c) = %v, want %v (document order)", got, want)
	}
}

func TestElementAncestors(t *testing.T) {
	el := ppFirstEl(t, `<html><body><section><p><span id="s">x</span></p></section></body></html>`, "#s")
	got := make([]string, 0)
	for _, a := range el.Ancestors() {
		got = append(got, a.TagName())
	}
	want := []string{"p", "section", "body", "html"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Ancestors = %v, want %v", got, want)
	}
}

func TestElementIsEmpty(t *testing.T) {
	cases := []struct {
		html, sel string
		want      bool
	}{
		{`<p id="x"></p>`, "#x", true},
		{`<p id="x">   </p>`, "#x", true},
		{`<p id="x">hi</p>`, "#x", false},
		{`<p id="x"><span>y</span></p>`, "#x", false},
		{`<p id="x"><!-- c --></p>`, "#x", false},
		{`<br id="x">`, "#x", true},
	}
	for _, c := range cases {
		el := ppFirstEl(t, c.html, c.sel)
		if got := el.IsEmpty(); got != c.want {
			t.Errorf("IsEmpty(%q) = %v, want %v", c.html, got, c.want)
		}
	}
}

func BenchmarkElementDataset(b *testing.B) {
	doc := Parse(`<div data-a="1" data-user-id="2" data-role-name="x" class="c" id="d">t</div>`)
	sel, _ := compileSelector("#d")
	el := wrapElement(sel.queryFirst(doc))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = el.Dataset()
	}
}
