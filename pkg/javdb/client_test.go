package javdb

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const movieListHTML = `
<!DOCTYPE html>
<html><body>
<a class="box" href="/v/abc123">
  <img src="https://cdn.example/cover.jpg">
  <div class="video-title"><strong>SSNI-678</strong> Office Lady</div>
  <div class="score"><span class="value">8.5分, 由123人評價</span></div>
  <div class="meta">2020-01-02</div>
  <div class="tags"><span class="tag">磁鏈</span></div>
</a>
</body></html>`

const detailHTML = `
<!DOCTYPE html>
<html>
<head><meta name="csrf-token" content="csrf-test-token"></head>
<body>
<h2 class="title"><strong>SSNI-678 Office Lady</strong></h2>
<div class="column-video-cover"><img src="https://cdn.example/cover.jpg"></div>
<div class="movie-panel-info">
  <div class="panel-block first-block"><strong>番號:</strong><span class="value">SSNI-678</span></div>
  <div class="panel-block"><strong>日期:</strong><span class="value">2020-01-02</span></div>
  <div class="panel-block"><strong>時長:</strong><span class="value">120分鐘</span></div>
  <div class="panel-block"><strong>導演:</strong><span class="value">Some Director</span></div>
  <div class="panel-block"><strong>片商:</strong><span class="value">S1</span></div>
  <div class="panel-block"><strong>系列:</strong><span class="value">Office</span></div>
  <div class="panel-block"><strong>評分:</strong><span class="value">8.5分, 由123人評價</span></div>
  <div class="panel-block"><strong>類別:</strong><span class="value"><a>OL</a><a>痴漢</a></span></div>
  <div class="panel-block"><strong>演員:</strong><span class="value"><a href="/actors/1">Yua</a></span></div>
</div>
<div class="magnet-links">
  <div class="item columns is-desktop">
    <div class="magnet-name">
      <a href="magnet:?xt=urn:btih:deadbeef">link</a>
      <span class="name">SSNI-678.mp4</span>
      <span class="meta">1.2GB</span>
      <div class="tags"><span class="tag">高清</span><span class="tag">字幕</span></div>
    </div>
    <div class="date"><span class="time">2020-02-02</span></div>
  </div>
</div>
</body></html>`

const reviewsHTML = `
<!DOCTYPE html>
<html><body>
<div class="review-items">
  <div class="review-item" id="review-item-7">
    <div class="review-title">A***B <span class="time">2021-03-04</span></div>
    <div class="score-stars"><i class="icon-star"></i><i class="icon-star"></i><i class="icon-star"></i><i class="icon-star gray"></i></div>
    <span class="likes-count">4</span>
    <div class="content"><p>ok</p></div>
  </div>
</div>
<a class="pagination-next" rel="next" href="?page=2">next</a>
</body></html>`

const bookmarksHTML = `
<!DOCTYPE html>
<html><body>
<div id="videos">
  <div class="item">
    <div class="box">
      <a href="/v/abc123">
        <div class="video-title"><strong>SSNI-678</strong> Office Lady</div>
      </a>
    </div>
    <a href="/v/abc123/reviews/42">review</a>
  </div>
</div>
<a class="pagination-next" rel="next" href="?page=2">next</a>
</body></html>`

func newTestClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return New(WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
}

func TestSearch(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" || r.URL.Query().Get("q") != "SSNI-678" {
			t.Errorf("unexpected request %s", r.URL)
		}
		if r.Header.Get("Cookie") == "" {
			t.Error("missing cookie")
		}
		_, _ = io.WriteString(w, movieListHTML)
	})
	res, err := c.Search(context.Background(), "SSNI-678")
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 || res.Movies[0].Code != "SSNI-678" {
		b, _ := json.Marshal(res)
		t.Fatalf("unexpected result %s", b)
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	c := New(WithHTTPClient(&http.Client{}))
	if _, err := c.Search(context.Background(), "  "); err == nil {
		t.Fatal("expected error")
	}
}

func TestGetRankings(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rankings/movies" {
			t.Errorf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("p") != "weekly" || r.URL.Query().Get("t") != "fc2" {
			t.Errorf("query %s", r.URL.RawQuery)
		}
		_, _ = io.WriteString(w, movieListHTML)
	})
	res, err := c.GetRankings(context.Background(), "weekly", "fc2")
	if err != nil {
		t.Fatal(err)
	}
	if res.Period != "weekly" || res.Type != "fc2" || len(res.Movies) != 1 {
		t.Fatalf("%+v", res)
	}
}

