// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

package validator

import (
	"strings"
	"testing"
)

func TestValidateImageName(t *testing.T) {
	tests := []struct {
		name      string
		imageName string
		wantErr   bool
	}{
		// Valid cases
		{"valid simple name", "nginx", false},
		{"valid with tag", "nginx:latest", false},
		{"valid with registry", "docker.io/library/nginx", false},
		{"valid with port", "registry.example.com:5000/nginx", false},
		{"valid full name", "docker.io/library/nginx:1.21.0", false},
		{"valid with digest", "nginx@sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", false},
		{"valid with namespace", "mycompany/myapp/nginx:v1", false},
		{"valid with dashes", "my-registry.com/my-namespace/my-app:my-tag", false},
		{"valid with underscores", "my_registry/my_app:v1_2_3", false},
		{"valid with dots", "registry.example.com/app.name:1.0.0", false},

		// Invalid cases - security
		{"with semicolon", "nginx; rm -rf /", true},
		{"with pipe", "nginx | cat /etc/passwd", true},
		{"with ampersand", "nginx && echo hack", true},
		{"with backtick", "nginx`whoami`", true},
		{"with dollar", "nginx$PATH", true},
		{"with redirect", "nginx > /tmp/hack", true},
		{"with newline", "nginx\nrm -rf /", true},
		{"with backslash", "nginx\\x00", true},

		// Invalid cases - format
		{"empty string", "", true},
		{"too long", strings.Repeat("a", 513), true},
		{"invalid start", "/nginx", true},
		{"invalid end", "nginx/", true},
		{"double slash", "nginx//latest", true},

		// Edge cases
		{"max length", strings.Repeat("a", 512), false},
		{"single char", "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateImageName(tt.imageName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateImageName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateArchitecture(t *testing.T) {
	tests := []struct {
		name    string
		arch    string
		wantErr bool
	}{
		// Valid cases
		{"empty (all)", "", false},
		{"all keyword", "all", false},
		{"linux/amd64", "linux/amd64", false},
		{"linux/arm64", "linux/arm64", false},
		{"linux/arm/v7", "linux/arm/v7", false},
		{"linux/386", "linux/386", false},
		{"windows/amd64", "windows/amd64", false},

		// Invalid cases
		{"only os", "linux", true},
		{"too many parts", "linux/arm/v7/extra", true},
		{"with uppercase", "Linux/AMD64", true},
		{"with spaces", "linux / amd64", true},
		{"with special char", "linux/amd64;", true},
		{"too long", strings.Repeat("a", 65), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateArchitecture(tt.arch)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateArchitecture() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		// Valid cases
		{"empty (optional)", "", false},
		{"simple", "user", false},
		{"with numbers", "user123", false},
		{"with dash", "user-name", false},
		{"with underscore", "user_name", false},
		{"with dot", "user.name", false},
		{"with at", "user@example.com", false},
		{"mixed", "user-123_test.name@example", false},

		// Invalid cases
		{"with space", "user name", true},
		{"with semicolon", "user;admin", true},
		{"with pipe", "user|admin", true},
		{"too long", strings.Repeat("a", 257), true},
		{"with special chars", "user#name", true},
		{"with slash", "user/admin", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUsername(tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUsername() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		// Valid cases
		{"empty (optional)", "", false},
		{"simple", "password123", false},
		{"with special chars", "P@ssw0rd!", false},
		{"with spaces", "my password", false},
		{"complex", "P@ssw0rd!#$%^&*()_+-=[]{}:\"'<>?,./", false},
		{"with semicolon", "pass;word", false},
		{"with pipe", "pass|word", false},
		{"with ampersand", "pass&word", false},
		{"with backtick", "pass`word", false},
		{"with dollar", "pass$word", false},

		// Invalid cases
		{"with newline", "pass\nword", true},
		{"with carriage return", "pass\rword", true},
		{"with null byte", "pass\x00word", true},
		{"too long", strings.Repeat("a", 513), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCredentials(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		wantErr  bool
	}{
		// Valid cases
		{"both empty", "", "", false},
		{"both provided", "user", "pass", false},
		{"valid complex", "user123", "P@ssw0rd!", false},

		// Invalid cases
		{"username only", "user", "", true},
		{"password only", "", "pass", true},
		{"invalid username", "user;admin", "pass", true},
		{"invalid password", "user", "pass\nword", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCredentials(tt.username, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		Field:   "testField",
		Message: "test message",
	}

	expected := "validation error for field 'testField': test message"
	if err.Error() != expected {
		t.Errorf("ValidationError.Error() = %v, want %v", err.Error(), expected)
	}
}
