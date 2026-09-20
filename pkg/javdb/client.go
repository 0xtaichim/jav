package javdb

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	baseURL   = "https://javdb.com"
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
)

// Client is the JavDB API client.
type Client struct {
	httpClient *http.Client
	baseURL    string
	cookies    string
}

// injectLocaleIntoCookies ensures the cookie string contains locale from JAVDB_LOCALE.
// Replaces existing locale=... or appends "; locale=<locale>" if absent.
func injectLocaleIntoCookies(cookies, locale string) string {
	re := regexp.MustCompile(`locale=[^;]*`)
	if re.MatchString(cookies) {
		return re.ReplaceAllString(cookies, "locale="+locale)
	}
	return strings.TrimRight(cookies, "; ") + "; locale=" + locale
}

// NewClient creates a new JavDB client.
func NewClient() *Client {
	tr := newTransport()

	locale := os.Getenv("JAVDB_LOCALE")
	if locale == "" {
		locale = "zh"
	}
	cookies := os.Getenv("JAVDB_COOKIES")
	if cookies == "" {
		cookies = "over18=1; theme=auto; locale=" + locale
	} else {
		cookies = injectLocaleIntoCookies(cookies, locale)
	}

	return &Client{
		httpClient: &http.Client{
			Transport: tr,
			Timeout:   30 * time.Second,
		},
		baseURL: baseURL,
		cookies: cookies,
	}
}

// isLoginRequired returns true if the response indicates login is required:
// 401 Unauthorized, or 200 OK with the login page (JavDB redirects to /users/sign_in).
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

// setHeaders sets request headers.
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Referer", c.baseURL)
	req.Header.Set("Cookie", c.cookies)

	// Add Sec-Ch-Ua headers to match Chrome 131 fingerprint
	req.Header.Set("Sec-Ch-Ua", `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)
}

// setAJAXHeaders sets headers for POST/DELETE AJAX (X-CSRF-Token, X-Requested-With).
func (c *Client) setAJAXHeaders(req *http.Request, csrfToken string, referer string) {
	c.setHeaders(req)
	req.Header.Set("Accept", "text/javascript, application/javascript, application/ecmascript, application/x-ecmascript, */*; q=0.01")
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
}

// getVideoIDFromCode resolves code to videoID and detail URL.
func (c *Client) getVideoIDFromCode(code string) (videoID string, detailURL string, err error) {
	searchResult, err := c.Search(code)
	if err != nil {
		return "", "", fmt.Errorf("search failed: %w", err)
	}
	if len(searchResult.Movies) == 0 {
		return "", "", fmt.Errorf("code not found: %s", code)
	}
	var targetURL string
	for _, movie := range searchResult.Movies {
		if strings.EqualFold(movie.Code, code) {
			targetURL = movie.URL
			break
		}
	}
	if targetURL == "" {
		targetURL = searchResult.Movies[0].URL
	}
	prefix := c.baseURL + "/v/"
	if !strings.HasPrefix(targetURL, prefix) {
		return "", "", fmt.Errorf("cannot parse video ID from detail URL: %s", targetURL)
	}
	rest := strings.TrimPrefix(targetURL, prefix)
	if idx := strings.IndexAny(rest, "/?"); idx >= 0 {
		rest = rest[:idx]
	}
	videoID = strings.TrimSpace(rest)
	if videoID == "" {
		return "", "", fmt.Errorf("cannot parse video ID from detail URL: %s", targetURL)
	}
	return videoID, targetURL, nil
}

// getCSRFToken fetches csrf-token from page meta.
func (c *Client) getCSRFToken(pageURL string) (string, error) {
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	c.setHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if isLoginRequired(resp) {
		return "", &LoginRequiredError{Message: "Unauthorized"}
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP status %d", resp.StatusCode)
	}
	doc, err := parseHTML(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}
	var token string
	doc.Find("meta[name=\"csrf-token\"]").Each(func(i int, s *goquery.Selection) {
		if t, ok := s.Attr("content"); ok && t != "" {
			token = t
		}
	})
	if token == "" {
		return "", &LoginRequiredError{Message: "Unauthorized"}
	}
	return token, nil
}
