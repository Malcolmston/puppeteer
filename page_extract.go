package puppeteer

import (
	"net/http"
	"strings"
)

// This file adds page-level scraping helpers that mirror common Puppeteer and
// Playwright conveniences: reading document metadata, enumerating sub-resource
// URLs, counting matches, and pulling text or a single attribute from the first
// element matching a selector — all resolved against the current document.

// Metas returns the document's <meta> metadata as a name-to-content map. Both
// the name attribute and the Open Graph style property attribute act as keys
// (property takes precedence when both are present on one tag); tags without a
// content attribute or without either key are skipped. Later duplicates win.
func (p *Page) Metas() map[string]string {
	out := map[string]string{}
	els, _ := p.QuerySelectorAll("meta")
	for _, e := range els {
		content, ok := e.Attr("content")
		if !ok {
			continue
		}
		key := e.AttrOr("property", "")
		if key == "" {
			key = e.AttrOr("name", "")
		}
		if key == "" {
			continue
		}
		out[key] = content
	}
	return out
}

// MetaContent returns the content of the first <meta> tag whose name or property
// attribute equals key (case-insensitive), and whether such a tag was found.
func (p *Page) MetaContent(key string) (string, bool) {
	want := strings.ToLower(strings.TrimSpace(key))
	els, _ := p.QuerySelectorAll("meta")
	for _, e := range els {
		name := strings.ToLower(e.AttrOr("name", ""))
		prop := strings.ToLower(e.AttrOr("property", ""))
		if name == want || prop == want {
			return e.AttrOr("content", ""), true
		}
	}
	return "", false
}

// Images returns every <img> source on the page resolved to an absolute URL,
// de-duplicated while preserving first-seen order.
func (p *Page) Images() []string {
	return p.resolvedResourceURLs("img[src]", "src")
}

// Scripts returns every external <script src> on the page resolved to an
// absolute URL, de-duplicated while preserving first-seen order. Inline scripts
// (which have no src) are omitted.
func (p *Page) Scripts() []string {
	return p.resolvedResourceURLs("script[src]", "src")
}

// Stylesheets returns the href of every <link rel="stylesheet"> resolved to an
// absolute URL, de-duplicated while preserving first-seen order.
func (p *Page) Stylesheets() []string {
	return p.resolvedResourceURLs(`link[rel~=stylesheet][href]`, "href")
}

// resolvedResourceURLs is the shared implementation behind Images, Scripts and
// Stylesheets. It selects elements, reads attr, resolves each value against the
// page URL and returns the unique absolute URLs in document order.
func (p *Page) resolvedResourceURLs(selector, attr string) []string {
	els, _ := p.QuerySelectorAll(selector)
	seen := map[string]bool{}
	var out []string
	for _, e := range els {
		raw, ok := e.Attr(attr)
		if !ok {
			continue
		}
		abs, err := p.resolve(strings.TrimSpace(raw))
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

// Count returns the number of elements matching selector in the current
// document. It reports an error only when the selector is invalid or no
// document is loaded.
func (p *Page) Count(selector string) (int, error) {
	els, err := p.QuerySelectorAll(selector)
	if err != nil {
		return 0, err
	}
	return len(els), nil
}

// TextContent returns the concatenated text of the first element matching
// selector, mirroring Playwright's page.textContent. The boolean is false when
// no element matches; an error is returned only for an invalid selector or a
// missing document.
func (p *Page) TextContent(selector string) (string, bool, error) {
	el, err := p.QuerySelector(selector)
	if err != nil {
		return "", false, err
	}
	if el == nil {
		return "", false, nil
	}
	return el.TextContent(), true, nil
}

// GetAttribute returns the value of attribute name on the first element matching
// selector, mirroring Playwright's page.getAttribute. The boolean is false when
// no element matches or the element lacks the attribute; an error is returned
// only for an invalid selector or a missing document.
func (p *Page) GetAttribute(selector, name string) (string, bool, error) {
	el, err := p.QuerySelector(selector)
	if err != nil {
		return "", false, err
	}
	if el == nil {
		return "", false, nil
	}
	v, ok := el.Attr(name)
	return v, ok, nil
}

// Cookies returns the cookies the browser's jar would send for the page's
// current URL, mirroring Puppeteer's page.cookies. It returns an error when no
// page has been loaded yet.
func (p *Page) Cookies() ([]*http.Cookie, error) {
	if p.url == nil {
		return nil, errNoDocument
	}
	return p.browser.jar.Cookies(p.url), nil
}
