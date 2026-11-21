package monarchbutterflyicloud

import "fmt"

// Code identifies why the package refused a token or an iCloud response.
type Code string

const (
	// CodeInvalidClient means NewClient was given a nil *http.Client or one
	// without a positive Timeout.
	CodeInvalidClient Code = "invalid_client"
	// CodeInvalidToken means the album token is too short or contains a
	// character outside [0-9A-Za-z]. Nothing was sent.
	CodeInvalidToken Code = "invalid_token"
	// CodeUnexpectedStatus means iCloud answered with a status other than
	// 2xx, or other than 330 on the first /webstream request. Error.Status
	// holds it.
	CodeUnexpectedStatus Code = "unexpected_status"
	// CodeBadRedirect means a 330 carried no usable host, or a second 330
	// arrived after the album was already redirected once.
	CodeBadRedirect Code = "bad_redirect"
	// CodeMalformedResponse means a response body was not the JSON shape
	// iCloud normally sends, or a field in it did not parse.
	CodeMalformedResponse Code = "malformed_response"
)

// Error is returned for every refusal this package makes. Transport errors
// from the caller's *http.Client, including context cancellation, are
// returned unchanged instead.
type Error struct {
	Code Code
	// Op is the endpoint involved ("webstream", "webasseturls"), or "token"
	// or "NewClient".
	Op string
	// Status is the HTTP status for CodeUnexpectedStatus and CodeBadRedirect,
	// and 0 otherwise.
	Status int
	// Detail names the offending field or value.
	Detail string
	// Err is the underlying decode or parse error, if there was one.
	Err error
}

func (e *Error) Error() string {
	msg := fmt.Sprintf("monarchbutterflyicloud: %s: %s", e.Op, e.Code)
	if e.Status != 0 {
		msg += fmt.Sprintf(" (HTTP %d)", e.Status)
	}
	if e.Detail != "" {
		msg += ": " + e.Detail
	}
	if e.Err != nil {
		msg += ": " + e.Err.Error()
	}
	return msg
}

func (e *Error) Unwrap() error { return e.Err }
