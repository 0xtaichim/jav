package javdb

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// GetDetail fetches movie detail including magnets.
func (c *Client) GetDetail(code string) (*MovieDetail, error) {
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

	req, err := http.NewRequest("GET", detailURL, nil)
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

	detail := &MovieDetail{
		URL:     detailURL,
		Magnets: []MagnetLink{},
		Actors:  []Actor{},
		Tags:    []string{},
	}

	firstBlock := doc.Find(".movie-panel-info .panel-block.first-block")
	if firstBlock.Length() > 0 {
		codeText := strings.TrimSpace(firstBlock.Find(".value").Text())
		if codeText != "" {
			detail.Code = codeText
		} else {
			detail.Code = code
		}
	} else {
		codeText := strings.TrimSpace(doc.Find("strong.current-title").Text())
		if codeText != "" {
			detail.Code = codeText
		} else {
			detail.Code = code
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

	doc.Find(".movie-panel-info .panel-block").Each(func(i int, s *goquery.Selection) {
		label := strings.TrimSpace(s.Find("strong").Text())
		label = strings.TrimSuffix(label, ":")

		if colonIndex := strings.Index(label, ":"); colonIndex != -1 {
			label = label[:colonIndex]
		}
		label = strings.TrimSpace(label)

		switch label {
		case "番號", "番号":
			codeFromPanel := strings.TrimSpace(s.Find(".value").Text())
			if codeFromPanel != "" && detail.Code == "" {
				detail.Code = codeFromPanel
			}
		case "日期":
			detail.Date = strings.TrimSpace(s.Find(".value").Text())
		case "時長", "时长":
			detail.Duration = strings.TrimSpace(s.Find(".value").Text())
		case "導演", "导演":
			detail.Director = strings.TrimSpace(s.Find(".value").Text())
		case "片商":
			detail.Publisher = strings.TrimSpace(s.Find(".value").Text())
		case "發行", "发行":
			if detail.Publisher == "" {
				detail.Publisher = strings.TrimSpace(s.Find(".value").Text())
			}
		case "系列":
			detail.Series = strings.TrimSpace(s.Find(".value").Text())
		case "評分", "评分":
			ratingText := strings.TrimSpace(s.Find(".value").Text())
			if ratingText != "" {
				parts := strings.Split(ratingText, ",")
				if len(parts) >= 1 {
					ratingStr := strings.TrimSuffix(strings.TrimSpace(parts[0]), "分")
					if rating, err := strconv.ParseFloat(ratingStr, 64); err == nil {
						detail.Rating = rating
					}
				}
				if len(parts) >= 2 {
					countStr := strings.TrimSpace(parts[1])
					countStr = strings.TrimPrefix(countStr, "由")
					countStr = strings.TrimSuffix(countStr, "人評價")
					countStr = strings.TrimSpace(countStr)
					if count, err := strconv.Atoi(countStr); err == nil {
						detail.RatingCount = count
					}
				}
			}
		case "類別", "类别":
			s.Find(".value a").Each(func(i int, tag *goquery.Selection) {
				tagText := strings.TrimSpace(tag.Text())
				if tagText != "" {
					detail.Tags = append(detail.Tags, tagText)
				}
			})
		case "演員", "演员":
			s.Find(".value a").Each(func(i int, actor *goquery.Selection) {
				actorName := strings.TrimSpace(actor.Text())
				actorURL, _ := actor.Attr("href")
				if actorURL != "" && strings.HasPrefix(actorURL, "/") {
					actorURL = c.baseURL + actorURL
				}
				if actorName != "" {
					detail.Actors = append(detail.Actors, Actor{
						Name: actorName,
						URL:  actorURL,
					})
				}
			})
		}
	})

	doc.Find(".magnet-links .item.columns.is-desktop").Each(func(i int, s *goquery.Selection) {
		magnet := MagnetLink{}

		magnetNameBlock := s.Find(".magnet-name")
		if magnetLink, exists := magnetNameBlock.Find("a[href^='magnet:']").Attr("href"); exists {
			magnet.Magnet = magnetLink
		}

		magnet.Name = strings.TrimSpace(magnetNameBlock.Find("span.name").Text())
		magnet.Size = strings.TrimSpace(magnetNameBlock.Find("span.meta").Text())

		magnetNameBlock.Find(".tags .tag").Each(func(i int, tag *goquery.Selection) {
			tagText := strings.TrimSpace(tag.Text())
			if strings.Contains(tagText, "高清") || strings.Contains(tagText, "HD") {
				magnet.IsHD = true
			}
			if strings.Contains(tagText, "字幕") || strings.Contains(tagText, "中字") {
				magnet.HasSubs = true
			}
		})

		dateText := strings.TrimSpace(s.Find(".date .time").Text())
		magnet.Date = dateText

		if magnet.Magnet != "" {
			detail.Magnets = append(detail.Magnets, magnet)
		}
	})

	return detail, nil
}
