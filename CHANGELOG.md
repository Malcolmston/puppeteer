# Changelog

All notable changes to this project are documented here. This project adheres to
semantic versioning.

## [0.2.0]

Added, toward closer parity with the Node.js Puppeteer/Playwright API, using the
Go standard library only (no third-party imports, no cgo):

### Session-history navigation (`navigation.go`)
- `Page.Reload` / `Page.ReloadContext` — re-fetch and re-parse the current URL
  without creating a new history entry (Puppeteer `page.reload`).
- `Page.GoBack` / `Page.GoBackContext` and `Page.GoForward` /
  `Page.GoForwardContext` — walk the recorded session history, re-fetching the
  target URL (Puppeteer `page.goBack` / `page.goForward`); resolve to `nil` at
  the ends of the list.
- `Page.CanGoBack`, `Page.CanGoForward`, `Page.History` — inspect the history
  cursor and the ordered list of visited URLs.

### Page scraping helpers (`page_extract.go`)
- `Page.Metas` and `Page.MetaContent` — read `<meta>` metadata (including Open
  Graph `property` tags).
- `Page.Images`, `Page.Scripts`, `Page.Stylesheets` — enumerate sub-resource
  URLs resolved to absolute form and de-duplicated.
- `Page.Count`, `Page.TextContent`, `Page.GetAttribute` — Playwright-style
  first-match convenience accessors.
- `Page.Cookies` — cookies the jar would send for the current URL
  (Puppeteer `page.cookies`).

### Element DOM helpers (`element_extra.go`)
- `Element.HasAttribute`, `Element.AttributeNames` — attribute introspection.
- `Element.Dataset` — the `data-*` attributes as a camelCased dataset map.
- `Element.Siblings`, `Element.NextAll`, `Element.PrevAll`, `Element.Ancestors`
  — jQuery-like traversal collections in document order.
- `Element.IsEmpty` — CSS `:empty`-style emptiness test.

All new exported identifiers ship with complete godoc and deterministic
known-answer tests; `Element.Dataset` has an accompanying benchmark.

## [0.1.0]

- Initial release: `Browser`/`Page`/`Element`/`Form` model, original HTML
  tokenizer and parser, and a CSS selector engine.
