package spielerplus

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"
)

// jQuery's default XHR Accept header, matched here since the HAR capture
// this client is grounded in shows the real frontend sending exactly this
// for its ajax calls (SpielerPlus may vary behavior on Accept).
const ajaxAcceptHeader = "application/json, text/javascript, */*; q=0.01"

// BaseURL is SpielerPlus's web app. There is no separate API host - the
// client scrapes the same server-rendered pages a browser would.
const BaseURL = "https://www.spielerplus.de"

// browserUserAgent works around SpielerPlus returning a bare 403 for the
// default Go/requests/etc. User-Agent, before login is even attempted -
// documented by the community projects this client's login flow and
// selectors are modeled after (christianwehe/calendar-sync, janic0/autospieler).
const browserUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

// ErrNotAuthenticated is returned when a request lands on SpielerPlus's
// login page instead of the requested content, meaning the session cookie
// is missing, invalid, or expired.
var ErrNotAuthenticated = fmt.Errorf("spielerplus: not authenticated (session cookie missing/expired) - capture a fresh SPIELERPLUS_SESSION_COOKIE from a logged-in browser session")

// DefaultRequestDelay is the minimum gap enforced between requests when the
// operator hasn't configured one explicitly (see WithRequestDelay) - a
// conservative default given this importer can issue a lot of requests in a
// tight loop (one per member's profile page, one per event's attendance),
// and there's no published rate limit to target instead.
const DefaultRequestDelay = 500 * time.Millisecond

// Client is a minimal HTTP client for the SpielerPlus pages this importer
// needs. It is read-only against SpielerPlus: it never submits the
// participation form or otherwise mutates SpielerPlus data.
type Client struct {
	httpClient *http.Client
	baseURL    string

	// requestDelay is the minimum gap enforced between the start of one
	// request and the next (see WithRequestDelay), so a long member/event
	// list doesn't hammer SpielerPlus in a tight loop. mu+lastRequest
	// implement that serially - this importer never issues concurrent
	// requests, so a simple mutex is enough (no need for a token-bucket
	// library).
	requestDelay time.Duration
	mu           sync.Mutex
	lastRequest  time.Time
}

// ClientOption configures optional Client behavior - see WithRequestDelay.
type ClientOption func(*Client)

// WithRequestDelay sets the minimum gap between requests (default
// DefaultRequestDelay if not set; pass 0 to disable throttling entirely).
func WithRequestDelay(d time.Duration) ClientOption {
	return func(c *Client) { c.requestDelay = d }
}

// NewClient builds a Client authenticated with sessionCookie, the raw
// "Cookie:" header value captured from a logged-in browser session (e.g.
// via the browser's DevTools -> Application/Storage -> Cookies for
// spielerplus.de, or the request headers of any XHR on the site).
func NewClient(sessionCookie string, opts ...ClientOption) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("spielerplus: create cookie jar: %w", err)
	}
	base, err := url.Parse(BaseURL)
	if err != nil {
		return nil, fmt.Errorf("spielerplus: parse base url: %w", err)
	}
	cookies := parseCookieHeader(sessionCookie)
	if len(cookies) == 0 {
		return nil, fmt.Errorf("spielerplus: SPIELERPLUS_SESSION_COOKIE did not contain any name=value pairs - paste the full Cookie header value captured from a logged-in browser session")
	}
	jar.SetCookies(base, cookies)

	c := &Client{
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
			// SpielerPlus redirects unauthenticated requests to
			// /site/login; follow so callers see the login page and can
			// be told to re-authenticate, rather than an opaque 302.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("spielerplus: too many redirects")
				}
				return nil
			},
		},
		baseURL:      BaseURL,
		requestDelay: DefaultRequestDelay,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// throttle blocks until at least requestDelay has passed since the last
// request this Client made, so a long member/event loop can't hammer
// SpielerPlus faster than configured.
func (c *Client) throttle() {
	if c.requestDelay <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if wait := c.requestDelay - time.Since(c.lastRequest); wait > 0 {
		time.Sleep(wait)
	}
	c.lastRequest = time.Now()
}

