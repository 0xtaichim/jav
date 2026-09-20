package javdb

// LoginRequiredError indicates that the operation requires a logged-in session
// (e.g. JAVDB_COOKIES). Callers should return HTTP 401 with this message.
type LoginRequiredError struct {
	Message string
}

func (e *LoginRequiredError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "Unauthorized"
}
