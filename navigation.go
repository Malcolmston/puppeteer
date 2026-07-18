package puppeteer

import (
	"context"
	"fmt"
)

// This file adds Puppeteer-style session-history navigation to Page. The Node
// Puppeteer API exposes page.reload, page.goBack and page.goForward, which walk
// the browser's back/forward list. Because this package keeps a static snapshot
// per navigation, "history" is the ordered list of URLs Goto (and Submit) have
// successfully loaded; GoBack and GoForward move a cursor over that list and
// re-fetch the corresponding URL. Redirects are followed exactly as a normal
// navigation would follow them.

// Reload re-fetches the current URL and re-parses the document, mirroring
// Puppeteer's page.reload. The history cursor is left unchanged: a reload does
// not create a new history entry. It returns an error if no page has been
// loaded yet.
func (p *Page) Reload() (*Response, error) {
	return p.ReloadContext(context.Background())
}

// ReloadContext is Reload with an explicit context for cancellation.
func (p *Page) ReloadContext(ctx context.Context) (*Response, error) {
	if p.url == nil {
		return nil, errNoDocument
	}
	current := p.url.String()
	p.histLock = true
	defer func() { p.histLock = false }()
	return p.GotoContext(ctx, current)
}

// CanGoBack reports whether there is an earlier entry in the session history to
// navigate back to.
func (p *Page) CanGoBack() bool { return p.histIdx > 0 }

// CanGoForward reports whether there is a later entry in the session history to
// navigate forward to.
func (p *Page) CanGoForward() bool {
	return len(p.history) > 0 && p.histIdx < len(p.history)-1
}

// GoBack navigates to the previous entry in the session history, re-fetching it
// like Puppeteer's page.goBack. It returns (nil, nil) when there is no earlier
// entry, matching Puppeteer's convention of resolving to null.
func (p *Page) GoBack() (*Response, error) {
	return p.GoBackContext(context.Background())
}

// GoBackContext is GoBack with an explicit context for cancellation.
func (p *Page) GoBackContext(ctx context.Context) (*Response, error) {
	if !p.CanGoBack() {
		return nil, nil
	}
	target := p.history[p.histIdx-1]
	resp, err := p.navigateHistory(ctx, target)
	if err != nil {
		return nil, err
	}
	p.histIdx--
	return resp, nil
}

// GoForward navigates to the next entry in the session history, re-fetching it
// like Puppeteer's page.goForward. It returns (nil, nil) when there is no later
// entry.
func (p *Page) GoForward() (*Response, error) {
	return p.GoForwardContext(context.Background())
}

// GoForwardContext is GoForward with an explicit context for cancellation.
func (p *Page) GoForwardContext(ctx context.Context) (*Response, error) {
	if !p.CanGoForward() {
		return nil, nil
	}
	target := p.history[p.histIdx+1]
	resp, err := p.navigateHistory(ctx, target)
	if err != nil {
		return nil, err
	}
	p.histIdx++
	return resp, nil
}

// History returns a copy of the URLs in the session history, oldest first. The
// current entry is History()[index] where index can be derived from the length
// and the availability reported by CanGoBack/CanGoForward.
func (p *Page) History() []string {
	out := make([]string, len(p.history))
	copy(out, p.history)
	return out
}

// navigateHistory fetches target without disturbing the recorded history, so
// that the caller can adjust the cursor deterministically.
func (p *Page) navigateHistory(ctx context.Context, target string) (*Response, error) {
	p.histLock = true
	defer func() { p.histLock = false }()
	resp, err := p.GotoContext(ctx, target)
	if err != nil {
		return nil, fmt.Errorf("puppeteer: history navigation to %q: %w", target, err)
	}
	return resp, nil
}
