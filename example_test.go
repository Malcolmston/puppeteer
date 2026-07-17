package puppeteer_test

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/malcolmston/puppeteer"
)

// Example demonstrates launching a browser, navigating to a page served by
// httptest, and selecting nodes with the CSS selector engine.
func Example() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `<!DOCTYPE html><html><head><title>Demo</title></head>
<body>
  <h1 class="title">Fruits</h1>
  <ul>
    <li class="fruit">Apple</li>
    <li class="fruit selected">Banana</li>
    <li class="fruit">Cherry</li>
  </ul>
</body></html>`)
	}))
	defer srv.Close()

	browser, err := puppeteer.Launch(nil)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = browser.Close() }()

	page := browser.NewPage()
	if _, err := page.Goto(srv.URL); err != nil {
		log.Fatal(err)
	}

	fmt.Println("title:", page.Title())

	h1, _ := page.QuerySelector("h1.title")
	fmt.Println("heading:", h1.TextContent())

	fruits, _ := page.QuerySelectorAll("li.fruit")
	for _, f := range fruits {
		fmt.Println("fruit:", f.TextContent())
	}

	selected, _ := page.QuerySelector("li:nth-child(2)")
	fmt.Println("selected:", selected.TextContent())

	// Output:
	// title: Demo
	// heading: Fruits
	// fruit: Apple
	// fruit: Banana
	// fruit: Cherry
	// selected: Banana
}
