package puppeteer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// DefaultUserAgent is sent when LaunchOptions does not override it.
const DefaultUserAgent = "puppeteer-go/0.1.0 (+https://github.com/malcolmston/puppeteer)"

// LaunchOptions configures a Browser. The zero value is valid and yields a
// browser with sensible defaults.
type LaunchOptions struct {
	// UserAgent overrides the default User-Agent header.
	UserAgent string
	// Timeout bounds each navigation. Zero means 30s; use a negative value for
	// no timeout.
	Timeout time.Duration
	// Headers are extra request headers applied to every navigation.
	Headers map[string]string
	// FollowRedirects controls whether 3xx responses are followed. Defaults to
	// true.
	FollowRedirects *bool
	// Jar overrides the cookie jar. When nil a fresh in-memory jar is created.
	Jar http.CookieJar
	// Transport overrides the HTTP transport. This is the seam used by tests to
	// point the browser at an httptest.Server.
	Transport http.RoundTripper
}

// Browser is a lightweight HTTP client with a cookie jar, shared headers and a
// user agent. It is the entry point of the package and models Puppeteer's
// Browser, minus any real rendering engine.
type Browser struct {
	client    *http.Client
	jar       http.CookieJar
	userAgent string
	headers   http.Header
	timeout   time.Duration
}

// Launch creates a Browser. Passing nil uses all defaults.
func Launch(opts *LaunchOptions) (*Browser, error) {
	if opts == nil {
		opts = &LaunchOptions{}
	}
	jar := opts.Jar
	if jar == nil {
		j, err := cookiejar.New(nil)
		if err != nil {
			return nil, fmt.Errorf("puppeteer: creating cookie jar: %w", err)
		}
		jar = j
	}

	timeout := opts.Timeout
	switch {
	case timeout == 0:
		timeout = 30 * time.Second
	case timeout < 0:
		timeout = 0
	}

	ua := opts.UserAgent
	if ua == "" {
		ua = DefaultUserAgent
	}

	headers := http.Header{}
	for k, v := range opts.Headers {
		headers.Set(k, v)
	}

	client := &http.Client{
		Jar:       jar,
		Timeout:   timeout,
		Transport: opts.Transport,
	}
	follow := true
	if opts.FollowRedirects != nil {
		follow = *opts.FollowRedirects
	}
	if !follow {
		client.CheckRedirect = func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return &Browser{
		client:    client,
		jar:       jar,
		userAgent: ua,
		headers:   headers,
		timeout:   timeout,
	}, nil
}

// UserAgent returns the browser's configured user-agent string.
func (b *Browser) UserAgent() string { return b.userAgent }

// SetUserAgent changes the user-agent sent on subsequent navigations.
func (b *Browser) SetUserAgent(ua string) { b.userAgent = ua }

// SetExtraHTTPHeaders replaces the browser-level extra headers.
func (b *Browser) SetExtraHTTPHeaders(h map[string]string) {
	b.headers = http.Header{}
	for k, v := range h {
		b.headers.Set(k, v)
	}
}

// Cookies returns the cookies the jar would send for rawurl.
func (b *Browser) Cookies(rawurl string) ([]*http.Cookie, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, fmt.Errorf("puppeteer: parsing url: %w", err)
	}
	return b.jar.Cookies(u), nil
}

// SetCookies stores cookies in the jar for rawurl.
func (b *Browser) SetCookies(rawurl string, cookies []*http.Cookie) error {
	u, err := url.Parse(rawurl)
	if err != nil {
		return fmt.Errorf("puppeteer: parsing url: %w", err)
	}
	b.jar.SetCookies(u, cookies)
	return nil
}

// NewPage returns a fresh Page bound to this browser.
func (b *Browser) NewPage() *Page {
	return &Page{browser: b, extraHeaders: http.Header{}}
}

// Close releases idle connections. The browser must not be used afterwards.
func (b *Browser) Close() error {
	b.client.CloseIdleConnections()
	return nil
}

// Response captures the result of a navigation.
type Response struct {
	URL        *url.URL
	Status     string
	StatusCode int
	Header     http.Header
	Body       []byte
}

// OK reports whether the response status is in the 2xx range.
func (r *Response) OK() bool { return r.StatusCode >= 200 && r.StatusCode < 300 }

// Page is a single navigable document. It holds the most recently fetched HTML,
// its parsed DOM tree and the resolved URL, and exposes selection, link and
// form helpers over that document.
type Page struct {
	browser      *Browser
	extraHeaders http.Header
	userAgent    string // per-page override; empty means use the browser's

	url  *url.URL
	doc  *Node
	body []byte
	resp *Response
}

// SetExtraHTTPHeaders sets page-level headers that augment the browser's.
func (p *Page) SetExtraHTTPHeaders(h map[string]string) {
	p.extraHeaders = http.Header{}
	for k, v := range h {
		p.extraHeaders.Set(k, v)
	}
}

// SetUserAgent overrides the user agent for this page only.
func (p *Page) SetUserAgent(ua string) { p.userAgent = ua }

// Goto navigates the page to rawurl using GET and parses the response body as
// HTML. Relative URLs are resolved against the page's current URL when one
// exists.
func (p *Page) Goto(rawurl string) (*Response, error) {
	return p.GotoContext(context.Background(), rawurl)
}

