package trademap

import "fmt"

// APIError is returned when Trade Map responds with a non-2xx status.
// Body is capped and should be treated as diagnostic text, not a stable schema.
type APIError struct {
	StatusCode int
	Status     string
	Method     string
	URL        string
	Body       string
}

func (err *APIError) Error() string {
	if err.Body == "" {
		return fmt.Sprintf("trademap: %s %s: %s", err.Method, err.URL, err.Status)
	}
	return fmt.Sprintf("trademap: %s %s: %s: %s", err.Method, err.URL, err.Status, err.Body)
}
