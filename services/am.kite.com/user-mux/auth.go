package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/codegangsta/negroni"
	"github.com/kiteco/kiteco/kite-go/community"
	"github.com/kiteco/kiteco/kite-go/hmacutil"
)

type contextKey string

func (c contextKey) String() string {
	return fmt.Sprintf("user-mux:%s", string(c))
}

const (
	userKey   = contextKey("user")
	regionKey = "Kite-Region"
)

func getUser(r *http.Request) (*community.User, bool) {
	user, ok := r.Context().Value(userKey).(*community.User)
	return user, ok
}

func getID(r *http.Request) (string, bool) {
	vals := r.URL.Query()
	id := vals.Get("id")
	if id != "" {
		return id, true
	}
	return "", false
}

type userAuthMidware struct {
	users *community.UserManager
}

func (u *userAuthMidware) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	start := time.Now()

	// Bypass authentication logic for /ping
	if r.URL.Path == "/ping" || r.URL.Path == "/api/ping" {
		next.ServeHTTP(w, r)
		return
	}

	// Set Kite-Region header for all responses
	if region != "" {
		w.Header().Set(regionKey, region)
	}

	// Check HMAC to bypass database hit. Fall back on session lookup
	if user, err := hmacutil.CheckRequest(r); err == nil {
		r = r.WithContext(context.WithValue(r.Context(), userKey, user))
		defer u.log(w, r, start)
		next.ServeHTTP(w, r)
		return
	}

	// Note: some requests legitimately will not have the session set. This is OK.
	// Worst-case, upstream will handle this. This allows us to avoid having to pick
	// which endpoints are protected vs not by auth at the user-mux layer.
	sessionKey, err := community.SessionKey(r)
	if err != nil {
		next.ServeHTTP(w, r)
		defer u.log(w, r, start)
		return
	}

	// Try to validate the session. If it doesn't validate, don't do anything here. Let
	// downstream user-node take care of this.
	user, session, err := u.users.ValidateSession(sessionKey)
	if err != nil {
		next.ServeHTTP(w, r)
		defer u.log(w, r, start)
		return
	}

	// Generate hmac headers, and apply to both the request and response so that downstream
	// request can avoid a 2nd database hit.
	headers := hmacutil.HeadersFromUserSession(user, session)
	for key, vals := range headers {
		r.Header.Set(key, vals[0])
		w.Header().Set(key, vals[0])
	}

	r = r.WithContext(context.WithValue(r.Context(), userKey, user))
	defer u.log(w, r, start)
	next.ServeHTTP(w, r)
}

func (u *userAuthMidware) log(w http.ResponseWriter, r *http.Request, start time.Time) {
	url := r.URL.Path
	if r.URL.RawQuery != "" {
		url += "?" + r.URL.Query().Encode()
	}

	userID := "no user"
	if user, ok := getUser(r); ok {
		userID = fmt.Sprintf("%d", user.ID)
	}

	requestIP := requestIP(r)
	userAgent := r.Header.Get("User-Agent")
	selectedSite := w.Header().Get(selectedSiteKey)

	switch rw := w.(type) {
	case negroni.ResponseWriter:
		// Log HTTP status and content size if this is a negroni.ResponseWriter
		logger.Info(requestIP, "->", selectedSite, fmt.Sprintf("(%s)", userID), r.Method, url, rw.Status(), rw.Size(), time.Now().Sub(start), userAgent)
	case http.ResponseWriter:
		// Log what we can for http.ResponseWriter
		logger.Info(requestIP, "->", selectedSite, fmt.Sprintf("(%s)", userID), r.Method, url, time.Now().Sub(start), userAgent)
	}
}

func requestIP(r *http.Request) string {
	if r == nil {
		return ""
	}

	remoteIP := r.Header.Get("X-Forwarded-For")

	// Intermediate proxies should be prepending their IP in this field,
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

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return ""
	}

	return host
}

func requireAuthHandler(w http.ResponseWriter, r *http.Request) {
	const (
		IDKey             = "ID"
		XAccelRedirectKey = "X-Accel-Redirect"
	)
	u, ok := getUser(r)
	if !ok {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	// if ID is passed in, the current user must have that ID
	if idStr := r.URL.Query().Get(IDKey); idStr != "" {
		if id, err := strconv.ParseInt(idStr, 10, 64); err != nil || u.ID != id {
			http.Error(w, "", http.StatusUnauthorized)
			return
		}
	}

	// If an XAccelRedirect URI was passed in, we set a corresponding
	// X-Accel-Redirect response header, with the ID query parameter added to the given URI.
	// This enables nginx-internal redirects: https://www.nginx.com/nginx-wiki/build/dirhtml/start/topics/examples/x-accel/.
	if redir := r.URL.Query().Get(XAccelRedirectKey); redir != "" {
		redirURL, err := url.Parse(redir)
		if err == nil {
			q := redirURL.Query()
			q.Set(IDKey, fmt.Sprintf("%d", u.ID))
			redirURL.RawQuery = q.Encode()

			// Also see https://github.com/slact/nchan#x-accel-redirect
			w.Header().Set(XAccelRedirectKey, redirURL.String())
			w.Header().Set("X-Accel-Buffering", "no")
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("%d", u.ID)))
}
