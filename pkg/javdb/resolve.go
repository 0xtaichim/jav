package javdb

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) resolveCode(ctx context.Context, code string) (videoID, detailURL string, err error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", "", fmt.Errorf("code is required")
	}
	searchResult, err := c.Search(ctx, code)
	if err != nil {
		return "", "", fmt.Errorf("search failed: %w", err)
	}
	if len(searchResult.Movies) == 0 {
		return "", "", fmt.Errorf("code not found: %s", code)
	}
	detailURL = pickMovieURL(searchResult.Movies, code)
	videoID, err = videoIDFromURL(c.baseURL, detailURL)
	if err != nil {
		return "", "", err
	}
	return videoID, detailURL, nil
}

func pickMovieURL(movies []Movie, code string) string {
	for _, movie := range movies {
		if strings.EqualFold(movie.Code, code) {
			return movie.URL
		}
	}
	if len(movies) == 0 {
		return ""
	}
	return movies[0].URL
}

func videoIDFromURL(baseURL, rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	prefix := strings.TrimRight(baseURL, "/") + "/v/"
	rest, ok := strings.CutPrefix(rawURL, prefix)
	if !ok {
		rest, ok = strings.CutPrefix(rawURL, "/v/")
		if !ok {
			return "", fmt.Errorf("cannot parse video ID from detail URL: %s", rawURL)
		}
	}
	if idx := strings.IndexAny(rest, "/?"); idx >= 0 {
		rest = rest[:idx]
	}
	id := strings.TrimSpace(rest)
	if id == "" {
		return "", fmt.Errorf("cannot parse video ID from detail URL: %s", rawURL)
	}
	return id, nil
}

func (c *Client) absURL(href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	if strings.HasPrefix(href, "/") {
		return c.baseURL + href
	}
	return href
}

func normalizePage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}
