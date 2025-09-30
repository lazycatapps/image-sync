// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package validator provides input validation utilities for security.
package validator

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

const (
	// Maximum input lengths to prevent DoS
	MaxImageNameLength    = 512
	MaxUsernameLength     = 256
	MaxPasswordLength     = 512
	MaxArchitectureLength = 64
	MaxConfigNameLength   = 64
)

// Image name validation regex patterns
var (
	// Valid image name format: [registry/][namespace/]repository[:tag|@digest]
	// Examples:
	//   - nginx:latest
	//   - docker.io/library/nginx:1.21
	//   - registry.example.com:5000/myapp/nginx@sha256:abc123...
	imageNameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?(:[0-9]+)?(/[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?)*(@sha256:[a-fA-F0-9]{64}|:[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?)?$`)

	// Valid architecture format: os/arch or os/arch/variant
	// Examples: linux/amd64, linux/arm/v7
	architectureRegex = regexp.MustCompile(`^[a-z0-9]+/[a-z0-9]+(/[a-z0-9]+)?$`)

	// Valid config name format: alphanumeric, dash, underscore, dot
	// Examples: default, prod-env, my.config, dev_1
	configNameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
)

// ValidationError represents an input validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
}

// ValidateImageName validates a container image name.
// It checks format, length, and ensures no malicious characters.
func ValidateImageName(image string) error {
	if image == "" {
		return &ValidationError{
			Field:   "image",
			Message: "image name cannot be empty",
		}
	}

	if len(image) > MaxImageNameLength {
		return &ValidationError{
			Field:   "image",
			Message: fmt.Sprintf("image name exceeds maximum length of %d characters", MaxImageNameLength),
		}
	}

	// Check for shell metacharacters that could be dangerous
	dangerousChars := []string{";", "&", "|", "`", "$", "(", ")", "<", ">", "\n", "\r", "\\"}
	for _, char := range dangerousChars {
		if strings.Contains(image, char) {
			return &ValidationError{
				Field:   "image",
				Message: fmt.Sprintf("image name contains invalid character: %s", char),
			}
		}
	}

	// Validate against regex pattern
	if !imageNameRegex.MatchString(image) {
		return &ValidationError{
			Field:   "image",
			Message: "image name format is invalid",
		}
	}

	return nil
}

// ValidateArchitecture validates an architecture string.
// Accepts "all" or format like "linux/amd64" or "linux/arm/v7".
func ValidateArchitecture(arch string) error {
	if arch == "" || arch == "all" {
		return nil
	}

	if len(arch) > MaxArchitectureLength {
		return &ValidationError{
			Field:   "architecture",
			Message: fmt.Sprintf("architecture exceeds maximum length of %d characters", MaxArchitectureLength),
		}
	}

	if !architectureRegex.MatchString(arch) {
		return &ValidationError{
			Field:   "architecture",
			Message: "architecture format is invalid (expected: os/arch or os/arch/variant)",
		}
	}

	return nil
}

// ValidateUsername validates a registry username.
// Allows alphanumeric, dash, underscore, dot, and @ symbol.
func ValidateUsername(username string) error {
	if username == "" {
		return nil // Username is optional
	}

	if len(username) > MaxUsernameLength {
		return &ValidationError{
			Field:   "username",
			Message: fmt.Sprintf("username exceeds maximum length of %d characters", MaxUsernameLength),
		}
	}

	// Allow common username characters: alphanumeric, dash, underscore, dot, @
	for _, r := range username {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' && r != '.' && r != '@' {
			return &ValidationError{
				Field:   "username",
				Message: fmt.Sprintf("username contains invalid character: %c", r),
			}
		}
	}

	return nil
}

// ValidatePassword validates a registry password.
// Checks length only, allows most characters except control characters and shell metacharacters.
func ValidatePassword(password string) error {
	if password == "" {
		return nil // Password is optional
	}

	if len(password) > MaxPasswordLength {
		return &ValidationError{
			Field:   "password",
			Message: fmt.Sprintf("password exceeds maximum length of %d characters", MaxPasswordLength),
		}
	}

	// Reject only the most dangerous characters for command injection
	// Note: Since we use exec.Command() args (not shell), we only need to block a minimal set
	dangerousChars := []string{"\n", "\r", "\x00"}
	for _, char := range dangerousChars {
		if strings.Contains(password, char) {
			return &ValidationError{
				Field:   "password",
				Message: "password contains invalid character",
			}
		}
	}

	return nil
}

// ValidateCredentials validates both username and password together.
// If one is provided, both must be provided.
func ValidateCredentials(username, password string) error {
	if (username != "" && password == "") || (username == "" && password != "") {
		return &ValidationError{
			Field:   "credentials",
			Message: "both username and password must be provided together",
		}
	}

	if err := ValidateUsername(username); err != nil {
		return err
	}

	if err := ValidatePassword(password); err != nil {
		return err
	}

	return nil
}

// ValidateConfigName validates a configuration name.
// Allows alphanumeric characters, dash, underscore, and dot.
// Prevents path traversal attacks.
func ValidateConfigName(name string) error {
	if name == "" {
		return &ValidationError{
			Field:   "configName",
			Message: "config name cannot be empty",
		}
	}

	if len(name) > MaxConfigNameLength {
		return &ValidationError{
			Field:   "configName",
			Message: fmt.Sprintf("config name exceeds maximum length of %d characters", MaxConfigNameLength),
		}
	}

	// Reject path traversal attempts
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		return &ValidationError{
			Field:   "configName",
			Message: "config name cannot contain path separators or '..'",
		}
	}

	// Validate against regex pattern
	if !configNameRegex.MatchString(name) {
		return &ValidationError{
			Field:   "configName",
			Message: "config name can only contain letters, numbers, dash, underscore, and dot",
		}
	}

	return nil
}
