# puppeteer

Headless page-automation toolkit for Go, inspired by the Node.js
[Puppeteer](https://pptr.dev/) API — implemented with **only the Go standard
library** (no cgo, no third-party modules, not even `golang.org/x/net`).

It fetches pages over `net/http`, parses the HTML with its own tokenizer and DOM
builder, and lets you query and traverse the document with a real CSS selector
engine.

## Important: no JavaScript, no rendering

A dependency-free Go library **cannot run a browser or execute JavaScript**.
This package therefore has:

- **no script execution** — content injected or mutated by client-side JS is
  never present in the DOM you query;
- **no rendering/layout** — no geometry, computed styles, screenshots, or
  visibility;
- **no live DOM** — the tree is a static snapshot of the bytes the server sent.

In practice it behaves like an HTTP client married to an HTML parser and a CSS
selector engine. It is great for scraping and automating **server-rendered**
pages, following links, submitting forms, and managing cookies and headers. It
is **not** a substitute for headless Chromium when a site depends on JavaScript.

## Install

```sh
go get github.com/malcolmston/puppeteer
```

Requires Go 1.24+.

## Quick start

```go
package main

import (
	"fmt"
	"log"

	"github.com/malcolmston/puppeteer"
)

func main() {
	browser, err := puppeteer.Launch(nil)
	if err != nil {
		log.Fatal(err)
	}
	defer browser.Close()

	page := browser.NewPage()
	if _, err := page.Goto("https://example.com"); err != nil {
		log.Fatal(err)
	}

	fmt.Println("title:", page.Title())

	// Select the first matching element.
	h1, _ := page.QuerySelector("h1")
	if h1 != nil {
		fmt.Println("heading:", h1.TextContent())
	}

	// Select many and read attributes.
	links, _ := page.QuerySelectorAll("a[href]")
	for _, a := range links {
		href, _ := a.Attr("href")
		fmt.Println("link:", href, "->", a.TextContent())
	}

	// Every href resolved to an absolute URL.
	fmt.Println(page.Links())
}
```

### Cookies, headers and user agent

```go
browser, _ := puppeteer.Launch(&puppeteer.LaunchOptions{
	UserAgent: "my-scraper/1.0",
	Headers:   map[string]string{"Accept-Language": "en"},
})
// Cookies set by the server are stored in a net/http/cookiejar and sent back
// automatically on subsequent navigations.
cookies, _ := browser.Cookies("https://example.com/")
```

### Forms

```go
page.Goto("https://example.com/login")

form, _ := page.FillForm("#login", map[string]string{
	"username": "alice",
	"password": "hunter2",
})

// Inspect or build the request without sending it...
req, _ := form.BuildRequest(context.Background()) // GET query or POST body

// ...or submit it and load the response into the page.
resp, _ := form.Submit()
fmt.Println(resp.StatusCode)
```

## Supported CSS selectors

- Type (`div`) and universal (`*`)
- `#id`, `.class`
- Attributes: `[attr]`, `[attr=val]`, `[attr^=val]`, `[attr$=val]`,
  `[attr*=val]`, `[attr~=val]`, `[attr|=val]`
- Combinators: descendant (space), child (`>`), adjacent sibling (`+`), general
  sibling (`~`)
- Selector lists (grouping with `,`)
- Structural pseudo-classes: `:first-child`, `:last-child`, `:nth-child()`
  including the `An+B` microsyntax (`odd`, `even`, `2n+1`, `-n+3`, …)

## API overview

- `Launch(*LaunchOptions) (*Browser, error)` — cookie jar, headers, user agent,
  timeout, redirect policy, custom transport.
- `Browser`: `NewPage`, `Cookies`, `SetCookies`, `SetUserAgent`,
  `SetExtraHTTPHeaders`, `Close`.
- `Page`: `Goto`/`GotoContext`, `Content`, `HTML`, `Title`, `URL`, `Document`,
  `QuerySelector`, `QuerySelectorAll`, `Links`, `Forms`, `FormBySelector`,
  `FillForm`, `WaitForSelector`.
- `Element`: `TagName`, `TextContent`, `InnerText`, `InnerHTML`, `OuterHTML`,
  `Attr`, `AttrOr`, `Attributes`, `ID`, `ClassList`, `HasClass`, `Children`,
  `Parent`, `Next`, `Prev`, `Closest`, `Matches`, `QuerySelector`,
  `QuerySelectorAll`, `Node`.
- `Form`: `Get`, `Set`, `Values`, `FieldNames`, `BuildRequest`, `Submit`.
- Low-level: `Parse(html string) *Node` and the `Node`/`Attribute` DOM types.

See the package documentation (`doc.go`) for full details.

## Development

```sh
go build ./...
go vet ./...
go test ./...          # deterministic, driven by httptest
```

## License

See repository.
