package javdb

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const maxBookmarkPages = 100

var bookmarkPaths = map[string]string{
	"want_watch": "/users/want_watch_videos",
	"watched":    "/users/watched_videos",
	"actors":     "/users/collection_actors",
}

// GetBookmarks fetches bookmarks (want_watch, watched, or actors). Requires login cookie.
func (c *Client) GetBookmarks(ctx context.Context, bookmarkType string, page int) (*BookmarkResult, error) {
	path, ok := bookmarkPaths[bookmarkType]
	if !ok {
		return nil, fmt.Errorf("unknown bookmark type %q; supported: want_watch, watched, actors", bookmarkType)
	}

	page = normalizePage(page)
	bookmarkURL := c.baseURL + path
	if page > 1 {
		bookmarkURL = fmt.Sprintf("%s?page=%d", bookmarkURL, page)
	}

	doc, err := c.getDoc(ctx, bookmarkURL)
	if err != nil {
		var he *HTTPError
		if errors.As(err, &he) {
			return nil, fmt.Errorf("%w (bookmarks require login; set JAVDB_COOKIES)", err)
		}
		return nil, err
	}

	result := &BookmarkResult{
		Type:   bookmarkType,
		Movies: []Movie{},
		Actors: []Actor{},
		Page:   page,
	}

	if bookmarkType == "actors" {
		doc.Find("#actors .actor-box a[href^='/actors/']").Each(func(_ int, s *goquery.Selection) {
			href, _ := s.Attr("href")
			if strings.Contains(href, "uncollect") {
				return
			}
			result.Actors = append(result.Actors, c.parseActorItem(s))
		})
		result.Total = len(result.Actors)
	} else {
		doc.Find("#videos .item").Each(func(_ int, item *goquery.Selection) {
			boxLink := item.Find(".box > a[href^='/v/']").First()
			if boxLink.Length() == 0 {
				return
			}
			movie := c.parseMovieItem(boxLink)
			item.Find("a[href*='/reviews/']").Each(func(_ int, a *goquery.Selection) {
				href, _ := a.Attr("href")
				if id := reviewIDFromHref(href); id > 0 {
					movie.ReviewID = id
				}
			})
			result.Movies = append(result.Movies, movie)
		})
		result.Total = len(result.Movies)
	}

	_, _, result.HasNext, result.NextPage = parsePagination(doc)
	return result, nil
}

// AddWantWatch adds "want to watch" for a code.
func (c *Client) AddWantWatch(ctx context.Context, code string) error {
	videoID, detailURL, csrfToken, err := c.prepareMutation(ctx, code)
	if err != nil {
		return err
	}
	postURL := fmt.Sprintf("%s/v/%s/reviews/want_to_watch", c.baseURL, videoID)
	form := url.Values{}
	form.Set("authenticity_token", csrfToken)
	if err := c.postForm(ctx, postURL, detailURL, csrfToken, form); err != nil {
		return fmt.Errorf("add want-watch failed: %w", err)
	}
	return nil
}

// RemoveWantWatch removes "want to watch" for a code.
func (c *Client) RemoveWantWatch(ctx context.Context, code string) error {
	return c.removeBookmark(ctx, "want_watch", code, "want-watch")
}

// AddWatched adds "watched" for a code (optional rating and content).
func (c *Client) AddWatched(ctx context.Context, code string, rating int, content string) error {
	videoID, detailURL, csrfToken, err := c.prepareMutation(ctx, code)
	if err != nil {
		return err
	}
	if rating < 1 || rating > 5 {
		rating = 3
	}
	postURL := fmt.Sprintf("%s/v/%s/reviews", c.baseURL, videoID)
	form := url.Values{}
	form.Set("authenticity_token", csrfToken)
	form.Set("video_review[score]", strconv.Itoa(rating))
	form.Set("video_review[content]", content)
	form.Set("video_review[status]", "watched")
	form.Set("commit", "保存")
	if err := c.postForm(ctx, postURL, detailURL, csrfToken, form); err != nil {
		return fmt.Errorf("add watched failed: %w", err)
	}
	return nil
}

// RemoveWatched removes "watched" for a code.
func (c *Client) RemoveWatched(ctx context.Context, code string) error {
	return c.removeBookmark(ctx, "watched", code, "watched")
}

func (c *Client) prepareMutation(ctx context.Context, code string) (videoID, detailURL, csrfToken string, err error) {
	videoID, detailURL, err = c.resolveCode(ctx, code)
	if err != nil {
		return "", "", "", err
	}
	csrfToken, err = c.getCSRFToken(ctx, detailURL)
	if err != nil {
		return "", "", "", err
	}
	return videoID, detailURL, csrfToken, nil
}

func (c *Client) removeBookmark(ctx context.Context, bookmarkType, code, label string) error {
	videoID, reviewID, err := c.findBookmarkReviewID(ctx, bookmarkType, code)
	if err != nil {
		return err
	}
	detailURL := fmt.Sprintf("%s/v/%s", c.baseURL, videoID)
	csrfToken, err := c.getCSRFToken(ctx, detailURL)
	if err != nil {
		return err
	}
	deleteURL := fmt.Sprintf("%s/v/%s/reviews/%d", c.baseURL, videoID, reviewID)
	if err := c.deleteAJAX(ctx, deleteURL, detailURL, csrfToken); err != nil {
		return fmt.Errorf("remove %s failed: %w", label, err)
	}
	return nil
}

func (c *Client) findBookmarkReviewID(ctx context.Context, bookmarkType, code string) (videoID string, reviewID int, err error) {
	page := 1
	for i := 0; i < maxBookmarkPages; i++ {
		result, err := c.GetBookmarks(ctx, bookmarkType, page)
		if err != nil {
			return "", 0, err
		}
		for _, m := range result.Movies {
			if !strings.EqualFold(m.Code, code) {
				continue
			}
			videoID, err = videoIDFromURL(c.baseURL, m.URL)
			if err != nil {
				return "", 0, fmt.Errorf("cannot parse video ID from bookmark URL: %s", m.URL)
			}
			if m.ReviewID <= 0 {
				return "", 0, fmt.Errorf("review_id not found for this %s item; try again later", bookmarkType)
			}
			return videoID, m.ReviewID, nil
		}
		if !result.HasNext || result.NextPage <= page {
			break
		}
		page = result.NextPage
	}
	return "", 0, fmt.Errorf("code %s not found in %s list", code, strings.ReplaceAll(bookmarkType, "_", "-"))
}

func reviewIDFromHref(href string) int {
	if href == "" || strings.Contains(href, "reviews/lastest") {
		return 0
	}
	idx := strings.Index(href, "/reviews/")
	if idx < 0 {
		return 0
	}
	rest := href[idx+len("/reviews/"):]
	if end := strings.IndexAny(rest, "/?"); end >= 0 {
		rest = rest[:end]
	}
	id, err := strconv.Atoi(strings.TrimSpace(rest))
	if err != nil || id <= 0 {
		return 0
	}
	return id
}
