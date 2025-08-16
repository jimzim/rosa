// Package errors provides structured error handling for the ROSA CLI
package errors

import (
	"errors"
	"fmt"
	"strings"
)

// Kind represents the category of error
type Kind int

const (
	// KindValidation indicates a validation error
	KindValidation Kind = iota
	// KindAPI indicates an API communication error
	KindAPI
	// KindAWS indicates an AWS service error
	KindAWS
	// KindPermission indicates a permissions/authorization error
	KindPermission
	// KindNetwork indicates a network connectivity error
	KindNetwork
	// KindNotFound indicates a resource was not found
	KindNotFound
	// KindConflict indicates a resource conflict
	KindConflict
	// KindTimeout indicates an operation timed out
	KindTimeout
	// KindInternal indicates an internal error
	KindInternal
)

// String returns the string representation of the error kind
func (k Kind) String() string {
	switch k {
	case KindValidation:
		return "Validation Error"
	case KindAPI:
		return "API Error"
	case KindAWS:
		return "AWS Error"
	case KindPermission:
		return "Permission Error"
	case KindNetwork:
		return "Network Error"
	case KindNotFound:
		return "Not Found"
	case KindConflict:
		return "Conflict"
	case KindTimeout:
		return "Timeout"
	case KindInternal:
		return "Internal Error"
	default:
		return "Unknown Error"
	}
}

// ROSAError represents a structured error with context
type ROSAError struct {
	Op         string                 // Operation being performed
	Kind       Kind                   // Error category
	Err        error                  // Underlying error
	Suggestion string                 // User-actionable suggestion
	Details    map[string]interface{} // Additional context
}

// Error implements the error interface
func (e *ROSAError) Error() string {
	var b strings.Builder

	// Start with the operation if provided
	if e.Op != "" {
		b.WriteString(e.Op)
		b.WriteString(": ")
	}

	// Add the kind
	b.WriteString(e.Kind.String())

	// Add the underlying error if present
	if e.Err != nil {
		b.WriteString(": ")
		b.WriteString(e.Err.Error())
	}

	// Add suggestion if present
	if e.Suggestion != "" {
		b.WriteString("\n💡 Suggestion: ")
		b.WriteString(e.Suggestion)
	}

	return b.String()
}

// Unwrap returns the underlying error
func (e *ROSAError) Unwrap() error {
	return e.Err
}

// Is implements errors.Is
func (e *ROSAError) Is(target error) bool {
	t, ok := target.(*ROSAError)
	if !ok {
		return false
	}
	return e.Kind == t.Kind
}

// WithSuggestion adds a suggestion to the error
func (e *ROSAError) WithSuggestion(suggestion string) *ROSAError {
	e.Suggestion = suggestion
	return e
}

// WithDetails adds additional context to the error
func (e *ROSAError) WithDetails(details map[string]interface{}) *ROSAError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	for k, v := range details {
		e.Details[k] = v
	}
	return e
}

// WithDetail adds a single detail to the error
func (e *ROSAError) WithDetail(key string, value interface{}) *ROSAError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// New creates a new ROSAError
func New(kind Kind, op string, err error) *ROSAError {
	return &ROSAError{
		Op:   op,
		Kind: kind,
		Err:  err,
	}
}

// Validation creates a validation error
func Validation(op string, err error) *ROSAError {
	return New(KindValidation, op, err)
}

// API creates an API error
func API(op string, err error) *ROSAError {
	return New(KindAPI, op, err)
}

// AWS creates an AWS error
func AWS(op string, err error) *ROSAError {
	return New(KindAWS, op, err)
}

// Permission creates a permission error
func Permission(op string, err error) *ROSAError {
	return New(KindPermission, op, err)
}

// Network creates a network error
func Network(op string, err error) *ROSAError {
	return New(KindNetwork, op, err)
}

// NotFound creates a not found error
func NotFound(op string, resource string) *ROSAError {
	return New(KindNotFound, op, fmt.Errorf("%s not found", resource))
}

// Conflict creates a conflict error
func Conflict(op string, err error) *ROSAError {
	return New(KindConflict, op, err)
}

// Timeout creates a timeout error
func Timeout(op string, err error) *ROSAError {
	return New(KindTimeout, op, err)
}

// Internal creates an internal error
func Internal(op string, err error) *ROSAError {
	return New(KindInternal, op, err)
}

// IsKind checks if an error is of a specific kind
func IsKind(err error, kind Kind) bool {
	var e *ROSAError
	if errors.As(err, &e) {
		return e.Kind == kind
	}
	return false
}

// IsValidation checks if an error is a validation error
func IsValidation(err error) bool {
	return IsKind(err, KindValidation)
}

// IsNotFound checks if an error is a not found error
func IsNotFound(err error) bool {
	return IsKind(err, KindNotFound)
}

// IsPermission checks if an error is a permission error
func IsPermission(err error) bool {
	return IsKind(err, KindPermission)
}

// Result represents an operation that can fail
type Result[T any] struct {
	value T
	err   error
}

// Ok creates a successful result
func Ok[T any](value T) Result[T] {
	return Result[T]{value: value}
}

// Err creates a failed result
func Err[T any](err error) Result[T] {
	return Result[T]{err: err}
}

// Unwrap returns the value and error
func (r Result[T]) Unwrap() (T, error) {
	return r.value, r.err
}

// IsOk checks if the result is successful
func (r Result[T]) IsOk() bool {
	return r.err == nil
}

// IsErr checks if the result is an error
func (r Result[T]) IsErr() bool {
	return r.err != nil
}

// Value returns the value, panicking if there's an error
func (r Result[T]) Value() T {
	if r.err != nil {
		panic(fmt.Sprintf("called Value() on error result: %v", r.err))
	}
	return r.value
}

// Error returns the error, panicking if there isn't one
func (r Result[T]) Error() error {
	if r.err == nil {
		panic("called Error() on ok result")
	}
	return r.err
}

// Map transforms the value if the result is successful
func (r Result[T]) Map(fn func(T) T) Result[T] {
	if r.err != nil {
		return r
	}
	return Ok(fn(r.value))
}

// MapErr transforms the error if the result is a failure
func (r Result[T]) MapErr(fn func(error) error) Result[T] {
	if r.err == nil {
		return r
	}
	return Err[T](fn(r.err))
}

// Then chains another operation if the result is successful
func (r Result[T]) Then(fn func(T) Result[T]) Result[T] {
	if r.err != nil {
		return r
	}
	return fn(r.value)
}
