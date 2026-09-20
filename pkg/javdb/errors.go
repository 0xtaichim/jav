package javdb

import (
	"errors"
	"fmt"
	"strings"
)

// ErrUnauthorized is returned when JavDB requires a logged-in session.
var ErrUnauthorized = errors.New("unauthorized")

// LoginRequiredError indicates that the operation requires a logged-in session
// (e.g. JAVDB_COOKIES). Callers should surface this as HTTP 401.
type LoginRequiredError struct {
	Message string
}

func (e *LoginRequiredError) Error() string {
	if e != nil && e.Message != "" {
		return e.Message
	}
	return "Unauthorized"
}

func (e *LoginRequiredError) Unwrap() error {
	return ErrUnauthorized
}

// HTTPError is a non-OK HTTP response from JavDB.
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	if e == nil {
		return "http error"
	}
	if strings.TrimSpace(e.Body) == "" {
		return fmt.Sprintf("HTTP status %d", e.StatusCode)
	}
	return fmt.Sprintf("HTTP status %d: %s", e.StatusCode, strings.TrimSpace(e.Body))
}
