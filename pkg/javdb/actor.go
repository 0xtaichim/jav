package javdb

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// parseActorItem parses a single actor from collection page.
func (c *Client) parseActorItem(s *goquery.Selection) Actor {
	actor := Actor{}
	actor.Name = strings.TrimSpace(s.Find("strong").Text())
	if href, exists := s.Attr("href"); exists {
		if strings.HasPrefix(href, "/") {
			actor.URL = c.baseURL + href
		} else {
			actor.URL = href
		}
	}
	if img, exists := s.Find("img.avatar").Attr("src"); exists {
		actor.ImageURL = img
	}
	return actor
}
