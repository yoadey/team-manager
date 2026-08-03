package spielerplus

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

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

// Client is a minimal HTTP client for the SpielerPlus pages this importer
// needs. It is read-only against SpielerPlus: it never submits the
// participation form or otherwise mutates SpielerPlus data.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient builds a Client authenticated with sessionCookie, the raw
// "Cookie:" header value captured from a logged-in browser session (e.g.
// via the browser's DevTools -> Application/Storage -> Cookies for
// spielerplus.de, or the request headers of any XHR on the site).
func NewClient(sessionCookie string) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("spielerplus: create cookie jar: %w", err)
	}
	base, err := url.Parse(BaseURL)
	if err != nil {
		return nil, fmt.Errorf("spielerplus: parse base url: %w", err)
	}
	jar.SetCookies(base, parseCookieHeader(sessionCookie))

	return &Client{
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
		baseURL: BaseURL,
	}, nil
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
