package puppeteer

import (
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

const ppExtractHTML = `<!DOCTYPE html><html><head>
<title>Doc</title>
<meta name="description" content="a page">
<meta property="og:title" content="OG Doc">
<meta charset="utf-8">
<link rel="stylesheet" href="/a.css">
<link rel="preload stylesheet" href="theme.css">
<link rel="icon" href="/favicon.ico">
<script src="/app.js"></script>
<script>var inline = 1;</script>
</head><body>
<img src="/logo.png"><img src="pic.png"><img alt="no src">
<p class="lead">Hello world</p>
<a href="/next" data-track="cta">Next</a>
</body></html>`

func ppServe(t *testing.T, body string) (*Page, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "sid", Value: "z9", Path: "/"})
		_, _ = io.WriteString(w, body)
	}))
	browser, err := Launch(&LaunchOptions{Transport: srv.Client().Transport})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	page := browser.NewPage()
	if _, err := page.Goto(srv.URL); err != nil {
		t.Fatalf("Goto: %v", err)
	}
	return page, srv.Close
}

func TestPageMetas(t *testing.T) {
	page, done := ppServe(t, ppExtractHTML)
	defer done()

	metas := page.Metas()
	if metas["description"] != "a page" {
		t.Errorf("description = %q", metas["description"])
	}
	if metas["og:title"] != "OG Doc" {
		t.Errorf("og:title = %q", metas["og:title"])
	}
	if _, ok := metas["charset"]; ok {
		t.Error("charset meta has no name/content key and should be skipped")
	}

	if v, ok := page.MetaContent("OG:TITLE"); !ok || v != "OG Doc" {
		t.Errorf("MetaContent(og:title) = %q,%v", v, ok)
	}
	if v, ok := page.MetaContent("description"); !ok || v != "a page" {
		t.Errorf("MetaContent(description) = %q,%v", v, ok)
	}
	if _, ok := page.MetaContent("missing"); ok {
		t.Error("MetaContent(missing) should be false")
	}
}

func TestPageResources(t *testing.T) {
	page, done := ppServe(t, ppExtractHTML)
	defer done()
	base := page.URL()

	imgs := page.Images()
	wantImgs := []string{base + "/logo.png", base + "/pic.png"}
	if !reflect.DeepEqual(imgs, wantImgs) {
		t.Errorf("Images = %v, want %v", imgs, wantImgs)
	}

	scripts := page.Scripts()
	if !reflect.DeepEqual(scripts, []string{base + "/app.js"}) {
		t.Errorf("Scripts = %v", scripts)
	}

	css := page.Stylesheets()
	wantCSS := []string{base + "/a.css", base + "/theme.css"}
	if !reflect.DeepEqual(css, wantCSS) {
		t.Errorf("Stylesheets = %v, want %v (rel~=stylesheet must match 'preload stylesheet')", css, wantCSS)
	}
}

func TestPageCountTextAttr(t *testing.T) {
	page, done := ppServe(t, ppExtractHTML)
	defer done()

	if n, err := page.Count("img"); err != nil || n != 3 {
		t.Errorf("Count(img) = %d,%v want 3", n, err)
	}
	if n, err := page.Count("script[src]"); err != nil || n != 1 {
		t.Errorf("Count(script[src]) = %d,%v want 1", n, err)
	}

	txt, ok, err := page.TextContent("p.lead")
	if err != nil || !ok || txt != "Hello world" {
		t.Errorf("TextContent(p.lead) = %q,%v,%v", txt, ok, err)
	}
	if _, ok, _ := page.TextContent("p.none"); ok {
		t.Error("TextContent(missing) should report ok=false")
	}

	v, ok, err := page.GetAttribute("a", "data-track")
	if err != nil || !ok || v != "cta" {
		t.Errorf("GetAttribute(a,data-track) = %q,%v,%v", v, ok, err)
	}
	if _, ok, _ := page.GetAttribute("a", "nope"); ok {
		t.Error("GetAttribute of absent attr should be ok=false")
	}
}

func TestPageCookies(t *testing.T) {
	page, done := ppServe(t, ppExtractHTML)
	defer done()
	cookies, err := page.Cookies()
	if err != nil {
		t.Fatalf("Cookies: %v", err)
	}
	found := false
	for _, c := range cookies {
		if c.Name == "sid" && c.Value == "z9" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected sid=z9 cookie, got %v", cookies)
	}

	// A fresh page with no navigation reports an error.
	b, _ := Launch(nil)
	if _, err := b.NewPage().Cookies(); err == nil {
		t.Error("Cookies before navigation should error")
	}
}