// GotoContext is Goto with an explicit context for cancellation.
func (p *Page) GotoContext(ctx context.Context, rawurl string) (*Response, error) {
	u, err := p.resolve(rawurl)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("puppeteer: building request: %w", err)
	}
	return p.do(req)
}

// resolve turns rawurl into an absolute URL relative to the current page.
func (p *Page) resolve(rawurl string) (*url.URL, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, fmt.Errorf("puppeteer: parsing url %q: %w", rawurl, err)
	}
	if u.IsAbs() {
		return u, nil
	}
	if p.url != nil {
		return p.url.ResolveReference(u), nil
	}
	return nil, fmt.Errorf("puppeteer: cannot resolve relative url %q without a current page", rawurl)
}

// applyHeaders sets the user agent and merged extra headers on req.
func (p *Page) applyHeaders(req *http.Request) {
	ua := p.userAgent
	if ua == "" {
		ua = p.browser.userAgent
	}
	req.Header.Set("User-Agent", ua)
	for k, vs := range p.browser.headers {
		for _, v := range vs {
			req.Header.Set(k, v)
		}
	}
	for k, vs := range p.extraHeaders {
		for _, v := range vs {
			req.Header.Set(k, v)
		}
	}
}

// do sends req through the browser client and loads the resulting document.
func (p *Page) do(req *http.Request) (*Response, error) {
	p.applyHeaders(req)
	resp, err := p.browser.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("puppeteer: navigation failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("puppeteer: reading body: %w", err)
	}

	final := resp.Request.URL
	p.url = final
	p.body = body
	p.doc = Parse(string(body))
	r := &Response{
		URL:        final,
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       body,
	}
	p.resp = r
	return r, nil
}

// URL returns the page's current absolute URL, or "" before any navigation.
func (p *Page) URL() string {
	if p.url == nil {
		return ""
	}
	return p.url.String()
}

// Content returns the raw HTML body of the current page.
func (p *Page) Content() string { return string(p.body) }

// HTML returns the serialized DOM of the current page.
func (p *Page) HTML() string {
	if p.doc == nil {
		return ""
	}
	var sb strings.Builder
	p.doc.render(&sb)
	return sb.String()
}

// Document returns the root DocumentNode of the parsed page, or nil.
func (p *Page) Document() *Node { return p.doc }

// Title returns the trimmed text of the document's <title>, if any.
func (p *Page) Title() string {
	el, _ := p.QuerySelector("title")
	if el == nil {
		return ""
	}
	return strings.TrimSpace(el.TextContent())
}

// QuerySelector returns the first element matching selector, or nil.
func (p *Page) QuerySelector(selector string) (*Element, error) {
	if p.doc == nil {
		return nil, errNoDocument
	}
	sel, err := compileSelector(selector)
	if err != nil {
		return nil, err
	}
	if n := sel.queryFirst(p.doc); n != nil {
		return wrapElement(n), nil
	}
	return nil, nil
}

// QuerySelectorAll returns all elements matching selector in document order.
func (p *Page) QuerySelectorAll(selector string) ([]*Element, error) {
	if p.doc == nil {
		return nil, errNoDocument
	}
	sel, err := compileSelector(selector)
	if err != nil {
		return nil, err
	}
	return wrapElements(sel.queryAll(p.doc)), nil
}

// Links returns every href on the page resolved to an absolute URL. Duplicates
// are removed while preserving first-seen order.
func (p *Page) Links() []string {
	els, _ := p.QuerySelectorAll("a[href]")
	seen := map[string]bool{}
	var out []string
	for _, e := range els {
		href, _ := e.Attr("href")
		abs, err := p.resolve(strings.TrimSpace(href))
		if err != nil {
			continue
		}
		s := abs.String()
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// WaitForSelectorOptions configures WaitForSelector.
type WaitForSelectorOptions struct {
	// Timeout bounds the total wait. Zero means 5s.
	Timeout time.Duration
	// Interval is the delay between re-fetches. Zero means 250ms.
	Interval time.Duration
}

// WaitForSelector re-fetches the current URL on an interval until selector
// matches or the timeout elapses. Because this package executes no JavaScript,
// a page's DOM cannot change on its own; this helper is therefore only useful
// for content that varies across identical requests (polling an endpoint). It
// returns the first matching element.
func (p *Page) WaitForSelector(selector string, opts *WaitForSelectorOptions) (*Element, error) {
	timeout := 5 * time.Second
	interval := 250 * time.Millisecond
	if opts != nil {
		if opts.Timeout > 0 {
			timeout = opts.Timeout
		}
		if opts.Interval > 0 {
			interval = opts.Interval
		}
	}
	if p.url == nil {
		return nil, errNoDocument
	}
	deadline := time.Now().Add(timeout)
	current := p.url.String()
	for {
		if el, err := p.QuerySelector(selector); err != nil {
			return nil, err
		} else if el != nil {
			return el, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("puppeteer: timed out waiting for selector %q", selector)
		}
		time.Sleep(interval)
		if _, err := p.Goto(current); err != nil {
			return nil, err
		}
	}
}

var errNoDocument = errors.New("puppeteer: no document loaded; call Goto first")
