// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	err := New("TEST_ERROR", "Test error message", http.StatusBadRequest)
	expected := "Test error message"

	if err.Error() != expected {
		t.Errorf("Expected error message %s, got %s", expected, err.Error())
	}
}

func TestAppError_ErrorWithWrapped(t *testing.T) {
	originalErr := errors.New("original error")
	err := Wrap(originalErr, "TEST_ERROR", "Test error message", http.StatusBadRequest)
	expected := "Test error message: original error"

	if err.Error() != expected {
		t.Errorf("Expected error message %s, got %s", expected, err.Error())
	}
}

func TestAppError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	err := Wrap(originalErr, "TEST_ERROR", "Test error message", http.StatusBadRequest)

	unwrapped := err.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("Expected unwrapped error to be original error")
	}
}

func TestPredefinedErrors(t *testing.T) {
	testCases := []struct {
		name           string
		err            *AppError
		expectedCode   string
		expectedStatus int
	}{
		{
			name:           "ErrTaskNotFound",
			err:            ErrTaskNotFound,
			expectedCode:   "TASK_NOT_FOUND",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "ErrInvalidInput",
			err:            ErrInvalidInput,
			expectedCode:   "INVALID_INPUT",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "ErrInternal",
			err:            ErrInternal,
			expectedCode:   "INTERNAL_ERROR",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "ErrCommandFailed",
			err:            ErrCommandFailed,
			expectedCode:   "COMMAND_FAILED",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err.Code != tc.expectedCode {
				t.Errorf("Expected code %s, got %s", tc.expectedCode, tc.err.Code)
			}

			if tc.err.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, tc.err.StatusCode)
			}
		})
	}
}

func TestWrappers(t *testing.T) {
	originalErr := errors.New("test error")

	testCases := []struct {
		name           string
		wrapper        func(error) *AppError
		expectedCode   string
		expectedStatus int
	}{
		{
			name:           "WrapTaskNotFound",
			wrapper:        WrapTaskNotFound,
			expectedCode:   "TASK_NOT_FOUND",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.wrapper(originalErr)

			if err.Code != tc.expectedCode {
				t.Errorf("Expected code %s, got %s", tc.expectedCode, err.Code)
			}

			if err.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, err.StatusCode)
			}

			if !errors.Is(err, originalErr) {
				t.Error("Expected wrapped error to be the original error")
			}
		})
	}
}

func TestWrapInvalidInput(t *testing.T) {
	originalErr := errors.New("test error")
	message := "Custom error message"

	err := WrapInvalidInput(originalErr, message)

	if err.Code != "INVALID_INPUT" {
		t.Errorf("Expected code INVALID_INPUT, got %s", err.Code)
	}

	if err.Message != message {
		t.Errorf("Expected message %s, got %s", message, err.Message)
	}

	if err.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, err.StatusCode)
	}
}

func TestWrapInternal(t *testing.T) {
	originalErr := errors.New("test error")
	message := "Custom error message"

	err := WrapInternal(originalErr, message)

	if err.Code != "INTERNAL_ERROR" {
		t.Errorf("Expected code INTERNAL_ERROR, got %s", err.Code)
	}

	if err.Message != message {
		t.Errorf("Expected message %s, got %s", message, err.Message)
	}

	if err.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, err.StatusCode)
	}
}

func TestWrapCommandFailed(t *testing.T) {
	originalErr := errors.New("test error")
	message := "Custom error message"

	err := WrapCommandFailed(originalErr, message)

	if err.Code != "COMMAND_FAILED" {
		t.Errorf("Expected code COMMAND_FAILED, got %s", err.Code)
	}

	if err.Message != message {
		t.Errorf("Expected message %s, got %s", message, err.Message)
	}

	if err.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, err.StatusCode)
	}
}
