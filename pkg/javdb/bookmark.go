package javdb

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// GetBookmarks fetches bookmarks (want_watch, watched, or actors). Requires login cookie.
func (c *Client) GetBookmarks(bookmarkType string, page int) (*BookmarkResult, error) {
	var path string
	switch bookmarkType {
	case "want_watch":
		path = "/users/want_watch_videos"
	case "watched":
		path = "/users/watched_videos"
	case "actors":
		path = "/users/collection_actors"
	default:
		return nil, fmt.Errorf("unknown bookmark type %q; supported: want_watch, watched, actors", bookmarkType)
	}

	bookmarkURL := c.baseURL + path
	if page > 1 {
		bookmarkURL = fmt.Sprintf("%s?page=%d", bookmarkURL, page)
	}

	req, err := http.NewRequest("GET", bookmarkURL, nil)
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
		return nil, fmt.Errorf("HTTP status %d (bookmarks require login; set JAVDB_COOKIES)", resp.StatusCode)
	}

	doc, err := parseHTML(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	result := &BookmarkResult{
		Type:   bookmarkType,
		Movies: []Movie{},
		Actors: []Actor{},
		Page:   page,
	}

	if bookmarkType == "actors" {
		doc.Find("#actors .actor-box a[href^='/actors/']").Each(func(i int, s *goquery.Selection) {
			href, _ := s.Attr("href")
			if strings.Contains(href, "uncollect") {
				return
			}
			actor := c.parseActorItem(s)
			result.Actors = append(result.Actors, actor)
		})
		result.Total = len(result.Actors)
	} else {
		doc.Find("#videos .item").Each(func(i int, item *goquery.Selection) {
			boxLink := item.Find(".box > a[href^='/v/']").First()
			if boxLink.Length() == 0 {
				return
			}
			movie := c.parseMovieItem(boxLink)
			item.Find("a[href*='/reviews/']").Each(func(_ int, a *goquery.Selection) {
				href, _ := a.Attr("href")
				if href == "" || strings.Contains(href, "reviews/lastest") {
					return
				}
				if idx := strings.Index(href, "/reviews/"); idx >= 0 {
					rest := href[idx+len("/reviews/"):]
					if end := strings.IndexAny(rest, "/?"); end >= 0 {
						rest = rest[:end]
					}
					if id, err := strconv.Atoi(strings.TrimSpace(rest)); err == nil && id > 0 {
						movie.ReviewID = id
					}
				}
			})
			result.Movies = append(result.Movies, movie)
		})
		result.Total = len(result.Movies)
	}

	nextSel := doc.Find("a.pagination-next[rel='next']")
	if nextSel.Length() > 0 {
		if nextHref, exists := nextSel.Attr("href"); exists && nextHref != "" {
			result.HasNext = true
			if p := parsePageFromQuery(nextHref); p > 0 {
				result.NextPage = p
			}
		}
	}

	return result, nil
}

// AddWantWatch adds "want to watch" for a code.
func (c *Client) AddWantWatch(code string) error {
	videoID, detailURL, err := c.getVideoIDFromCode(code)
	if err != nil {
		return err
	}
	csrfToken, err := c.getCSRFToken(detailURL)
	if err != nil {
		return err
	}
	postURL := fmt.Sprintf("%s/v/%s/reviews/want_to_watch", c.baseURL, videoID)
	form := url.Values{}
	form.Set("authenticity_token", csrfToken)
	body := form.Encode()
	req, err := http.NewRequest("POST", postURL, bytes.NewReader([]byte(body)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	c.setAJAXHeaders(req, csrfToken, detailURL)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.ContentLength = int64(len(body))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("add want-watch failed: HTTP %d %s", resp.StatusCode, string(msg))
	}
	return nil
}

// findWantWatchReviewID finds videoID and review_id for a code in want_watch list.
func (c *Client) findWantWatchReviewID(code string) (videoID string, reviewID int, err error) {
	page := 1
	for {
		result, err := c.GetBookmarks("want_watch", page)
		if err != nil {
			return "", 0, err
		}
		for _, m := range result.Movies {
			if strings.EqualFold(m.Code, code) {
				prefix := c.baseURL + "/v/"
				if !strings.HasPrefix(m.URL, prefix) {
					return "", 0, fmt.Errorf("cannot parse video ID from bookmark URL: %s", m.URL)
				}
				rest := strings.TrimPrefix(m.URL, prefix)
				if idx := strings.IndexAny(rest, "/?"); idx >= 0 {
					rest = rest[:idx]
				}
				videoID = strings.TrimSpace(rest)
				if m.ReviewID <= 0 {
					return "", 0, fmt.Errorf("review_id not found for this want-watch item; try again later")
				}
				return videoID, m.ReviewID, nil
			}
		}
		if !result.HasNext {
			return "", 0, fmt.Errorf("code %s not found in want-watch list", code)
		}
		page = result.NextPage
		if page <= 0 {
			break
		}
	}
	return "", 0, fmt.Errorf("code %s not found in want-watch list", code)
}

// RemoveWantWatch removes "want to watch" for a code.
func (c *Client) RemoveWantWatch(code string) error {
	videoID, reviewID, err := c.findWantWatchReviewID(code)
	if err != nil {
		return err
	}
	detailURL := fmt.Sprintf("%s/v/%s", c.baseURL, videoID)
	csrfToken, err := c.getCSRFToken(detailURL)
	if err != nil {
		return err
	}
	deleteURL := fmt.Sprintf("%s/v/%s/reviews/%d", c.baseURL, videoID, reviewID)
	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	c.setAJAXHeaders(req, csrfToken, detailURL)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("remove want-watch failed: HTTP %d %s", resp.StatusCode, string(msg))
	}
	return nil
}

