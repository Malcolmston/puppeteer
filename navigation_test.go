package puppeteer

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

// ppHistServer serves /1../9, each page reporting its own path and a hit counter
// so reloads are observable.
func ppHistServer(t *testing.T) (*Browser, *httptest.Server, string) {
	t.Helper()
	hits := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits[r.URL.Path]++
		_, _ = io.WriteString(w, fmt.Sprintf(
			`<html><head><title>P%s</title></head><body><p id="hits">%d</p></body></html>`,
			r.URL.Path, hits[r.URL.Path]))
	}))
	browser, err := Launch(&LaunchOptions{Transport: srv.Client().Transport})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	return browser, srv, srv.URL
}

func TestHistoryBackForward(t *testing.T) {
	browser, srv, base := ppHistServer(t)
	defer srv.Close()
	page := browser.NewPage()

	for _, path := range []string{"/1", "/2", "/3"} {
		if _, err := page.Goto(base + path); err != nil {
			t.Fatalf("Goto %s: %v", path, err)
		}
	}
	if got, want := page.History(), []string{base + "/1", base + "/2", base + "/3"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("History = %v, want %v", got, want)
	}
	if page.Title() != "P/3" {
		t.Fatalf("title = %q, want P/3", page.Title())
	}
	if !page.CanGoBack() || page.CanGoForward() {
		t.Fatalf("expected CanGoBack && !CanGoForward at end of history")
	}

	// Back to /2.
	if resp, err := page.GoBack(); err != nil || resp == nil {
		t.Fatalf("GoBack: %v", err)
	}
	if page.URL() != base+"/2" || page.Title() != "P/2" {
		t.Fatalf("after GoBack url=%q title=%q", page.URL(), page.Title())
	}
	if !page.CanGoForward() {
		t.Fatal("CanGoForward should be true after going back")
	}

	// Back to /1.
	if _, err := page.GoBack(); err != nil {
		t.Fatalf("GoBack: %v", err)
	}
	if page.URL() != base+"/1" {
		t.Fatalf("url = %q, want /1", page.URL())
	}
	if page.CanGoBack() {
		t.Fatal("CanGoBack should be false at oldest entry")
	}
	// No earlier entry: (nil, nil).
	if resp, err := page.GoBack(); resp != nil || err != nil {
		t.Fatalf("GoBack at start = %v,%v want nil,nil", resp, err)
	}

	// Forward to /2.
	if _, err := page.GoForward(); err != nil {
		t.Fatalf("GoForward: %v", err)
	}
	if page.URL() != base+"/2" {
		t.Fatalf("url = %q, want /2", page.URL())
	}

	// Navigating anew truncates the forward history (/3 is discarded).
	if _, err := page.Goto(base + "/4"); err != nil {
		t.Fatalf("Goto /4: %v", err)
	}
	if got, want := page.History(), []string{base + "/1", base + "/2", base + "/4"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("History after truncation = %v, want %v", got, want)
	}
	if page.CanGoForward() {
		t.Fatal("CanGoForward should be false after a fresh navigation")
	}
}

func TestReload(t *testing.T) {
	browser, srv, base := ppHistServer(t)
	defer srv.Close()
	page := browser.NewPage()

	if _, err := page.Goto(base + "/r"); err != nil {
		t.Fatalf("Goto: %v", err)
	}
	hits, _ := page.QuerySelector("#hits")
	if hits.TextContent() != "1" {
		t.Fatalf("first hit count = %q, want 1", hits.TextContent())
	}
	if _, err := page.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	hits, _ = page.QuerySelector("#hits")
	if hits.TextContent() != "2" {
		t.Fatalf("reloaded hit count = %q, want 2", hits.TextContent())
	}
	// Reload must not add a history entry.
	if got := page.History(); !reflect.DeepEqual(got, []string{base + "/r"}) {
		t.Fatalf("History after reload = %v, want one entry", got)
	}
}

func TestReloadWithoutNavigationErrors(t *testing.T) {
	browser, _ := Launch(nil)
	page := browser.NewPage()
	if _, err := page.Reload(); err == nil {
		t.Error("Reload before any navigation should error")
	}
}
