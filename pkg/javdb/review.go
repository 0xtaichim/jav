package javdb

import (
	"context"
	"fmt"

	"github.com/PuerkitoBio/goquery"
)

// GetReviews fetches review list for a code (paginated).
func (c *Client) GetReviews(ctx context.Context, code string, page int) (*ReviewResult, error) {
	videoID, _, err := c.resolveCode(ctx, code)
	if err != nil {
		return nil, err
	}

	page = normalizePage(page)
	reviewsURL := fmt.Sprintf("%s/v/%s/reviews/lastest?page=%d", c.baseURL, videoID, page)
	doc, err := c.getDoc(ctx, reviewsURL)
	if err != nil {
		return nil, err
	}

	result := &ReviewResult{
		Code:    code,
		Page:    page,
		Reviews: []Review{},
	}
	doc.Find(".review-items .review-item").Each(func(_ int, s *goquery.Selection) {
		result.Reviews = append(result.Reviews, parseReviewItem(s))
	})
	result.HasPrev, result.PrevPage, result.HasNext, result.NextPage = parsePagination(doc)
	return result, nil
}
