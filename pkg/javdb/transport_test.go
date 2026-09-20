package javdb

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestSmartTransportHTTPUsesH1(t *testing.T) {
	t.Parallel()
	h1Called := false
	tr := &smartTransport{
		h2: roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("h2 should not be used for http")
			return nil, nil
		}),
		h1: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			h1Called = true
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok")), Request: req, Header: make(http.Header)}, nil
		}),
	}
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if !h1Called {
		t.Fatal("expected h1")
	}
}

func TestSmartTransportHTTPSFallback(t *testing.T) {
	t.Parallel()
	h1Called := false
	tr := &smartTransport{
		h2: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, io.EOF
		}),
		h1: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			h1Called = true
			return &http.Response{StatusCode: 200, Body: http.NoBody, Request: req, Header: make(http.Header)}, nil
		}),
	}
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	if _, err := tr.RoundTrip(req); err != nil {
		t.Fatal(err)
	}
	if !h1Called {
		t.Fatal("expected fallback to h1")
	}
}

func TestSmartTransportDoesNotRetryUnreplayableBody(t *testing.T) {
	t.Parallel()
	tr := &smartTransport{
		h2: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, io.EOF
		}),
		h1: roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("must not retry")
			return nil, nil
		}),
	}
	req, _ := http.NewRequest(http.MethodPost, "https://example.com", io.NopCloser(strings.NewReader("body")))
	if _, err := tr.RoundTrip(req); err == nil {
		t.Fatal("expected h2 error")
	}
}

func TestTransportPlainHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	}))
	t.Cleanup(srv.Close)

	client := &http.Client{Transport: newTransport("", false)}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestCanRetryRequest(t *testing.T) {
	t.Parallel()
	get, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	if !canRetryRequest(get) {
		t.Fatal("GET should retry")
	}
	post, _ := http.NewRequest(http.MethodPost, "https://example.com", io.NopCloser(strings.NewReader("x")))
	if canRetryRequest(post) {
		t.Fatal("POST without GetBody should not retry")
	}
	replayable, _ := http.NewRequest(http.MethodPost, "https://example.com", strings.NewReader("x"))
	if !canRetryRequest(replayable) {
		t.Fatal("POST with replayable body should retry")
	}
}
