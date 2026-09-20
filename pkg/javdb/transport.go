package javdb

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
	"golang.org/x/net/proxy"
)

func newTransport(proxyAddr string, insecure bool) http.RoundTripper {
	d := newProxyDialer(proxyAddr)
	return &smartTransport{
		h2: &http2.Transport{
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return d.dialUTLS(ctx, network, addr, []string{"h2"}, insecure)
			},
			AllowHTTP: false,
		},
		h1: &http.Transport{
			DialContext: d.dial,
			DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return d.dialUTLS(ctx, network, addr, []string{"http/1.1"}, insecure)
			},
			ForceAttemptHTTP2: false,
		},
	}
}

type smartTransport struct {
	h2 http.RoundTripper
	h1 http.RoundTripper
}

func (t *smartTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme != "https" {
		return t.h1.RoundTrip(req)
	}

	resp, err := t.h2.RoundTrip(req)
	if err == nil {
		return resp, nil
	}
	if !canRetryRequest(req) {
		return nil, err
	}
	if rewindErr := rewindBody(req); rewindErr != nil {
		return nil, err
	}
	return t.h1.RoundTrip(req)
}

func canRetryRequest(req *http.Request) bool {
	if req.Body == nil || req.Body == http.NoBody {
		return true
	}
	return req.GetBody != nil
}

func rewindBody(req *http.Request) error {
	if req.Body == nil || req.Body == http.NoBody {
		return nil
	}
	if req.GetBody == nil {
		return fmt.Errorf("request body cannot be replayed")
	}
	body, err := req.GetBody()
	if err != nil {
		return err
	}
	req.Body = body
	return nil
}

type proxyDialer struct {
	base  *net.Dialer
	socks proxy.Dialer
	err   error
}

func newProxyDialer(proxyAddr string) *proxyDialer {
	base := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	d := &proxyDialer{base: base}
	proxyAddr = strings.TrimSpace(proxyAddr)
	if proxyAddr == "" {
		return d
	}
	socks, err := newSOCKS5Dialer(proxyAddr, base)
	if err != nil {
		d.err = err
		return d
	}
	d.socks = socks
	return d
}

func (d *proxyDialer) dial(ctx context.Context, network, addr string) (net.Conn, error) {
	if d.err != nil {
		return nil, d.err
	}
	if d.socks == nil {
		return d.base.DialContext(ctx, network, addr)
	}
	if cd, ok := d.socks.(proxy.ContextDialer); ok {
		return cd.DialContext(ctx, network, addr)
	}
	return dialWithContext(ctx, d.socks, network, addr)
}

func dialWithContext(ctx context.Context, d proxy.Dialer, network, addr string) (net.Conn, error) {
	var (
		conn net.Conn
		err  error
		done = make(chan struct{})
	)
	go func() {
		conn, err = d.Dial(network, addr)
		close(done)
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-done:
		return conn, err
	}
}

func (d *proxyDialer) dialUTLS(ctx context.Context, network, addr string, alpn []string, insecure bool) (net.Conn, error) {
	conn, err := d.dial(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	hostname := addr
	if host, _, splitErr := net.SplitHostPort(addr); splitErr == nil {
		hostname = host
	}

	uConn := utls.UClient(conn, &utls.Config{
		ServerName:         hostname,
		InsecureSkipVerify: insecure,
		NextProtos:         alpn,
	}, utls.HelloChrome_Auto)

	if spec, specErr := utls.UTLSIdToSpec(utls.HelloChrome_Auto); specErr == nil {
		for i, ext := range spec.Extensions {
			if _, ok := ext.(*utls.ALPNExtension); ok {
				spec.Extensions[i] = &utls.ALPNExtension{AlpnProtocols: alpn}
			}
		}
		if applyErr := uConn.ApplyPreset(&spec); applyErr != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("utls preset failed: %w", applyErr)
		}
	}

	if err := uConn.HandshakeContext(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("utls handshake failed: %w", err)
	}
	return uConn, nil
}

func newSOCKS5Dialer(raw string, forward proxy.Dialer) (proxy.Dialer, error) {
	u, err := parseSOCKS5URL(raw)
	if err != nil {
		return nil, err
	}
	dialer, err := proxy.FromURL(u, forward)
	if err != nil {
		return nil, fmt.Errorf("socks5 proxy %s: %w", u.Host, err)
	}
	return dialer, nil
}

// parseSOCKS5URL accepts host:port, user:pass@host:port, or a socks/socks5/socks5h URL.
// Username/password auth (RFC 1929) is taken from the URL userinfo.
func parseSOCKS5URL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("socks5 proxy address is empty")
	}
	if !strings.Contains(raw, "://") {
		raw = "socks5://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid socks5 proxy address")
	}
	switch strings.ToLower(u.Scheme) {
	case "socks":
		u.Scheme = "socks5"
	case "socks5", "socks5h":
	default:
		return nil, fmt.Errorf("unsupported proxy scheme %q (only socks5)", u.Scheme)
	}
	if u.Hostname() == "" {
		return nil, fmt.Errorf("socks5 proxy missing host")
	}
	if u.User != nil {
		if u.User.Username() == "" {
			return nil, fmt.Errorf("socks5 proxy username is empty")
		}
		if len(u.User.Username()) > 255 {
			return nil, fmt.Errorf("socks5 proxy username is too long")
		}
		if pass, ok := u.User.Password(); ok && len(pass) > 255 {
			return nil, fmt.Errorf("socks5 proxy password is too long")
		}
	}
	return u, nil
}
