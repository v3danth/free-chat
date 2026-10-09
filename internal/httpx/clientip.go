package httpx

import (
	"net"
	"net/http"
	"strings"
)

// IPResolver returns the client IP for a request.
type IPResolver func(*http.Request) string

// NewIPResolver trusts X-Forwarded-For only when the server sits behind a
// reverse proxy it controls; otherwise the header is attacker-controlled and
// the socket address is the only truth. With a proxy, the last entry is the
// one that proxy appended.
func NewIPResolver(behindProxy bool) IPResolver {
	return func(r *http.Request) string {
		if behindProxy {
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				parts := strings.Split(xff, ",")
				return strings.TrimSpace(parts[len(parts)-1])
			}
		}
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return r.RemoteAddr
		}
		return host
	}
}
