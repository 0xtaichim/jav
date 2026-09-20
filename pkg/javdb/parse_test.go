package javdb

import (
	"errors"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestParseScore(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in         string
		wantRating float64
		wantCount  int
	}{
		{"8.5分, 由123人評價", 8.5, 123},
		{"9.0分, 由1,234人评价", 9.0, 1234},
		{"7分", 7, 0},
		{"", 0, 0},
		{"  6.2 分 , 由 10 人評價 ", 6.2, 10},
	}
	for _, tt := range tests {
		gotRating, gotCount := parseScore(tt.in)
		if gotRating != tt.wantRating || gotCount != tt.wantCount {
			t.Errorf("parseScore(%q) = (%v, %v), want (%v, %v)", tt.in, gotRating, gotCount, tt.wantRating, tt.wantCount)
		}
	}
}

func TestParseMovieItem(t *testing.T) {
	t.Parallel()
	html := `
<a class="box" href="/v/abc123">
  <img src="https://cdn.example/cover.jpg">
  <div class="video-title"><strong>SSNI-678</strong> Office Lady</div>
  <div class="score"><span class="value">8.5分, 由123人評價</span></div>
  <div class="meta">2020-01-02</div>
  <div class="tags"><span class="tag">磁鏈</span><span class="tag">字幕</span></div>
</a>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	c := &Client{baseURL: "https://javdb.com"}
	movie := c.parseMovieItem(doc.Find("a.box"))
	if movie.Code != "SSNI-678" {
		t.Errorf("code = %q", movie.Code)
	}
	if movie.Title != "Office Lady" {
		t.Errorf("title = %q", movie.Title)
	}
	if movie.URL != "https://javdb.com/v/abc123" {
		t.Errorf("url = %q", movie.URL)
	}
	if movie.Rating != 8.5 || movie.RatingCount != 123 {
		t.Errorf("rating = %v count = %v", movie.Rating, movie.RatingCount)
	}
	if movie.Date != "2020-01-02" {
		t.Errorf("date = %q", movie.Date)
	}
	if !movie.HasMagnet {
		t.Error("expected has_magnet")
	}
	if movie.ImageURL != "https://cdn.example/cover.jpg" {
		t.Errorf("image = %q", movie.ImageURL)
	}
}

func TestParseReviewItem(t *testing.T) {
	t.Parallel()
	html := `
<div class="review-item" id="review-item-42">
  <div class="review-title">User***name <span class="time">2024-01-02</span></div>
  <div class="score-stars"><i class="icon-star"></i><i class="icon-star"></i><i class="icon-star gray"></i></div>
  <span class="likes-count">7</span>
  <div class="content"><p>Nice movie</p></div>
</div>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	review := parseReviewItem(doc.Find(".review-item"))
	if review.ID != 42 {
		t.Errorf("id = %d", review.ID)
	}
	if review.Author != "User***name" {
		t.Errorf("author = %q", review.Author)
	}
	if review.Rating != 2 {
		t.Errorf("rating = %d", review.Rating)
	}
	if review.Likes != 7 {
		t.Errorf("likes = %d", review.Likes)
	}
	if review.Content != "Nice movie" {
		t.Errorf("content = %q", review.Content)
	}
	if review.Date != "2024-01-02" {
		t.Errorf("date = %q", review.Date)
	}
}

func TestParsePageFromQuery(t *testing.T) {
	t.Parallel()
	if got := parsePageFromQuery("/x?page=3&foo=1"); got != 3 {
		t.Errorf("got %d", got)
	}
	if got := parsePageFromQuery("/x"); got != 0 {
		t.Errorf("got %d", got)
	}
}

func TestLoginRequiredErrorUnwrap(t *testing.T) {
	t.Parallel()
	err := &LoginRequiredError{Message: "Unauthorized"}
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatal("expected errors.Is(ErrUnauthorized)")
	}
}
