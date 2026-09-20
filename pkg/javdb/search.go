package javdb

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Search searches by code or keyword.
func (c *Client) Search(ctx context.Context, query string) (*SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}

	searchURL := c.baseURL + "/search?q=" + url.QueryEscape(query)
	doc, err := c.getDoc(ctx, searchURL)
	if err != nil {
		return nil, err
	}

	movies := c.parseMovieList(doc)
	return &SearchResult{
		Query:  query,
		Movies: movies,
		Total:  len(movies),
	}, nil
}
