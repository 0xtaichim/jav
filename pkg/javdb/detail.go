package javdb

import (
	"context"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// GetDetail fetches movie detail including magnets.
func (c *Client) GetDetail(ctx context.Context, code string) (*MovieDetail, error) {
	_, detailURL, err := c.resolveCode(ctx, code)
	if err != nil {
		return nil, err
	}

	doc, err := c.getDoc(ctx, detailURL)
	if err != nil {
		return nil, err
	}

	detail := &MovieDetail{
		URL:     detailURL,
		Magnets: []MagnetLink{},
		Actors:  []Actor{},
		Tags:    []string{},
	}
	c.parseDetailPanel(doc, detail, code)
	c.parseMagnetLinks(doc, detail)
	return detail, nil
}

func (c *Client) parseDetailPanel(doc *goquery.Document, detail *MovieDetail, fallbackCode string) {
	firstBlock := doc.Find(".movie-panel-info .panel-block.first-block")
	if firstBlock.Length() > 0 {
		codeText := strings.TrimSpace(firstBlock.Find(".value").Text())
		if codeText != "" {
			detail.Code = codeText
		} else {
			detail.Code = fallbackCode
		}
	} else {
		codeText := strings.TrimSpace(doc.Find("strong.current-title").Text())
		if codeText != "" {
			detail.Code = codeText
		} else {
			detail.Code = fallbackCode
		}
	}

	titleText := strings.TrimSpace(doc.Find("h2.title strong").Text())
	if titleText == "" {
		titleText = strings.TrimSpace(doc.Find("h2.title").Text())
	}
	detail.Title = titleText

	if img, exists := doc.Find(".column-video-cover img").Attr("src"); exists {
		detail.ImageURL = img
	}

	doc.Find(".movie-panel-info .panel-block").Each(func(_ int, s *goquery.Selection) {
		value := strings.TrimSpace(s.Find(".value").Text())
		switch panelLabel(s) {
		case "番號", "番号":
			if value != "" && detail.Code == "" {
				detail.Code = value
			}
		case "日期":
			detail.Date = value
		case "時長", "时长":
			detail.Duration = value
		case "導演", "导演":
			detail.Director = value
		case "片商":
			detail.Publisher = value
		case "發行", "发行":
			if detail.Publisher == "" {
				detail.Publisher = value
			}
		case "系列":
			detail.Series = value
		case "評分", "评分":
			detail.Rating, detail.RatingCount = parseScore(s.Find(".value").Text())
		case "類別", "类别":
			s.Find(".value a").Each(func(_ int, tag *goquery.Selection) {
				tagText := strings.TrimSpace(tag.Text())
				if tagText != "" {
					detail.Tags = append(detail.Tags, tagText)
				}
			})
		case "演員", "演员":
			s.Find(".value a").Each(func(_ int, actor *goquery.Selection) {
				actorName := strings.TrimSpace(actor.Text())
				if actorName == "" {
					return
				}
				actorURL, _ := actor.Attr("href")
				detail.Actors = append(detail.Actors, Actor{
					Name: actorName,
					URL:  c.absURL(actorURL),
				})
			})
		}
	})
}

func (c *Client) parseMagnetLinks(doc *goquery.Document, detail *MovieDetail) {
	// JavDB markup evolved: magnets live under #magnets-content.magnet-links > .item
	// (older pages used .item.columns.is-desktop). Overlapping selectors are
	// expected; magnet URLs are de-duplicated below.
	sel := doc.Find("#magnets-content.magnet-links .item, .magnet-links .item.columns.is-desktop, .magnet-links .item")
	seen := map[string]bool{}
	sel.Each(func(_ int, s *goquery.Selection) {
		magnet := MagnetLink{}
		magnetNameBlock := s.Find(".magnet-name")
		if magnetLink, exists := magnetNameBlock.Find("a[href^='magnet:']").Attr("href"); exists {
			magnet.Magnet = magnetLink
		}
		if magnet.Magnet == "" {
			if clip, exists := s.Find("button.copy-to-clipboard").Attr("data-clipboard-text"); exists && strings.HasPrefix(clip, "magnet:") {
				magnet.Magnet = clip
			}
		}
		magnet.Name = strings.TrimSpace(magnetNameBlock.Find("span.name").Text())
		magnet.Size = strings.TrimSpace(magnetNameBlock.Find("span.meta").Text())
		magnetNameBlock.Find(".tags .tag").Each(func(_ int, tag *goquery.Selection) {
			tagText := strings.TrimSpace(tag.Text())
			if strings.Contains(tagText, "高清") || strings.Contains(tagText, "HD") {
				magnet.IsHD = true
			}
			if strings.Contains(tagText, "字幕") || strings.Contains(tagText, "中字") {
				magnet.HasSubs = true
			}
		})
		magnet.Date = strings.TrimSpace(s.Find(".date .time").Text())
		if magnet.Magnet == "" {
			return
		}
		if seen[magnet.Magnet] {
			return
		}
		seen[magnet.Magnet] = true
		detail.Magnets = append(detail.Magnets, magnet)
	})
}
