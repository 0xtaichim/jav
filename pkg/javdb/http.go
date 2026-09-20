package javdb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const maxErrorBody = 4 << 10

func (c *Client) getDoc(ctx context.Context, rawURL string) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.setHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer drainAndClose(resp.Body)

	if err := checkResponse(resp, false); err != nil {
		return nil, err
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	return doc, nil
}

func (c *Client) postForm(ctx context.Context, rawURL, referer, csrfToken string, form url.Values) error {
	body := []byte(form.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	req.ContentLength = int64(len(body))
	c.setAJAXHeaders(req, csrfToken, referer)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	return c.doExpectOK(req)
}

func (c *Client) deleteAJAX(ctx context.Context, rawURL, referer, csrfToken string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, rawURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	c.setAJAXHeaders(req, csrfToken, referer)
	return c.doExpectOK(req)
}

func (c *Client) doExpectOK(req *http.Request) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer drainAndClose(resp.Body)
	return checkResponse(resp, true)
}

func (c *Client) getCSRFToken(ctx context.Context, pageURL string) (string, error) {
	doc, err := c.getDoc(ctx, pageURL)
	if err != nil {
		return "", err
	}
	token, ok := doc.Find(`meta[name="csrf-token"]`).Attr("content")
	token = strings.TrimSpace(token)
	if !ok || token == "" {
		return "", &LoginRequiredError{Message: "Unauthorized"}
	}
	return token, nil
}

func checkResponse(resp *http.Response, includeBody bool) error {
	if isLoginRequired(resp) {
		return &LoginRequiredError{Message: "Unauthorized"}
	}
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	he := &HTTPError{StatusCode: resp.StatusCode}
	if includeBody && resp.Body != nil {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		he.Body = strings.TrimSpace(string(msg))
	}
	return he
}

func drainAndClose(body io.ReadCloser) {
	if body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(body, 8<<10))
	_ = body.Close()
}
