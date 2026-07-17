// Library content for the puppeteer documentation site. Mirrors the shape used by
// the malcolmston/go landing site's data.ts so the sibling sites stay in sync.
export interface Lib {
  id: string; name: string; icon: string; accent: string; pkg: string; node: string;
  repo: string; docs: string; tagline: string; blurb: string; tags: string[];
  features: string[]; node_code: string; go_code: string; integrate: string;
}

export const NODE_ACCENT = '#8cc84b';

export const PUPPETEER: Lib = {
  id:"puppeteer", name:"puppeteer", icon:'<i class="fa-solid fa-masks-theater"></i>', accent:"#40b5a4",
  pkg:"github.com/malcolmston/puppeteer", node:"puppeteer/puppeteer",
  repo:"https://github.com/malcolmston/puppeteer", docs:"https://malcolmston.github.io/puppeteer/",
  tagline:"Puppeteer-style page automation for Go, standard library only.",
  blurb:"A from-scratch, standard-library-only Go toolkit inspired by the Node.js Puppeteer API. It fetches "+
    "pages over net/http, parses the HTML with its own tokenizer and DOM builder, and queries the document "+
    "with a real CSS selector engine — no cgo, no third-party modules, not even golang.org/x/net. "+
    "Because a dependency-free Go library cannot embed a browser, it deliberately runs NO JavaScript and does "+
    "NO rendering: there is no script execution, no layout/geometry/screenshots, and no live DOM — the tree is "+
    "a static snapshot of the bytes the server sent. In practice it behaves like an HTTP client married to an "+
    "HTML parser and a selector engine, ideal for scraping and automating server-rendered pages, following "+
    "links, submitting forms and managing cookies and headers. Launch returns a Browser (cookie jar, shared "+
    "headers, user agent, per-navigation timeout); Browser.NewPage returns a Page whose Goto fetches and parses "+
    "a URL, and from which you select nodes, enumerate resolved Links and discover, fill and submit Forms. "+
    "The import path is github.com/malcolmston/puppeteer and the package is named puppeteer.",
  tags:["net/http","no JavaScript","no rendering","cookie jar","HTML tokenizer","DOM builder","CSS selectors",":nth-child(An+B)","Links","forms","zero deps"],
  features:[
    "<code>Launch</code> a <code>Browser</code> with a <code>LaunchOptions</code> cookie jar, shared headers, user agent, per-navigation timeout and custom transport",
    "<code>Browser.NewPage</code> returns a <code>Page</code>; <code>Page.Goto</code> / <code>GotoContext</code> fetch over <code>net/http</code>, follow redirects and update cookies",
    "Own HTML tokenizer + DOM builder — <code>Parse</code> never fails, recovering malformed input the way browsers do into a <code>Node</code> tree",
    "A real CSS selector engine — <code>QuerySelector</code> and <code>QuerySelectorAll</code> supporting type/<code>*</code>, <code>#id</code>, <code>.class</code>, attribute and combinator selectors",
    "Structural pseudo-classes including the <code>:nth-child()</code> <code>An+B</code> microsyntax (<code>odd</code>, <code>even</code>, <code>2n+1</code>, <code>-n+3</code>)",
    "<code>Element</code> handles — <code>TextContent</code>, <code>InnerHTML</code>, <code>OuterHTML</code>, <code>Attr</code>/<code>Attributes</code>, <code>ClassList</code>/<code>HasClass</code> and node-relative queries",
    "DOM traversal — <code>Children</code>, <code>Parent</code>, <code>Next</code>, <code>Prev</code>, <code>Closest</code> and <code>Matches</code>",
    "Resolved <code>Links</code> — every <code>a[href]</code> turned into an absolute URL against the page's location",
    "Form automation — <code>Forms</code>, <code>FormBySelector</code>, <code>FillForm</code>, then <code>BuildRequest</code> or <code>Submit</code> (GET query or POST body)",
    "Cookies &amp; headers — <code>Cookies</code>, <code>SetCookies</code>, <code>SetUserAgent</code>, <code>SetExtraHTTPHeaders</code> over a <code>net/http/cookiejar</code>",
    "<b>No JavaScript, no rendering</b> — no script execution, no layout/geometry/screenshots, no live DOM; a static snapshot of server-sent bytes",
    "Zero dependencies — pure Go standard library, nothing to audit but the toolchain"
  ],
  node_code:
`import puppeteer from "puppeteer";

const browser = await puppeteer.launch();
const page = await browser.newPage();
await page.goto("https://example.com");

console.log("title:", await page.title());

const links = await page.$$eval("a[href]", (as) =>
  as.map((a) => [a.getAttribute("href"), a.textContent]),
);
for (const [href, text] of links) console.log("link:", href, "->", text);

await browser.close();`,
  go_code:
`import "github.com/malcolmston/puppeteer"

browser, _ := puppeteer.Launch(nil)
defer browser.Close()

page := browser.NewPage()
if _, err := page.Goto("https://example.com"); err != nil {
	log.Fatal(err)
}

fmt.Println("title:", page.Title())

links, _ := page.QuerySelectorAll("a[href]")
for _, a := range links {
	href, _ := a.Attr("href")
	fmt.Println("link:", href, "->", a.TextContent())
}`,
  integrate:
`<span class="tok-c">// Launch a browser with a custom user agent and a shared header; the</span>
<span class="tok-c">// cookie jar stores server-set cookies and replays them automatically.</span>
browser, _ := puppeteer.Launch(&puppeteer.LaunchOptions{
	UserAgent: "my-scraper/1.0",
	Headers:   map[string]string{"Accept-Language": "en"},
})
defer browser.Close()

page := browser.NewPage()
page.Goto("https://example.com/login")

<span class="tok-c">// Discover a form by selector, fill named fields, and submit it —</span>
<span class="tok-c">// FillForm returns a Form you can Submit (GET query or POST body).</span>
form, _ := page.FillForm("#login", map[string]string{
	"username": "alice",
	"password": "hunter2",
})
resp, _ := form.Submit()
fmt.Println("status:", resp.StatusCode)

<span class="tok-c">// Query with the selector engine, including :nth-child(An+B), then</span>
<span class="tok-c">// traverse the static DOM snapshot and read attributes.</span>
rows, _ := page.QuerySelectorAll("table.results tr:nth-child(odd)")
for _, tr := range rows {
	if link, _ := tr.QuerySelector("a[href]"); link != nil {
		href, _ := link.Attr("href")
		fmt.Println(link.TextContent(), "->", href)
	}
}

<span class="tok-c">// Every href resolved to an absolute URL against the page location.</span>
fmt.Println(page.Links())`
};
