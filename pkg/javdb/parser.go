package javdb

import (
	"io"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// parseHTML is a helper function to create a goquery document from an io.Reader.
func parseHTML(r io.Reader) (*goquery.Document, error) {
	return goquery.NewDocumentFromReader(r)
}

// parseMovieItem parses a movie entry from list HTML.
func (c *Client) parseMovieItem(s *goquery.Selection) Movie {
	movie := Movie{}

	movie.Code = strings.TrimSpace(s.Find("strong").Text())

	titleText := s.Find(".video-title").Text()
	titleText = strings.Replace(titleText, movie.Code, "", 1)
	movie.Title = strings.TrimSpace(titleText)

	if href, exists := s.Attr("href"); exists {
		if strings.HasPrefix(href, "/") {
			movie.URL = c.baseURL + href
		} else {
			movie.URL = href
		}
	}

	ratingText := s.Find(".score .value").Text()
	ratingText = strings.TrimSpace(ratingText)
	if ratingText != "" {
		ratingText = strings.ReplaceAll(ratingText, " ", "")
		ratingText = strings.ReplaceAll(ratingText, "\n", "")
		ratingText = strings.ReplaceAll(ratingText, "\t", "")
		ratingText = strings.ReplaceAll(ratingText, "&nbsp;", "")

		parts := strings.Split(ratingText, ",")
		if len(parts) >= 1 {
			ratingStr := strings.TrimSuffix(strings.TrimSpace(parts[0]), "分")
			if rating, err := strconv.ParseFloat(ratingStr, 64); err == nil {
				movie.Rating = rating
			}
		}
		if len(parts) >= 2 {
			countStr := strings.TrimSpace(parts[1])
			countStr = strings.TrimPrefix(countStr, "由")
			countStr = strings.TrimSuffix(countStr, "人評價")
			countStr = strings.TrimSpace(countStr)
			if count, err := strconv.Atoi(countStr); err == nil {
				movie.RatingCount = count
			}
		}
	}

	movie.Date = strings.TrimSpace(s.Find(".meta").Text())

	s.Find(".tags .tag").Each(func(i int, tag *goquery.Selection) {
		tagText := strings.TrimSpace(tag.Text())
		if strings.Contains(tagText, "磁鏈") || strings.Contains(tagText, "磁链") {
			movie.HasMagnet = true
		}
		if tagText != "" {
			movie.Tags = append(movie.Tags, tagText)
		}
	})

	if img, exists := s.Find("img").Attr("src"); exists {
		movie.ImageURL = img
	}

	return movie
}
