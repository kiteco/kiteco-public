package auth

import "errors"

// ErrNotAuthenticated indicates a request without authentication
var ErrNotAuthenticated = errors.New("not authenticated")
