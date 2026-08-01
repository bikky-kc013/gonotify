// Package common
package common

import (
	"errors"
	"net/http"
)

// Sentinel errors. Use these with errors.Is when you need to check the
// *kind* of failure without depending on a specific message or HTTP status.
var (
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrValidation         = errors.New("validation error")
	ErrNotFound           = errors.New("resource not found")
	ErrBadRequest         = errors.New("bad request")
	ErrConflict           = errors.New("resource conflict")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInternalServer     = errors.New("internal server error")
	ErrExpiredToken       = errors.New("expired token")
	ErrInvalidToken       = errors.New("invalid token")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrRateLimited        = errors.New("rate limit exceeded")
)

// Error codes returned to API clients.
const (
	// Auth
	ErrCodeUnauthorized       = "AUTH_UNAUTHORIZED"
	ErrCodeForbidden          = "AUTH_FORBIDDEN"
	ErrCodeInvalidToken       = "AUTH_INVALID_TOKEN"
	ErrCodeExpiredToken       = "AUTH_EXPIRED_TOKEN"
	ErrCodeInvalidCredentials = "AUTH_INVALID_CREDENTIALS"

	// Validation
	ErrCodeValidation = "VALIDATION_ERROR"
	ErrCodeBadRequest = "BAD_REQUEST"

	// Resource
	ErrCodeNotFound = "RESOURCE_NOT_FOUND"
	ErrCodeConflict = "RESOURCE_CONFLICT"

	// System
	ErrCodeInternal           = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	ErrCodeRateLimited        = "RATE_LIMITED"
	ErrRouteNotFound          = "ROUTE_NOT_FOUND"
)

// AppError is the application-level error type. It carries an HTTP status,
// and also shows the underlying error that caused it.
type AppError struct {
	Code      int    `json:"code"`
	ErrorCode string `json:"error_code,omitempty"`
	Message   string `json:"message"`
	Err       error  `json:"-"`
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap allows errors.Is / errors.As to see through AppError to the
// wrapped cause (e.g. a sentinel like ErrNotFound).
func (e *AppError) Unwrap() error {
	return e.Err
}

// Is lets errors.Is(someErr, common.ErrNotFound) succeed even when someErr
// is an *AppError wrapping ErrNotFound — same as default unwrap behavior,
// kept explicit here for clarity and in case matching logic grows later.
func (e *AppError) Is(target error) bool {
	return errors.Is(e.Err, target)
}

// New is the single constructor all others build on. Prefer the typed
// helpers below (NewNotFoundError, NewForbiddenError, ...) for common
// cases; use New/NewErrorWithCode directly for anything bespoke.
func New(httpStatus int, errorCode, message string, err error) *AppError {
	return &AppError{
		Code:      httpStatus,
		ErrorCode: errorCode,
		Message:   message,
		Err:       err,
	}
}

// NewErrorWithCode is an alias for New, kept for call-site readability
// when constructing an AppError with a custom HTTP status and error code.
func NewErrorWithCode(httpStatus int, errorCode, message string, err error) *AppError {
	return New(httpStatus, errorCode, message, err)
}

// Each wraps a sentinel by default so errors.Is(err, common.ErrX) works
// even if the caller doesn't pass an explicit err.

func NewNotFoundError(message string, err error) *AppError {
	return New(http.StatusNotFound, ErrCodeNotFound, message, orDefault(err, ErrNotFound))
}

func NewUnauthorizedError(message string) *AppError {
	return New(http.StatusUnauthorized, ErrCodeUnauthorized, message, ErrUnauthorized)
}

func NewBadRequestError(message string, err error) *AppError {
	return New(http.StatusBadRequest, ErrCodeBadRequest, message, orDefault(err, ErrBadRequest))
}

func NewInternalError(message string, err error) *AppError {
	return New(http.StatusInternalServerError, ErrCodeInternal, message, orDefault(err, ErrInternalServer))
}
func NewRouteNotFoundError(message string, err error) *AppError {
	return New(http.StatusNotFound, ErrRouteNotFound, message, err)
}

func NewInternalServerError(message string) *AppError {
	return New(http.StatusInternalServerError, ErrCodeInternal, message, ErrInternalServer)
}

func NewConflictError(message string) *AppError {
	return New(http.StatusConflict, ErrCodeConflict, message, ErrConflict)
}

func NewValidationError(message string) *AppError {
	return New(http.StatusBadRequest, ErrCodeValidation, message, ErrValidation)
}

func NewServiceUnavailableError(message string) *AppError {
	return New(http.StatusServiceUnavailable, ErrCodeServiceUnavailable, message, ErrServiceUnavailable)
}

func NewTooManyRequestsError(message string) *AppError {
	return New(http.StatusTooManyRequests, ErrCodeRateLimited, message, ErrRateLimited)
}

func NewForbiddenError(message string) *AppError {
	return New(http.StatusForbidden, ErrCodeForbidden, message, ErrForbidden)
}

// orDefault returns err if non-nil, otherwise fallback. Used so callers can
// pass their own wrapped cause (e.g. a DB error) while still getting a
// sensible sentinel when they only have a message.
func orDefault(err, fallback error) error {
	if err != nil {
		return err
	}
	return fallback
}
