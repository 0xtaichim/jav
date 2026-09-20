package javdb

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var reviewAuthorRe = regexp.MustCompile(`\S*\*+\S*`)

func (c *Client) parseMovieList(doc *goquery.Document) []Movie {
	movies := []Movie{}
	doc.Find("a.box[href^='/v/']").Each(func(_ int, s *goquery.Selection) {
		movies = append(movies, c.parseMovieItem(s))
	})
	return movies
}

func (c *Client) parseMovieItem(s *goquery.Selection) Movie {
	movie := Movie{}
	movie.Code = strings.TrimSpace(s.Find("strong").Text())

	titleText := s.Find(".video-title").Text()
	titleText = strings.Replace(titleText, movie.Code, "", 1)
	movie.Title = strings.TrimSpace(titleText)

	if href, exists := s.Attr("href"); exists {
		movie.URL = c.absURL(href)
	}

	movie.Rating, movie.RatingCount = parseScore(s.Find(".score .value").Text())
	movie.Date = strings.TrimSpace(s.Find(".meta").Text())

	s.Find(".tags .tag").Each(func(_ int, tag *goquery.Selection) {
		tagText := strings.TrimSpace(tag.Text())
		if tagText == "" {
			return
		}
		if strings.Contains(tagText, "磁鏈") || strings.Contains(tagText, "磁链") {
			movie.HasMagnet = true
		}
		movie.Tags = append(movie.Tags, tagText)
	})

	if img, exists := s.Find("img").Attr("src"); exists {
		movie.ImageURL = img
	}
	return movie
}

func (c *Client) parseActorItem(s *goquery.Selection) Actor {
	actor := Actor{
		Name: strings.TrimSpace(s.Find("strong").Text()),
	}
	if href, exists := s.Attr("href"); exists {
		actor.URL = c.absURL(href)
	}
	if img, exists := s.Find("img.avatar").Attr("src"); exists {
		actor.ImageURL = img
	}
	return actor
}

func parseReviewItem(s *goquery.Selection) Review {
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
	review.Author = extractReviewAuthor(titleText)
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

func parseScore(text string) (rating float64, count int) {
	text = compactRatingText(text)
	if text == "" {
		return 0, 0
	}
	ratingPart, countPart, found := strings.Cut(text, ",")
	ratingPart = strings.TrimSuffix(ratingPart, "分")
	if v, err := strconv.ParseFloat(ratingPart, 64); err == nil {
		rating = v
	}
	if !found {
		return rating, 0
	}
	countPart = strings.TrimPrefix(countPart, "由")
	countPart = strings.TrimSuffix(countPart, "人評價")
	countPart = strings.TrimSuffix(countPart, "人评价")
	countPart = strings.ReplaceAll(countPart, ",", "")
	if n, err := strconv.Atoi(countPart); err == nil {
		count = n
	}
	return rating, count
}

func compactRatingText(s string) string {
	r := strings.NewReplacer(
		"\u00a0", "",
		"&nbsp;", "",
		" ", "",
		"\n", "",
		"\t", "",
	)
	return r.Replace(s)
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
	n, err := strconv.Atoi(p)
	if err != nil {
		return 0
	}
	return n
}

func parsePagination(doc *goquery.Document) (hasPrev bool, prevPage int, hasNext bool, nextPage int) {
	if prevLink, exists := doc.Find("a.pagination-previous[rel='prev']").Attr("href"); exists && prevLink != "" {
		hasPrev = true
		if p := parsePageFromQuery(prevLink); p > 0 {
			prevPage = p
		}
	}
	if nextLink, exists := doc.Find("a.pagination-next[rel='next']").Attr("href"); exists && nextLink != "" {
		hasNext = true
		if p := parsePageFromQuery(nextLink); p > 0 {
			nextPage = p
		}
	}
	return hasPrev, prevPage, hasNext, nextPage
}

func panelLabel(s *goquery.Selection) string {
	label := strings.TrimSpace(s.Find("strong").Text())
	label = strings.TrimSuffix(label, ":")
	if colonIndex := strings.Index(label, ":"); colonIndex != -1 {
		label = label[:colonIndex]
	}
	return strings.TrimSpace(label)
}