// parseCookieHeader turns a raw "name=value; name2=value2" Cookie header
// into http.Cookie values the cookie jar can store.
func parseCookieHeader(header string) []*http.Cookie {
	var cookies []*http.Cookie
	for _, part := range strings.Split(header, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, value, found := strings.Cut(part, "=")
		if !found {
			continue
		}
		cookies = append(cookies, &http.Cookie{
			Name:  strings.TrimSpace(name),
			Value: strings.TrimSpace(value),
		})
	}
	return cookies
}

// get fetches path (relative to BaseURL) and returns the response body.
// Returns ErrNotAuthenticated if SpielerPlus served its login page instead.
func (c *Client) get(path string) (io.ReadCloser, error) {
	c.throttle()
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("spielerplus: build request for %s: %w", path, err)
	}
	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("spielerplus: GET %s: %w", path, err)
	}
	if resp.StatusCode == http.StatusForbidden {
		resp.Body.Close()
		return nil, fmt.Errorf("spielerplus: GET %s: 403 Forbidden (session cookie invalid or User-Agent blocked)", path)
	}
	if strings.Contains(resp.Request.URL.Path, "/site/login") {
		resp.Body.Close()
		return nil, ErrNotAuthenticated
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("spielerplus: GET %s: unexpected status %s", path, resp.Status)
	}
	return resp.Body, nil
}

// maxAssetBytes caps a fetched profile photo's size, matching the backend's
// own 2 MB upload limit (backend/internal/auth/handler.go) so this importer
// never uploads something the backend itself would have rejected.
const maxAssetBytes = 2 << 20

// FetchAsset downloads assetURL (an absolute URL, e.g. a member's photo on
// SpielerPlus's assets.spielerplus.de CDN - not relative to BaseURL, and
// not necessarily same-host, so this bypasses the session-scoped get/
// postForm helpers). Still throttled the same as every other request this
// client makes. Rejects a response larger than maxAssetBytes rather than
// reading an unbounded body.
func (c *Client) FetchAsset(assetURL string) ([]byte, error) {
	c.throttle()
	req, err := http.NewRequest(http.MethodGet, assetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("spielerplus: build request for asset %s: %w", assetURL, err)
	}
	req.Header.Set("User-Agent", browserUserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("spielerplus: GET asset %s: %w", assetURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spielerplus: GET asset %s: unexpected status %s", assetURL, resp.Status)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxAssetBytes+1))
	if err != nil {
		return nil, fmt.Errorf("spielerplus: read asset %s: %w", assetURL, err)
	}
	if len(data) > maxAssetBytes {
		return nil, fmt.Errorf("spielerplus: asset %s exceeds %d bytes", assetURL, maxAssetBytes)
	}
	return data, nil
}

// postForm submits an XHR-style POST to path (relative to BaseURL), matching
// the real frontend's ajax calls captured in a HAR export: form-encoded
// body, X-Requested-With: XMLHttpRequest, and the same jQuery-style Accept
// header. Used for the ajaxgetevents/ajaxgetparticipation endpoints, which
// render server-side HTML fragments rather than a full page.
func (c *Client) postForm(path string, form url.Values) (io.ReadCloser, error) {
	c.throttle()
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("spielerplus: build request for %s: %w", path, err)
	}
	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Accept", ajaxAcceptHeader)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("spielerplus: POST %s: %w", path, err)
	}
	if resp.StatusCode == http.StatusForbidden {
		resp.Body.Close()
		return nil, fmt.Errorf("spielerplus: POST %s: 403 Forbidden (session cookie invalid or User-Agent blocked)", path)
	}
	if strings.Contains(resp.Request.URL.Path, "/site/login") {
		resp.Body.Close()
		return nil, ErrNotAuthenticated
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("spielerplus: POST %s: unexpected status %s", path, resp.Status)
	}
	return resp.Body, nil
}
