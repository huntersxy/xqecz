package aigen

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

func publicIP(ip net.IP) bool {
	a, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	a = a.Unmap()
	if !a.IsGlobalUnicast() || a.IsPrivate() || a.IsLoopback() || a.IsLinkLocalUnicast() {
		return false
	}
	for _, raw := range []string{"100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "2001:db8::/32"} {
		if netip.MustParsePrefix(raw).Contains(a) {
			return false
		}
	}
	return true
}
func validateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Port() != "" && u.Port() != "443") || strings.ContainsAny(raw, "\r\n") {
		return errors.New("仅允许 HTTPS 公开供应商地址")
	}
	if ip := net.ParseIP(u.Hostname()); ip != nil && !publicIP(ip) {
		return errors.New("不允许内网地址")
	}
	return nil
}

// Resolve and pin each connection to a verified public IP, preventing DNS
// rebinding. Redirects and environment proxies cannot bypass this boundary.
func safeHTTPClient(timeout time.Duration) *http.Client {
	transport := &http.Transport{Proxy: nil, MaxResponseHeaderBytes: 64 << 10, ResponseHeaderTimeout: timeout, TLSHandshakeTimeout: 10 * time.Second, IdleConnTimeout: 30 * time.Second}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil || len(ips) == 0 {
			return nil, errors.New("供应商地址无法解析")
		}
		for _, ip := range ips {
			if !publicIP(ip.IP) {
				return nil, errors.New("供应商地址解析到非公开网络")
			}
		}
		dialer := net.Dialer{Timeout: 10 * time.Second}
		for _, ip := range ips {
			conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
			if dialErr == nil {
				return conn, nil
			}
			err = dialErr
		}
		return nil, err
	}
	return &http.Client{Transport: transport, Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}
