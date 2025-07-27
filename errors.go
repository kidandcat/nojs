package nojs

import "fmt"

// HTTPError represents an HTTP error with status code
type HTTPError struct {
	Code    int
	Message string
	Err     error
}

// Error implements the error interface
func (e *HTTPError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("HTTP %d: %s - %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("HTTP %d: %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error
func (e *HTTPError) Unwrap() error {
	return e.Err
}

// NewHTTPError creates a new HTTP error
func NewHTTPError(code int, message string) *HTTPError {
	return &HTTPError{
		Code:    code,
		Message: message,
	}
}

// WrapHTTPError wraps an error with HTTP error information
func WrapHTTPError(code int, message string, err error) *HTTPError {
	return &HTTPError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}