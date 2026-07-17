package puppeteer

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc123", Path: "/"})
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, `<!DOCTYPE html><html><head><title>Home &amp; Away</title></head>
<body>
<h1>Welcome</h1>
<nav><a href="/about">About</a> <a href="page2.html">Next</a> <a href="/about">About dup</a></nav>
<p>User-Agent: `+r.Header.Get("User-Agent")+`</p>
<p>X-Test: `+r.Header.Get("X-Test")+`</p>
</body></html>`)
	})
	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		// Echo whether the session cookie came back.
		c, _ := r.Cookie("session")
		val := ""
		if c != nil {
			val = c.Value
		}
		_, _ = io.WriteString(w, `<html><body><h1 id="t">About</h1><span class="cookie">`+val+`</span></body></html>`)
	})
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/about", http.StatusFound)
	})
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		method := r.Method
		var q string
		if method == http.MethodGet {
			q = r.URL.Query().Get("q")
		} else {
			_ = r.ParseForm()
			q = r.PostFormValue("q")
		}
		_, _ = io.WriteString(w, `<html><body><p class="result">`+method+`:`+q+`</p></body></html>`)
	})
	mux.HandleFunc("/form", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `<html><body>
<form id="f" action="/search" method="post">
  <input type="text" name="q" value="default">
  <input type="hidden" name="token" value="xyz">
  <input type="checkbox" name="remember" value="1" checked>
  <input type="checkbox" name="news" value="1">
  <textarea name="comment">hello</textarea>
  <select name="cat"><option value="a">A</option><option value="b" selected>B</option></select>
  <input type="submit" name="go" value="Go">
</form>
<form id="g" action="/search" method="get">
  <input type="text" name="q" value="">
</form>
</body></html>`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestGotoAndTitle(t *testing.T) {
	srv := newTestServer(t)
	b, err := Launch(&LaunchOptions{UserAgent: "test-agent", Headers: map[string]string{"X-Test": "yes"}})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = b.Close() }()
	p := b.NewPage()
	resp, err := p.Goto(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	if !resp.OK() {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if p.Title() != "Home & Away" {
		t.Errorf("title = %q", p.Title())
	}
	if p.URL() != srv.URL+"/" {
		t.Errorf("url = %q", p.URL())
	}
	if !strings.Contains(p.Content(), "User-Agent: test-agent") {
		t.Error("custom user agent not sent")
	}
	if !strings.Contains(p.Content(), "X-Test: yes") {
		t.Error("custom header not sent")
	}
	if !strings.Contains(p.HTML(), "<h1>") {
		t.Error("HTML serialization missing content")
	}
}

func TestLinksResolution(t *testing.T) {
	srv := newTestServer(t)
	b, _ := Launch(nil)
	p := b.NewPage()
	if _, err := p.Goto(srv.URL + "/"); err != nil {
		t.Fatal(err)
	}
	links := p.Links()
	// dedup: /about appears twice -> once; plus page2.html resolved absolute
	want := map[string]bool{
		srv.URL + "/about":      true,
		srv.URL + "/page2.html": true,
	}
	if len(links) != 2 {
		t.Fatalf("links = %v", links)
	}
	for _, l := range links {
		if !want[l] {
			t.Errorf("unexpected link %q", l)
		}
	}
}

func TestCookiesPersist(t *testing.T) {
	srv := newTestServer(t)
	b, _ := Launch(nil)
	p := b.NewPage()
	if _, err := p.Goto(srv.URL + "/"); err != nil {
		t.Fatal(err)
	}
	cookies, err := b.Cookies(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	if len(cookies) != 1 || cookies[0].Value != "abc123" {
		t.Fatalf("cookies = %v", cookies)
	}
	// Second navigation should send the cookie back.
	if _, err := p.Goto(srv.URL + "/about"); err != nil {
		t.Fatal(err)
	}
	el, _ := p.QuerySelector("span.cookie")
	if el == nil || el.TextContent() != "abc123" {
		t.Errorf("cookie not echoed: %v", el)
	}
}

func TestRedirectFollow(t *testing.T) {
	srv := newTestServer(t)
	b, _ := Launch(nil)
	p := b.NewPage()
	resp, err := p.Goto(srv.URL + "/redirect")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if !strings.HasSuffix(p.URL(), "/about") {
		t.Errorf("final url = %q", p.URL())
	}
}

func TestRedirectDisabled(t *testing.T) {
	srv := newTestServer(t)
	no := false
	b, _ := Launch(&LaunchOptions{FollowRedirects: &no})
	p := b.NewPage()
	resp, err := p.Goto(srv.URL + "/redirect")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusFound {
		t.Errorf("status = %d, want 302", resp.StatusCode)
	}
}

func TestRelativeGotoRequiresBase(t *testing.T) {
	b, _ := Launch(nil)
	p := b.NewPage()
	if _, err := p.Goto("relative/path"); err == nil {
		t.Error("expected error for relative goto without base")
	}
}

func TestSetCookies(t *testing.T) {
	srv := newTestServer(t)
	b, _ := Launch(nil)
	if err := b.SetCookies(srv.URL+"/", []*http.Cookie{{Name: "session", Value: "manual"}}); err != nil {
		t.Fatal(err)
	}
	p := b.NewPage()
	if _, err := p.Goto(srv.URL + "/about"); err != nil {
		t.Fatal(err)
	}
	el, _ := p.QuerySelector(".cookie")
	if el == nil || el.TextContent() != "manual" {
		t.Errorf("manual cookie not sent: %v", el)
	}
}

func TestQueryBeforeGoto(t *testing.T) {
	b, _ := Launch(nil)
	p := b.NewPage()
	if _, err := p.QuerySelector("div"); err == nil {
		t.Error("expected error querying before navigation")
	}
	if p.URL() != "" || p.Content() != "" || p.HTML() != "" || p.Title() != "" {
		t.Error("empty page accessors should be zero values")
	}
}

func TestFormGetBuild(t *testing.T) {
	srv := newTestServer(t)
	b, _ := Launch(nil)
	p := b.NewPage()
	if _, err := p.Goto(srv.URL + "/form"); err != nil {
		t.Fatal(err)
	}
	f, err := p.FillForm("#g", map[string]string{"q": "golang"})
	if err != nil {
		t.Fatal(err)
	}
	if f.Method != http.MethodGet {
		t.Fatalf("method = %q", f.Method)
	}
	req, err := f.BuildRequest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != http.MethodGet || req.URL.Query().Get("q") != "golang" {
		t.Errorf("GET request wrong: %s %s", req.Method, req.URL)
	}
}

func TestFormPostSubmit(t *testing.T) {
	srv := newTestServer(t)
	b, _ := Launch(nil)
	p := b.NewPage()
	if _, err := p.Goto(srv.URL + "/form"); err != nil {
		t.Fatal(err)
	}
	f, err := p.FormBySelector("#f")
	if err != nil {
		t.Fatal(err)
	}
	// Defaults captured from markup.
	if v, _ := f.Get("q"); v != "default" {
		t.Errorf("default q = %q", v)
	}
	if v, _ := f.Get("comment"); v != "hello" {
		t.Errorf("textarea value = %q", v)
	}
	if v, _ := f.Get("cat"); v != "b" {
		t.Errorf("selected option = %q", v)
	}
	vals := f.Values()
	if vals.Get("remember") != "1" {
		t.Error("checked checkbox should be included")
	}
	if _, ok := vals["news"]; ok {
		t.Error("unchecked checkbox should be excluded")
	}
	if _, ok := vals["go"]; ok {
		t.Error("submit button should be excluded")
	}
	// Override and submit (POST).
	f.Set("q", "hello world")
	resp, err := f.Submit()
	if err != nil {
		t.Fatal(err)
	}
	if !resp.OK() {
		t.Fatalf("submit status %d", resp.StatusCode)
	}
	res, _ := p.QuerySelector(".result")
	if res == nil || res.TextContent() != "POST:hello world" {
		t.Errorf("post result = %v", res)
	}
}

func TestFormBuildPostBody(t *testing.T) {
	srv := newTestServer(t)
	b, _ := Launch(nil)
	p := b.NewPage()
	if _, err := p.Goto(srv.URL + "/form"); err != nil {
		t.Fatal(err)
	}
	f, _ := p.FormBySelector("#f")
	req, err := f.BuildRequest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != http.MethodPost {
		t.Fatalf("method = %q", req.Method)
	}
	if ct := req.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
		t.Errorf("content-type = %q", ct)
	}
	body, _ := io.ReadAll(req.Body)
	if !strings.Contains(string(body), "token=xyz") {
		t.Errorf("body missing hidden field: %q", body)
	}
}

func TestFormNotFound(t *testing.T) {
	srv := newTestServer(t)
	b, _ := Launch(nil)
	p := b.NewPage()
	if _, err := p.Goto(srv.URL + "/"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.FormBySelector("#nope"); err == nil {
		t.Error("expected error for missing form")
	}
	forms, err := p.Forms()
	if err != nil {
		t.Fatal(err)
	}
	if len(forms) != 0 {
		t.Errorf("forms = %d, want 0", len(forms))
	}
}

func TestWaitForSelectorImmediate(t *testing.T) {
	srv := newTestServer(t)
	b, _ := Launch(nil)
	p := b.NewPage()
	if _, err := p.Goto(srv.URL + "/about"); err != nil {
		t.Fatal(err)
	}
	el, err := p.WaitForSelector("#t", &WaitForSelectorOptions{Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if el.TextContent() != "About" {
		t.Errorf("waited element text = %q", el.TextContent())
	}
}

func TestWaitForSelectorTimeout(t *testing.T) {
	srv := newTestServer(t)
	b, _ := Launch(nil)
	p := b.NewPage()
	if _, err := p.Goto(srv.URL + "/about"); err != nil {
		t.Fatal(err)
	}
	_, err := p.WaitForSelector("#missing", &WaitForSelectorOptions{Timeout: 200 * time.Millisecond, Interval: 50 * time.Millisecond})
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestPageUserAgentOverride(t *testing.T) {
	srv := newTestServer(t)
	b, _ := Launch(nil)
	p := b.NewPage()
	p.SetUserAgent("page-ua")
	p.SetExtraHTTPHeaders(map[string]string{"X-Test": "page"})
	if _, err := p.Goto(srv.URL + "/"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(p.Content(), "User-Agent: page-ua") {
		t.Error("page UA override not applied")
	}
	if !strings.Contains(p.Content(), "X-Test: page") {
		t.Error("page header not applied")
	}
	b.SetUserAgent("browser-ua")
	if b.UserAgent() != "browser-ua" {
		t.Error("browser SetUserAgent failed")
	}
	b.SetExtraHTTPHeaders(map[string]string{"X-Test": "browser"})
}
