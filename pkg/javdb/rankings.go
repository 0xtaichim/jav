package javdb

import (
	"context"
	"fmt"
	"strings"
)

var (
	validRankingPeriods = map[string]struct{}{
		"daily":   {},
		"weekly":  {},
		"monthly": {},
	}
	validRankingTypes = map[string]struct{}{
		"censored":   {},
		"uncensored": {},
		"western":    {},
		"fc2":        {},
	}
)

// GetRankings fetches ranking list.
func (c *Client) GetRankings(ctx context.Context, period, rankingType string) (*RankingResult, error) {
	period, rankingType, err := normalizeRanking(period, rankingType)
	if err != nil {
		return nil, err
	}

	rankingURL := fmt.Sprintf("%s/rankings/movies?p=%s&t=%s", c.baseURL, period, rankingType)
	doc, err := c.getDoc(ctx, rankingURL)
	if err != nil {
		return nil, err
	}

	return &RankingResult{
		Period: period,
		Type:   rankingType,
		Movies: c.parseMovieList(doc),
	}, nil
}

func normalizeRanking(period, rankingType string) (string, string, error) {
	period = strings.ToLower(strings.TrimSpace(period))
	rankingType = strings.ToLower(strings.TrimSpace(rankingType))
	if period == "" {
		period = "daily"
	}
	if rankingType == "" {
		rankingType = "censored"
	}
	if _, ok := validRankingPeriods[period]; !ok {
		return "", "", fmt.Errorf("unknown ranking period %q; supported: daily, weekly, monthly", period)
	}
	if _, ok := validRankingTypes[rankingType]; !ok {
		return "", "", fmt.Errorf("unknown ranking type %q; supported: censored, uncensored, western, fc2", rankingType)
	}
	return period, rankingType, nil
}
