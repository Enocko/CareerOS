package middleware

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP rewrites r.RemoteAddr to the real client IP.
//
// trustedProxyHops is the number of reverse proxies in front of the API that
// append to X-Forwarded-For (counting the proxy that opens the connection).
// Only those trailing entries are trusted; anything further left was supplied
// by the client and can be forged. With 0 hops the TCP peer address is used and
// X-Forwarded-For is ignored entirely.
func ClientIP(trustedProxyHops int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.RemoteAddr = resolveClientIP(r.RemoteAddr, r.Header.Values("X-Forwarded-For"), trustedProxyHops)
			next.ServeHTTP(w, r)
		})
	}
}

func resolveClientIP(remoteAddr string, forwardedFor []string, trustedProxyHops int) string {
	peer := hostOnly(remoteAddr)
	if trustedProxyHops <= 0 {
		return peer
	}

	// Chain ordered client → nearest proxy, ending with the TCP peer.
	var chain []string
	for _, header := range forwardedFor {
		for _, part := range strings.Split(header, ",") {
			if ip := strings.TrimSpace(part); ip != "" {
				chain = append(chain, ip)
			}
		}
	}
	chain = append(chain, peer)

	idx := len(chain) - 1 - trustedProxyHops
	if idx < 0 {
		idx = 0
	}
	if ip := net.ParseIP(hostOnly(chain[idx])); ip != nil {
		return ip.String()
	}
	return peer
}

func hostOnly(addr string) string {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return strings.Trim(addr, "[]")
}