// AddWatched adds "watched" for a code (optional rating and content).
func (c *Client) AddWatched(code string, rating int, content string) error {
	videoID, detailURL, err := c.getVideoIDFromCode(code)
	if err != nil {
		return err
	}
	csrfToken, err := c.getCSRFToken(detailURL)
	if err != nil {
		return err
	}
	postURL := fmt.Sprintf("%s/v/%s/reviews", c.baseURL, videoID)
	form := url.Values{}
	form.Set("authenticity_token", csrfToken)
	if rating >= 1 && rating <= 5 {
		form.Set("review[rating]", strconv.Itoa(rating))
	} else {
		form.Set("review[rating]", "3")
	}
	form.Set("review[content]", content)
	form.Set("video_review[status]", "watched")
	form.Set("commit", "保存")
	body := form.Encode()
	req, err := http.NewRequest("POST", postURL, bytes.NewReader([]byte(body)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	c.setAJAXHeaders(req, csrfToken, detailURL)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.ContentLength = int64(len(body))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("add watched failed: HTTP %d %s", resp.StatusCode, string(msg))
	}
	return nil
}

// findWatchedReviewID finds videoID and review_id for a code in watched list.
func (c *Client) findWatchedReviewID(code string) (videoID string, reviewID int, err error) {
	page := 1
	for {
		result, err := c.GetBookmarks("watched", page)
		if err != nil {
			return "", 0, err
		}
		for _, m := range result.Movies {
			if strings.EqualFold(m.Code, code) {
				prefix := c.baseURL + "/v/"
				if !strings.HasPrefix(m.URL, prefix) {
					return "", 0, fmt.Errorf("cannot parse video ID from bookmark URL: %s", m.URL)
				}
				rest := strings.TrimPrefix(m.URL, prefix)
				if idx := strings.IndexAny(rest, "/?"); idx >= 0 {
					rest = rest[:idx]
				}
				videoID = strings.TrimSpace(rest)
				if m.ReviewID <= 0 {
					return "", 0, fmt.Errorf("review_id not found for this watched item; try again later")
				}
				return videoID, m.ReviewID, nil
			}
		}
		if !result.HasNext {
			return "", 0, fmt.Errorf("code %s not found in watched list", code)
		}
		page = result.NextPage
		if page <= 0 {
			break
		}
	}
	return "", 0, fmt.Errorf("code %s not found in watched list", code)
}

// RemoveWatched removes "watched" for a code.
func (c *Client) RemoveWatched(code string) error {
	videoID, reviewID, err := c.findWatchedReviewID(code)
	if err != nil {
		return err
	}
	detailURL := fmt.Sprintf("%s/v/%s", c.baseURL, videoID)
	csrfToken, err := c.getCSRFToken(detailURL)
	if err != nil {
		return err
	}
	deleteURL := fmt.Sprintf("%s/v/%s/reviews/%d", c.baseURL, videoID, reviewID)
	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	c.setAJAXHeaders(req, csrfToken, detailURL)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("remove watched failed: HTTP %d %s", resp.StatusCode, string(msg))
	}
	return nil
}
