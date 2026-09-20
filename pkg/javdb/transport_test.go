package javdb

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
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

func TestParseSOCKS5URL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in      string
		scheme  string
		host    string
		user    string
		pass    string
		hasUser bool
		wantErr string
	}{
		{in: "127.0.0.1:1080", scheme: "socks5", host: "127.0.0.1:1080"},
		{in: "  socks5://127.0.0.1:1080 ", scheme: "socks5", host: "127.0.0.1:1080"},
		{in: "socks5h://127.0.0.1:1080", scheme: "socks5h", host: "127.0.0.1:1080"},
		{in: "socks://127.0.0.1:1080", scheme: "socks5", host: "127.0.0.1:1080"},
		{in: "user:pass@127.0.0.1:1080", scheme: "socks5", host: "127.0.0.1:1080", user: "user", pass: "pass", hasUser: true},
		{in: "socks5://alice:secret@127.0.0.1:1080", scheme: "socks5", host: "127.0.0.1:1080", user: "alice", pass: "secret", hasUser: true},
		{in: "socks5://alice:p%40ss%3Aword@127.0.0.1:1080", scheme: "socks5", host: "127.0.0.1:1080", user: "alice", pass: "p@ss:word", hasUser: true},
		{in: "socks5://[::1]:1080", scheme: "socks5", host: "[::1]:1080"},
		{in: "socks5://u:p@[::1]:1080", scheme: "socks5", host: "[::1]:1080", user: "u", pass: "p", hasUser: true},
		{in: "127.0.0.1", scheme: "socks5", host: "127.0.0.1"},
		{in: "http://127.0.0.1:8080", wantErr: "unsupported proxy scheme"},
		{in: "socks5://user:supersecret@", wantErr: "missing host"},
		{in: "socks5://:pass@127.0.0.1:1080", wantErr: "username is empty"},
	}
	for _, tt := range tests {
		u, err := parseSOCKS5URL(tt.in)
		if tt.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("parseSOCKS5URL(%q) err = %v, want %q", tt.in, err, tt.wantErr)
			}
			if err != nil && strings.Contains(err.Error(), "supersecret") {
				t.Errorf("parseSOCKS5URL(%q) leaked password: %v", tt.in, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseSOCKS5URL(%q) unexpected err %v", tt.in, err)
			continue
		}
		if u.Scheme != tt.scheme || u.Host != tt.host {
			t.Errorf("parseSOCKS5URL(%q) = %s://%s, want %s://%s", tt.in, u.Scheme, u.Host, tt.scheme, tt.host)
		}
		if tt.hasUser {
			if u.User == nil {
				t.Errorf("parseSOCKS5URL(%q) missing userinfo", tt.in)
				continue
			}
			pass, _ := u.User.Password()
			if u.User.Username() != tt.user || pass != tt.pass {
				t.Errorf("parseSOCKS5URL(%q) auth = %s:%s, want %s:%s", tt.in, u.User.Username(), pass, tt.user, tt.pass)
			}
		} else if u.User != nil {
			t.Errorf("parseSOCKS5URL(%q) unexpected userinfo %q", tt.in, u.User.String())
		}
	}
}

func TestSOCKS5NoAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "direct-ok")
	}))
	t.Cleanup(srv.Close)
	proxyAddr := startTestSOCKS5(t, "", "")
	got := httpGetViaProxy(t, proxyAddr, srv.URL)
	if got != "direct-ok" {
		t.Fatalf("got %q", got)
	}
}

func TestSOCKS5UsernamePassword(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "auth-ok")
	}))
	t.Cleanup(srv.Close)
	proxyAddr := startTestSOCKS5(t, "alice", "p@ss:word")

	encoded := (&url.URL{
		Scheme: "socks5",
		User:   url.UserPassword("alice", "p@ss:word"),
		Host:   proxyAddr,
	}).String()
	if got := httpGetViaProxy(t, encoded, srv.URL); got != "auth-ok" {
		t.Fatalf("url form got %q", got)
	}

	bare := "alice:" + url.QueryEscape("p@ss:word") + "@" + proxyAddr
	if got := httpGetViaProxy(t, bare, srv.URL); got != "auth-ok" {
		t.Fatalf("userinfo form got %q", got)
	}
}

func TestSOCKS5WrongPassword(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("origin should not be reached")
	}))
	t.Cleanup(srv.Close)
	proxyAddr := startTestSOCKS5(t, "alice", "secret")
	client := &http.Client{Transport: newTransport("socks5://alice:wrong@"+proxyAddr, false), Timeout: 5 * time.Second}
	resp, err := client.Get(srv.URL)
	if err == nil {
		resp.Body.Close()
		t.Fatal("expected authentication error")
	}
	if !strings.Contains(err.Error(), "authentication") && !strings.Contains(err.Error(), "username/password") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSOCKS5AuthRequiredWithoutCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("origin should not be reached")
	}))
	t.Cleanup(srv.Close)
	proxyAddr := startTestSOCKS5(t, "alice", "secret")
	client := &http.Client{Transport: newTransport(proxyAddr, false), Timeout: 5 * time.Second}
	resp, err := client.Get(srv.URL)
	if err == nil {
		resp.Body.Close()
		t.Fatal("expected error without credentials")
	}
}

