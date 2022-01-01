package auth

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/kiteco/kiteco/kite-golib/domains"
	"golang.org/x/net/publicsuffix"
)

// cookieJar is a http.CookieJar that copies cookies from *.kite.com to rc.kite.com.
// It enables authenticating with staging.kite.com, and reusing the session on rc.kite.com.
type cookieJar struct {
	j http.CookieJar
}

func newCookieJar() http.CookieJar {
	j, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return nil
	}
	return cookieJar{j}
}

func (j cookieJar) Cookies(u *url.URL) []*http.Cookie {
	return j.j.Cookies(u)
}

func (j cookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.j.SetCookies(u, cookies)

	host := u.Hostname()
	if !strings.HasSuffix(host, "."+domains.PrimaryHost) {
		return
	}
	v := *u
	v.Host = domains.RemoteConfig
	j.j.SetCookies(&v, cookies)
}
