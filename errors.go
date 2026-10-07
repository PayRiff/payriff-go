package payriff

import (
	"errors"
	"fmt"
)

type ErrorKind int

const (
	KindAPI ErrorKind = iota
	KindAuthentication
	KindValidation
	KindRequestRejected
	KindInsufficientBalance
	KindPayoutLimit
	KindConnection
)

func (k ErrorKind) String() string {
	switch k {
	case KindAuthentication:
		return "authentication"
	case KindValidation:
		return "validation"
	case KindRequestRejected:
		return "request rejected"
	case KindInsufficientBalance:
		return "insufficient balance"
	case KindPayoutLimit:
		return "payout limit"
	case KindConnection:
		return "connection"
	default:
		return "api"
	}
}

var (
	ErrAuthentication      = errors.New("payriff: authentication failed")
	ErrValidation          = errors.New("payriff: validation failed")
	ErrRequestRejected     = errors.New("payriff: request rejected")
	ErrInsufficientBalance = errors.New("payriff: insufficient balance")
	ErrPayoutLimit         = errors.New("payriff: payout limit reached")
	ErrConnection          = errors.New("payriff: connection failed")
	ErrInvalidArgument     = errors.New("payriff: invalid argument")
)

var kindSentinels = map[ErrorKind]error{
	KindAuthentication:      ErrAuthentication,
	KindValidation:          ErrValidation,
	KindRequestRejected:     ErrRequestRejected,
	KindInsufficientBalance: ErrInsufficientBalance,
	KindPayoutLimit:         ErrPayoutLimit,
	KindConnection:          ErrConnection,
}

type Error struct {
	Kind       ErrorKind
	HTTPStatus int
	Code       string
	ResponseID string
	Message    string
	Err        error
}

func (e *Error) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("payriff: %s (code %s, HTTP %d)", e.Message, e.Code, e.HTTPStatus)
	}
	if e.HTTPStatus != 0 {
		return fmt.Sprintf("payriff: %s (HTTP %d)", e.Message, e.HTTPStatus)
	}
	return "payriff: " + e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

func (e *Error) Is(target error) bool {
	sentinel, ok := kindSentinels[e.Kind]
	return ok && sentinel == target
}

const successCode = "00000"

func newAPIError(message string, httpStatus int, code, responseID string) *Error {
	if message == "" {
		message = fmt.Sprintf("Payriff request failed (HTTP %d)", httpStatus)
	}
	return &Error{Kind: kindFor(httpStatus, code), HTTPStatus: httpStatus, Code: code, ResponseID: responseID, Message: message}
}

func kindFor(httpStatus int, code string) ErrorKind {
	switch code {
	case "14010", "14013", "14014", "14015":
		return KindAuthentication
	case "01300", "01400", "01500":
		return KindPayoutLimit
	case "01200":
		return KindInsufficientBalance
	case "01000":
		return KindRequestRejected
	case "15400":
		return KindValidation
	}
	switch httpStatus {
	case 401, 403:
		return KindAuthentication
	case 400:
		return KindValidation
	}
	return KindAPI
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidArgument, fmt.Sprintf(format, args...))
}
