package javdb

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
	"golang.org/x/net/proxy"
)

// newTransport creates a specialized transport that mimics Chrome.
// It prioritizes HTTP/2 using utls and falls back to HTTP/1.1 if needed.
func newTransport() http.RoundTripper {
	return &smartTransport{
		h2: &http2.Transport{
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return dialUTLS(ctx, network, addr, []string{"h2"})
			},
			AllowHTTP: false,
		},
		h1: &http.Transport{
			DialContext: dialBase,
			DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialUTLS(ctx, network, addr, []string{"http/1.1"})
			},
		},
	}
}

type smartTransport struct {
	h2 *http2.Transport
	h1 *http.Transport
}

func (t *smartTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme != "https" {
		return t.h1.RoundTrip(req)
	}
	// Try HTTP/2 first (Chrome prefers H2)
	resp, err := t.h2.RoundTrip(req)
	if err == nil {
		return resp, nil
	}
	// Fallback to HTTP/1.1 if H2 fails (e.g. server reset, timeout, or negotiation fail)
	return t.h1.RoundTrip(req)
}

// dialBase handles the underlying TCP connection, including SOCKS5 proxy support.
func dialBase(ctx context.Context, network, addr string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	
	if proxyAddr := os.Getenv("SOCKS5_PROXY"); proxyAddr != "" {
		if d, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct); err == nil {
			if cd, ok := d.(proxy.ContextDialer); ok {
				return cd.DialContext(ctx, network, addr)
			}
		}
	}
	return dialer.DialContext(ctx, network, addr)
}

// dialUTLS establishes a TLS connection using utls with Chrome fingerprint.
func dialUTLS(ctx context.Context, network, addr string, alpn []string) (net.Conn, error) {
	conn, err := dialBase(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	hostname := addr
	if host, _, err := net.SplitHostPort(addr); err == nil {
		hostname = host
	}

	uConn := utls.UClient(conn, &utls.Config{
		ServerName:         hostname,
		InsecureSkipVerify: true,
		NextProtos:         alpn,
	}, utls.HelloChrome_Auto)

	// Force ALPN to match our expectation (h2 or http/1.1)
	spec, err := utls.UTLSIdToSpec(utls.HelloChrome_Auto)
	if err == nil {
		for i, ext := range spec.Extensions {
			if _, ok := ext.(*utls.ALPNExtension); ok {
				spec.Extensions[i] = &utls.ALPNExtension{AlpnProtocols: alpn}
			}
		}
		uConn.ApplyPreset(&spec)
	}

	if err := uConn.Handshake(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("utls handshake failed: %w", err)
	}

	return uConn, nil
}
