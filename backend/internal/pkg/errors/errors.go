// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package errors provides unified error handling for the Image Sync application.
package errors

import (
	"fmt"
	"net/http"
)

// AppError represents an application error with HTTP status code and error code.
// It implements the error interface and supports error wrapping (Go 1.13+).
type AppError struct {
	Code       string `json:"code"`       // Error code (e.g., "TASK_NOT_FOUND")
	Message    string `json:"message"`    // Human-readable error message
	StatusCode int    `json:"-"`          // HTTP status code (not serialized)
	Err        error  `json:"-"`          // Wrapped error (not serialized)
}

// Error returns the error message string.
// Implements the error interface.
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the wrapped error.
// Enables Go 1.13+ error unwrapping with errors.Is() and errors.As().
func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new AppError without wrapping an existing error.
func New(code, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// Wrap creates a new AppError that wraps an existing error.
func Wrap(err error, code, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Err:        err,
	}
}

// Predefined error instances for common error scenarios.
var (
	ErrTaskNotFound  = New("TASK_NOT_FOUND", "Task not found", http.StatusNotFound)
	ErrInvalidInput  = New("INVALID_INPUT", "Invalid input parameters", http.StatusBadRequest)
	ErrInternal      = New("INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError)
	ErrCommandFailed = New("COMMAND_FAILED", "Command execution failed", http.StatusInternalServerError)
)

// WrapTaskNotFound wraps an error as a task not found error (404).
func WrapTaskNotFound(err error) *AppError {
	return Wrap(err, "TASK_NOT_FOUND", "Task not found", http.StatusNotFound)
}

// NewInvalidInput creates a new invalid input error (400) without wrapping.
func NewInvalidInput(message string) *AppError {
	return New("INVALID_INPUT", message, http.StatusBadRequest)
}

// WrapInvalidInput wraps an error as an invalid input error (400).
func WrapInvalidInput(err error, message string) *AppError {
	return Wrap(err, "INVALID_INPUT", message, http.StatusBadRequest)
}

// WrapInternal wraps an error as an internal server error (500).
func WrapInternal(err error, message string) *AppError {
	return Wrap(err, "INTERNAL_ERROR", message, http.StatusInternalServerError)
}

// WrapCommandFailed wraps an error as a command execution failure (500).
func WrapCommandFailed(err error, message string) *AppError {
	return Wrap(err, "COMMAND_FAILED", message, http.StatusInternalServerError)
}