func TestGetDetail(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/search":
			_, _ = io.WriteString(w, movieListHTML)
		case strings.HasPrefix(r.URL.Path, "/v/"):
			_, _ = io.WriteString(w, detailHTML)
		default:
			http.NotFound(w, r)
		}
	})
	d, err := c.GetDetail(context.Background(), "SSNI-678")
	if err != nil {
		t.Fatal(err)
	}
	if d.Code != "SSNI-678" || d.Director != "Some Director" || d.Duration != "120分鐘" {
		t.Fatalf("detail meta %+v", d)
	}
	if len(d.Actors) != 1 || d.Actors[0].Name != "Yua" {
		t.Fatalf("actors %+v", d.Actors)
	}
	if len(d.Magnets) != 1 || !d.Magnets[0].IsHD || !d.Magnets[0].HasSubs {
		t.Fatalf("magnets %+v", d.Magnets)
	}
	if d.Magnets[0].Magnet != "magnet:?xt=urn:btih:deadbeef" {
		t.Fatalf("magnet %s", d.Magnets[0].Magnet)
	}
}

func TestGetReviews(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/search" {
			_, _ = io.WriteString(w, movieListHTML)
			return
		}
		if r.URL.Path != "/v/abc123/reviews/lastest" {
			t.Errorf("path %s", r.URL.Path)
		}
		_, _ = io.WriteString(w, reviewsHTML)
	})
	res, err := c.GetReviews(context.Background(), "SSNI-678", 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.Page != 1 || !res.HasNext || res.NextPage != 2 || len(res.Reviews) != 1 {
		t.Fatalf("%+v", res)
	}
	if res.Reviews[0].ID != 7 || res.Reviews[0].Rating != 3 {
		t.Fatalf("review %+v", res.Reviews[0])
	}
}

func TestGetBookmarks(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/want_watch_videos" {
			t.Errorf("path %s", r.URL.Path)
		}
		_, _ = io.WriteString(w, bookmarksHTML)
	})
	res, err := c.GetBookmarks(context.Background(), "want_watch", 1)
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 || res.Movies[0].ReviewID != 42 || !res.HasNext {
		t.Fatalf("%+v", res)
	}
}

func TestAddWantWatch(t *testing.T) {
	var posted bool
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/search":
			_, _ = io.WriteString(w, movieListHTML)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v/"):
			_, _ = io.WriteString(w, detailHTML)
		case r.Method == http.MethodPost && r.URL.Path == "/v/abc123/reviews/want_to_watch":
			if r.Header.Get("X-CSRF-Token") != "csrf-test-token" {
				t.Errorf("csrf %s", r.Header.Get("X-CSRF-Token"))
			}
			posted = true
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	})
	if err := c.AddWantWatch(context.Background(), "SSNI-678"); err != nil {
		t.Fatal(err)
	}
	if !posted {
		t.Fatal("expected POST")
	}
}

func TestRemoveWantWatch(t *testing.T) {
	var deleted bool
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/users/want_watch_videos":
			_, _ = io.WriteString(w, bookmarksHTML)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v/"):
			_, _ = io.WriteString(w, detailHTML)
		case r.Method == http.MethodDelete && r.URL.Path == "/v/abc123/reviews/42":
			deleted = true
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	})
	if err := c.RemoveWantWatch(context.Background(), "SSNI-678"); err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("expected DELETE")
	}
}

func TestGetBookmarksActors(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/collection_actors" {
			t.Errorf("path %s", r.URL.Path)
		}
		_, _ = io.WriteString(w, `
<div id="actors">
  <div class="actor-box">
    <a href="/actors/1"><img class="avatar" src="https://cdn.example/a.jpg"><strong>Yua</strong></a>
    <a href="/actors/1/uncollect">x</a>
  </div>
</div>`)
	})
	res, err := c.GetBookmarks(context.Background(), "actors", 1)
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 || res.Actors[0].Name != "Yua" || res.Actors[0].URL == "" {
		t.Fatalf("%+v", res)
	}
}

func TestAddWatched(t *testing.T) {
	var posted bool
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/search":
			_, _ = io.WriteString(w, movieListHTML)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v/"):
			_, _ = io.WriteString(w, detailHTML)
		case r.Method == http.MethodPost && r.URL.Path == "/v/abc123/reviews":
			_ = r.ParseForm()
			if r.Form.Get("video_review[score]") != "5" || r.Form.Get("video_review[status]") != "watched" {
				t.Errorf("form %v", r.Form)
			}
			posted = true
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	})
	if err := c.AddWatched(context.Background(), "SSNI-678", 5, "great"); err != nil {
		t.Fatal(err)
	}
	if !posted {
		t.Fatal("expected POST")
	}
}

func TestUnknownBookmarkType(t *testing.T) {
	c := New(WithHTTPClient(&http.Client{}))
	if _, err := c.GetBookmarks(context.Background(), "nope", 1); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoginRequired(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/users/sign_in", http.StatusFound)
	})
	mux.HandleFunc("/users/sign_in", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "<html>login</html>")
	})
	c := newTestClient(t, mux.ServeHTTP)
	_, err := c.Search(context.Background(), "SSNI-678")
	var lr *LoginRequiredError
	if !errors.As(err, &lr) {
		t.Fatalf("got %v", err)
	}
}

func TestHTTPErrorStatus(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusTeapot)
	})
	_, err := c.Search(context.Background(), "SSNI-678")
	var he *HTTPError
	if !errors.As(err, &he) || he.StatusCode != http.StatusTeapot {
		t.Fatalf("got %v", err)
	}
}
