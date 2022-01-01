package account

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func newGAProxy() *httputil.ReverseProxy {
	targetURL, _ := url.Parse("https://www.google-analytics.com")

	director := func(req *http.Request) {
		req.Host = targetURL.Host
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		ip := requestIP(req)
		values, _ := url.ParseQuery(req.URL.RawQuery)
		values.Set("uip", ip)
		req.URL.RawQuery = values.Encode()
	}
	return &httputil.ReverseProxy{Director: director}
}

func requestIP(r *http.Request) string {
	if r == nil {
		return ""
	}

	remoteIP := r.Header.Get("X-Forwarded-For")

	// Intermediate proxies should be append their IP in this field,
	// meaning the client IP is the first entry.
	parts := strings.Split(remoteIP, ",")
	if len(parts) > 0 {
		ipStr := strings.TrimSpace(parts[0])
		if ip := net.ParseIP(parts[0]); ip != nil {
			return ipStr
		}
		if host, _, err := net.SplitHostPort(parts[0]); err == nil {
			if ip := net.ParseIP(host); ip != nil {
				return host
			}
		}
	}

	log.Println("unable to extract IP from X-Forwarded-For:", remoteIP)

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return ""
	}

	return host
}
