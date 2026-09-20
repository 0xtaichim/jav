package javdb

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/PuerkitoBio/goquery"
)

// Search searches by code or keyword.
func (c *Client) Search(query string) (*SearchResult, error) {
	searchURL := fmt.Sprintf("%s/search?q=%s", c.baseURL, url.QueryEscape(query))

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if isLoginRequired(resp) {
		return nil, &LoginRequiredError{Message: "Unauthorized"}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	doc, err := parseHTML(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	result := &SearchResult{
		Query:  query,
		Movies: []Movie{},
	}

	doc.Find("a.box[href^='/v/']").Each(func(i int, s *goquery.Selection) {
		movie := c.parseMovieItem(s)
		result.Movies = append(result.Movies, movie)
	})

	result.Total = len(result.Movies)

	return result, nil
}
