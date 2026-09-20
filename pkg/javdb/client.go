package javdb

import (
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	baseURL   = "https://javdb.com"
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
)

var localeCookieRe = regexp.MustCompile(`(?i)locale=[^;]*`)

// Client is the JavDB API client.
type Client struct {
	httpClient  *http.Client
	baseURL     string
	cookies     string
	locale      string
	proxy       string
	timeout     time.Duration
	tlsInsecure bool
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient injects a custom HTTP client (tests, custom transports).
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// WithBaseURL overrides the JavDB origin. Trailing slashes are stripped.
func WithBaseURL(u string) Option {
	return func(c *Client) {
		if u = strings.TrimRight(strings.TrimSpace(u), "/"); u != "" {
			c.baseURL = u
		}
	}
}

// WithCookies sets the Cookie header (locale is injected separately).
func WithCookies(cookies string) Option {
	return func(c *Client) {
		c.cookies = cookies
	}
}

// WithLocale sets the locale cookie (default zh).
func WithLocale(locale string) Option {
	return func(c *Client) {
		c.locale = locale
	}
}

// WithProxy sets a SOCKS5 proxy address (host:port or socks5://host:port).
func WithProxy(addr string) Option {
	return func(c *Client) {
		c.proxy = addr
	}
}

// WithTimeout sets the HTTP client timeout. Ignored when WithHTTPClient is used.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.timeout = d
		}
	}
}

// WithTLSInsecure skips TLS certificate verification when using the built-in transport.
func WithTLSInsecure(insecure bool) Option {
	return func(c *Client) {
		c.tlsInsecure = insecure
	}
}

// New creates a JavDB client. Zero options yield a client with default headers,
// locale "zh", and a Chrome-fingerprint HTTP transport.
func New(opts ...Option) *Client {
	c := &Client{
		baseURL: baseURL,
		locale:  "zh",
		timeout: 30 * time.Second,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
	if c.locale == "" {
		c.locale = "zh"
	}
	c.cookies = buildCookies(c.cookies, c.locale)
	if c.httpClient == nil {
		c.httpClient = &http.Client{
			Transport: newTransport(c.proxy, c.tlsInsecure),
			Timeout:   c.timeout,
		}
	}
	return c
}

// NewClient creates a client from process environment variables.
// Prefer New with explicit options when wiring from config or tests.
func NewClient() *Client {
	return New(
		WithCookies(os.Getenv("JAVDB_COOKIES")),
		WithLocale(os.Getenv("JAVDB_LOCALE")),
		WithProxy(os.Getenv("SOCKS5_PROXY")),
		WithBaseURL(os.Getenv("JAVDB_BASE_URL")),
		WithTLSInsecure(envTruthy(os.Getenv("JAVDB_TLS_INSECURE"))),
	)
}

func envTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func buildCookies(cookies, locale string) string {
	if locale == "" {
		locale = "zh"
	}
	if strings.TrimSpace(cookies) == "" {
		return "over18=1; theme=auto; locale=" + locale
	}
	return injectLocaleIntoCookies(cookies, locale)
}

func injectLocaleIntoCookies(cookies, locale string) string {
	if localeCookieRe.MatchString(cookies) {
		return localeCookieRe.ReplaceAllString(cookies, "locale="+locale)
	}
	return strings.TrimRight(cookies, "; ") + "; locale=" + locale
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Referer", c.baseURL+"/")
	req.Header.Set("Cookie", c.cookies)
	req.Header.Set("Sec-Ch-Ua", `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)
}

func (c *Client) setAJAXHeaders(req *http.Request, csrfToken, referer string) {
	c.setHeaders(req)
	req.Header.Set("Accept", "text/javascript, application/javascript, application/ecmascript, application/x-ecmascript, */*; q=0.01")
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
}

func isLoginRequired(resp *http.Response) bool {
	if resp == nil {
		return false
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return true
	}
	if resp.StatusCode != http.StatusOK || resp.Request == nil || resp.Request.URL == nil {
		return false
	}
	p := resp.Request.URL.Path
	return strings.Contains(p, "/users/sign_in") || strings.Contains(p, "/login")
}
