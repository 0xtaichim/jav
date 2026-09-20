package javdb

import (
	"fmt"
	"net/http"

	"github.com/PuerkitoBio/goquery"
)

// GetRankings fetches ranking list.
func (c *Client) GetRankings(period, rankingType string) (*RankingResult, error) {
	rankingURL := fmt.Sprintf("%s/rankings/movies?p=%s&t=%s", c.baseURL, period, rankingType)

	req, err := http.NewRequest("GET", rankingURL, nil)
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

	result := &RankingResult{
		Period: period,
		Type:   rankingType,
		Movies: []Movie{},
	}

	doc.Find("a.box[href^='/v/']").Each(func(i int, s *goquery.Selection) {
		movie := c.parseMovieItem(s)
		result.Movies = append(result.Movies, movie)
	})

	return result, nil
}