func TestSOCKS5HTTPS(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "tls-ok")
	}))
	t.Cleanup(srv.Close)
	proxyAddr := startTestSOCKS5(t, "bob", "tls-secret")
	proxyURL := fmt.Sprintf("socks5://bob:%s@%s", url.QueryEscape("tls-secret"), proxyAddr)
	got := httpGetViaProxy(t, proxyURL, srv.URL)
	if got != "tls-ok" {
		t.Fatalf("got %q", got)
	}
}

func httpGetViaProxy(t *testing.T, proxyAddr, target string) string {
	t.Helper()
	client := &http.Client{
		Transport: newTransport(proxyAddr, true),
		Timeout:   5 * time.Second,
	}
	resp, err := client.Get(target)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d body %s", resp.StatusCode, body)
	}
	return string(body)
}

func startTestSOCKS5(t *testing.T, username, password string) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go serveTestSOCKS5(conn, username, password)
		}
	}()
	return ln.Addr().String()
}

func serveTestSOCKS5(conn net.Conn, username, password string) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if err := handshakeTestSOCKS5(conn, username, password); err != nil {
		return
	}
	dest, err := readTestSOCKS5Connect(conn)
	if err != nil {
		return
	}
	target, err := net.DialTimeout("tcp", dest, 3*time.Second)
	if err != nil {
		_, _ = conn.Write([]byte{5, 5, 0, 1, 0, 0, 0, 0, 0, 0})
		return
	}
	defer target.Close()
	if _, err := conn.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0}); err != nil {
		return
	}
	_ = conn.SetDeadline(time.Time{})
	errc := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(target, conn); errc <- struct{}{} }()
	go func() { _, _ = io.Copy(conn, target); errc <- struct{}{} }()
	<-errc
}

func handshakeTestSOCKS5(conn net.Conn, username, password string) error {
	hdr := make([]byte, 2)
	if _, err := io.ReadFull(conn, hdr); err != nil {
		return err
	}
	if hdr[0] != 5 {
		return fmt.Errorf("bad socks version %d", hdr[0])
	}
	methods := make([]byte, int(hdr[1]))
	if _, err := io.ReadFull(conn, methods); err != nil {
		return err
	}
	if username == "" {
		_, err := conn.Write([]byte{5, 0})
		return err
	}
	if !bytesContains(methods, 2) {
		_, _ = conn.Write([]byte{5, 0xff})
		return fmt.Errorf("client did not offer username/password")
	}
	if _, err := conn.Write([]byte{5, 2}); err != nil {
		return err
	}
	auth := make([]byte, 2)
	if _, err := io.ReadFull(conn, auth); err != nil {
		return err
	}
	if auth[0] != 1 {
		return fmt.Errorf("bad auth version %d", auth[0])
	}
	uname := make([]byte, int(auth[1]))
	if _, err := io.ReadFull(conn, uname); err != nil {
		return err
	}
	plen := make([]byte, 1)
	if _, err := io.ReadFull(conn, plen); err != nil {
		return err
	}
	passwd := make([]byte, int(plen[0]))
	if _, err := io.ReadFull(conn, passwd); err != nil {
		return err
	}
	if string(uname) != username || string(passwd) != password {
		_, _ = conn.Write([]byte{1, 1})
		return fmt.Errorf("bad credentials")
	}
	_, err := conn.Write([]byte{1, 0})
	return err
}

func readTestSOCKS5Connect(conn net.Conn) (string, error) {
	hdr := make([]byte, 4)
	if _, err := io.ReadFull(conn, hdr); err != nil {
		return "", err
	}
	if hdr[0] != 5 || hdr[1] != 1 {
		return "", fmt.Errorf("unsupported socks request")
	}
	var host string
	switch hdr[3] {
	case 1:
		ip := make([]byte, 4)
		if _, err := io.ReadFull(conn, ip); err != nil {
			return "", err
		}
		host = net.IP(ip).String()
	case 4:
		ip := make([]byte, 16)
		if _, err := io.ReadFull(conn, ip); err != nil {
			return "", err
		}
		host = net.IP(ip).String()
	case 3:
		l := make([]byte, 1)
		if _, err := io.ReadFull(conn, l); err != nil {
			return "", err
		}
		name := make([]byte, int(l[0]))
		if _, err := io.ReadFull(conn, name); err != nil {
			return "", err
		}
		host = string(name)
	default:
		return "", fmt.Errorf("bad address type %d", hdr[3])
	}
	pb := make([]byte, 2)
	if _, err := io.ReadFull(conn, pb); err != nil {
		return "", err
	}
	port := int(pb[0])<<8 | int(pb[1])
	return net.JoinHostPort(host, fmt.Sprintf("%d", port)), nil
}

func bytesContains(b []byte, v byte) bool {
	for _, x := range b {
		if x == v {
			return true
		}
	}
	return false
}
