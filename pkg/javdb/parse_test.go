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

func parseMagnetsFromHTML(t *testing.T, html string) []MagnetLink {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatal(err)
	}
	detail := &MovieDetail{Magnets: []MagnetLink{}}
	(&Client{}).parseMagnetLinks(doc, detail)
	return detail.Magnets
}

func TestParseMagnetLinksCurrentMarkup(t *testing.T) {
	t.Parallel()
	html := `
<div id="magnets-content" class="magnet-links">
  <div class="item">
    <div class="magnet-name">
      <a href="magnet:?xt=urn:btih:aaa111">link</a>
      <span class="name">SSNI-678-C.mp4</span>
      <span class="meta">2.1GB</span>
      <div class="tags"><span class="tag">高清</span><span class="tag">字幕</span></div>
    </div>
    <div class="date"><span class="time">2024-05-06</span></div>
  </div>
  <div class="item">
    <div class="magnet-name">
      <a href="magnet:?xt=urn:btih:bbb222">link</a>
      <span class="name">SSNI-678.mp4</span>
      <span class="meta">1.8GB</span>
    </div>
    <div class="date"><span class="time">2024-05-07</span></div>
  </div>
</div>`
	magnets := parseMagnetsFromHTML(t, html)
	if len(magnets) != 2 {
		t.Fatalf("len = %d, magnets = %+v", len(magnets), magnets)
	}
	if magnets[0].Magnet != "magnet:?xt=urn:btih:aaa111" || magnets[0].Name != "SSNI-678-C.mp4" || magnets[0].Size != "2.1GB" || magnets[0].Date != "2024-05-06" {
		t.Fatalf("first magnet %+v", magnets[0])
	}
	if !magnets[0].IsHD || !magnets[0].HasSubs {
		t.Fatalf("expected HD+subs on first magnet %+v", magnets[0])
	}
	if magnets[1].Magnet != "magnet:?xt=urn:btih:bbb222" || magnets[1].IsHD || magnets[1].HasSubs {
		t.Fatalf("second magnet %+v", magnets[1])
	}
}

func TestParseMagnetLinksClipboardFallback(t *testing.T) {
	t.Parallel()
	html := `
<div id="magnets-content" class="magnet-links">
  <div class="item">
    <div class="magnet-name">
      <span class="name">SSNI-678.mp4</span>
      <span class="meta">1.2GB</span>
    </div>
    <button class="copy-to-clipboard" data-clipboard-text="magnet:?xt=urn:btih:clipme">Copy</button>
    <div class="date"><span class="time">2024-08-01</span></div>
  </div>
  <div class="item">
    <div class="magnet-name"><span class="name">not-a-magnet</span></div>
    <button class="copy-to-clipboard" data-clipboard-text="https://example.com/not-magnet">Copy</button>
  </div>
</div>`
	magnets := parseMagnetsFromHTML(t, html)
	if len(magnets) != 1 {
		t.Fatalf("len = %d, magnets = %+v", len(magnets), magnets)
	}
	if magnets[0].Magnet != "magnet:?xt=urn:btih:clipme" || magnets[0].Name != "SSNI-678.mp4" {
		t.Fatalf("magnet %+v", magnets[0])
	}
}

func TestParseMagnetLinksDedupeAndLegacyMarkup(t *testing.T) {
	t.Parallel()
	html := `
<div class="magnet-links">
  <div class="item columns is-desktop">
    <div class="magnet-name">
      <a href="magnet:?xt=urn:btih:deadbeef">link</a>
      <span class="name">legacy.mp4</span>
      <span class="meta">1.2GB</span>
      <div class="tags"><span class="tag">HD</span><span class="tag">中字</span></div>
    </div>
    <button class="copy-to-clipboard" data-clipboard-text="magnet:?xt=urn:btih:deadbeef">Copy</button>
    <div class="date"><span class="time">2020-02-02</span></div>
  </div>
</div>`
	magnets := parseMagnetsFromHTML(t, html)
	if len(magnets) != 1 {
		t.Fatalf("expected 1 unique magnet, got %d: %+v", len(magnets), magnets)
	}
	if magnets[0].Magnet != "magnet:?xt=urn:btih:deadbeef" || magnets[0].Name != "legacy.mp4" {
		t.Fatalf("magnet %+v", magnets[0])
	}
	if !magnets[0].IsHD || !magnets[0].HasSubs {
		t.Fatalf("expected HD+subs %+v", magnets[0])
	}
}

func TestLoginRequiredErrorUnwrap(t *testing.T) {
	t.Parallel()
	err := &LoginRequiredError{Message: "Unauthorized"}
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatal("expected errors.Is(ErrUnauthorized)")
	}
}
