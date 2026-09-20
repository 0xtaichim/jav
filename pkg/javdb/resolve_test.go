package javdb

import (
	"testing"
)

func TestVideoIDFromURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		base, raw, want string
		wantErr         bool
	}{
		{"https://javdb.com", "https://javdb.com/v/abc123", "abc123", false},
		{"https://javdb.com", "https://javdb.com/v/abc123?locale=zh", "abc123", false},
		{"https://javdb.com", "https://javdb.com/v/abc123/reviews", "abc123", false},
		{"https://javdb.com", "/v/rel456", "rel456", false},
		{"https://javdb.com", "https://example.com/x", "", true},
		{"https://javdb.com", "https://javdb.com/v/", "", true},
	}
	for _, tt := range tests {
		got, err := videoIDFromURL(tt.base, tt.raw)
		if tt.wantErr {
			if err == nil {
				t.Errorf("videoIDFromURL(%q) expected error", tt.raw)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("videoIDFromURL(%q) = (%q, %v), want %q", tt.raw, got, err, tt.want)
		}
	}
}

func TestPickMovieURL(t *testing.T) {
	t.Parallel()
	movies := []Movie{
		{Code: "AAA-001", URL: "https://javdb.com/v/aaa"},
		{Code: "SSNI-678", URL: "https://javdb.com/v/ssni"},
	}
	if got := pickMovieURL(movies, "ssni-678"); got != movies[1].URL {
		t.Errorf("exact match ignored: %s", got)
	}
	if got := pickMovieURL(movies, "NOPE"); got != movies[0].URL {
		t.Errorf("fallback = %s", got)
	}
}

func TestInjectLocaleIntoCookies(t *testing.T) {
	t.Parallel()
	got := injectLocaleIntoCookies("over18=1; locale=en; theme=auto", "zh")
	if got != "over18=1; locale=zh; theme=auto" {
		t.Errorf("replace = %q", got)
	}
	got = injectLocaleIntoCookies("over18=1", "zh")
	if got != "over18=1; locale=zh" {
		t.Errorf("append = %q", got)
	}
}

func TestBuildCookiesDefault(t *testing.T) {
	t.Parallel()
	got := buildCookies("", "zh")
	if got != "over18=1; theme=auto; locale=zh" {
		t.Errorf("got %q", got)
	}
}

func TestNormalizeRanking(t *testing.T) {
	t.Parallel()
	p, typ, err := normalizeRanking("", "")
	if err != nil || p != "daily" || typ != "censored" {
		t.Fatalf("defaults = %s %s %v", p, typ, err)
	}
	if _, _, err := normalizeRanking("yearly", "censored"); err == nil {
		t.Fatal("expected period error")
	}
	if _, _, err := normalizeRanking("daily", "foo"); err == nil {
		t.Fatal("expected type error")
	}
}

func TestReviewIDFromHref(t *testing.T) {
	t.Parallel()
	if got := reviewIDFromHref("/v/abc/reviews/99"); got != 99 {
		t.Errorf("got %d", got)
	}
	if got := reviewIDFromHref("/v/abc/reviews/lastest"); got != 0 {
		t.Errorf("got %d", got)
	}
}

func TestNormalizePage(t *testing.T) {
	t.Parallel()
	if normalizePage(0) != 1 || normalizePage(-3) != 1 || normalizePage(4) != 4 {
		t.Fatal("normalizePage")
	}
}
