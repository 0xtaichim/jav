package javdb

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var reviewAuthorRe = regexp.MustCompile(`\S*\*+\S*`)

// GetReviews fetches review list for a code (paginated).
func (c *Client) GetReviews(code string, page int) (*ReviewResult, error) {
	searchResult, err := c.Search(code)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	if len(searchResult.Movies) == 0 {
		return nil, fmt.Errorf("code not found: %s", code)
	}

	var detailURL string
	for _, movie := range searchResult.Movies {
		if strings.EqualFold(movie.Code, code) {
			detailURL = movie.URL
			break
		}
	}
	if detailURL == "" {
		detailURL = searchResult.Movies[0].URL
	}

	prefix := c.baseURL + "/v/"
	if !strings.HasPrefix(detailURL, prefix) {
		return nil, fmt.Errorf("cannot parse video ID from detail URL: %s", detailURL)
	}
	rest := strings.TrimPrefix(detailURL, prefix)
	if idx := strings.IndexAny(rest, "/?"); idx >= 0 {
		rest = rest[:idx]
	}
	videoID := strings.TrimSpace(rest)
	if videoID == "" {
		return nil, fmt.Errorf("cannot parse video ID from detail URL: %s", detailURL)
	}

	reviewsURL := fmt.Sprintf("%s/v/%s/reviews/lastest?page=%d", c.baseURL, videoID, page)
	req, err := http.NewRequest("GET", reviewsURL, nil)
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

	result := &ReviewResult{
		Code:    code,
		Page:    page,
		Reviews: []Review{},
	}

	doc.Find(".review-items .review-item").Each(func(i int, s *goquery.Selection) {
		review := c.parseReviewItem(s)
		result.Reviews = append(result.Reviews, review)
	})

	if prevLink, exists := doc.Find("a.pagination-previous[rel='prev']").Attr("href"); exists && prevLink != "" {
		result.HasPrev = true
		if p := parsePageFromQuery(prevLink); p > 0 {
			result.PrevPage = p
		}
	}
	if nextLink, exists := doc.Find("a.pagination-next[rel='next']").Attr("href"); exists && nextLink != "" {
		result.HasNext = true
		if p := parsePageFromQuery(nextLink); p > 0 {
			result.NextPage = p
		}
	}

	return result, nil
}

// parseReviewItem parses a single review from .review-item.
func (c *Client) parseReviewItem(s *goquery.Selection) Review {
	review := Review{}

	if idStr, ok := s.Attr("id"); ok && strings.HasPrefix(idStr, "review-item-") {
		if id, err := strconv.Atoi(strings.TrimPrefix(idStr, "review-item-")); err == nil {
			review.ID = id
		}
	}

	titleText := strings.TrimSpace(s.Find(".review-title").Text())
	timeText := strings.TrimSpace(s.Find(".review-title .time").Text())
	if timeText != "" {
		titleText = strings.TrimSpace(strings.Replace(titleText, timeText, "", 1))
	}
	if author := extractReviewAuthor(titleText); author != "" {
		review.Author = author
	}

	review.Rating = s.Find(".score-stars .icon-star").Not(".gray").Length()

	likesStr := strings.TrimSpace(s.Find(".likes-count").Text())
	if likes, err := strconv.Atoi(likesStr); err == nil {
		review.Likes = likes
	}

	review.Content = strings.TrimSpace(s.Find(".content p").Text())
	review.Date = strings.TrimSpace(s.Find(".review-title .time").Text())

	return review
}

func extractReviewAuthor(titleText string) string {
	if m := reviewAuthorRe.FindString(titleText); m != "" {
		return strings.TrimSpace(m)
	}
	return ""
}

func parsePageFromQuery(href string) int {
	u, err := url.Parse(href)
	if err != nil {
		return 0
	}
	p := u.Query().Get("page")
	if p == "" {
		return 0
	}
	n, _ := strconv.Atoi(p)
	return n
}
