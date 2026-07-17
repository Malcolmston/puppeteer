// Package puppeteer is a pure-Go, standard-library-only page-automation toolkit
// inspired by the Node.js Puppeteer API. It fetches pages over net/http, parses
// the returned HTML with its own tokenizer and DOM builder, and lets you query
// and traverse the resulting document with a real CSS selector engine.
//
// # No JavaScript, no rendering
//
// This package deliberately does NOT run a browser and does NOT execute
// JavaScript. A dependency-free Go library cannot embed a JavaScript engine or
// a layout/rendering engine, so there is:
//
//   - no script execution: content injected or mutated by client-side JS is
//     never present in the DOM you query;
//   - no rendering, layout, or painting: there are no element geometries,
//     computed styles, screenshots, or visibility calculations;
//   - no live DOM: the tree is a static snapshot of the bytes the server sent.
//
// In practical terms, puppeteer here behaves like an HTTP client married to an
// HTML parser and selector engine. It is ideal for scraping and automating
// server-rendered pages, following links, submitting forms, and managing
// cookies and headers. It is NOT a substitute for a headless Chromium when a
// site's content depends on JavaScript. The WaitForSelector helper polls by
// re-fetching the URL rather than observing a live DOM (see its documentation).
//
// # Model
//
// Launch returns a [Browser], which owns an *http.Client, a cookie jar
// (net/http/cookiejar), shared headers, a user agent and a per-navigation
// timeout. Browser.NewPage returns a [Page]. Page.Goto fetches a URL (following
// redirects, updating cookies), reads the body and parses it into a DOM tree.
//
// From a Page you can:
//
//   - select nodes with QuerySelector / QuerySelectorAll;
//   - read Content (raw HTML), HTML (re-serialized DOM), Title and URL;
//   - enumerate resolved Links;
//   - discover and fill Forms, then build or submit the resulting request.
//
// Selected nodes are returned as [Element] handles offering TextContent,
// InnerHTML, OuterHTML, Attr/Attributes, class helpers and DOM traversal
// (Children, Parent, Next, Prev, Closest, Matches) plus node-relative
// QuerySelector/QuerySelectorAll.
//
// # Supported CSS selectors
//
// The selector engine ([compileSelector] and the QuerySelector methods)
// implements:
//
//   - type selectors (div), the universal selector (*);
//   - #id and .class;
//   - attribute selectors [attr], [attr=val], [attr^=val], [attr$=val],
//     [attr*=val], [attr~=val], [attr|=val];
//   - combinators: descendant (space), child (>), adjacent sibling (+) and
//     general sibling (~);
//   - selector lists (grouping with commas);
//   - structural pseudo-classes :first-child, :last-child and :nth-child(),
//     including the An+B microsyntax (odd, even, 2n+1, -n+3, ...).
//
// # HTML parsing
//
// The tokenizer and parser are original code (they do not use golang.org/x/net).
// They handle tags, quoted/unquoted attributes, comments, doctypes, void
// elements, raw-text elements (script, style), common character references and
// the most frequent implicit end-tag rules (for example <li> closing a previous
// <li>). Parse never fails; malformed input is recovered the way browsers do.
//
// # Example
//
// A minimal end-to-end use looks like this:
//
//	browser, _ := puppeteer.Launch(nil)
//	page := browser.NewPage()
//	if _, err := page.Goto("https://example.com"); err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(page.Title())
//	links := page.Links()
//	_ = links
//
// See the package examples for a runnable version backed by httptest.
package puppeteer
