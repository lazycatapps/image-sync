// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package middleware provides HTTP middleware functions.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SessionValidator is an interface for validating sessions.
type SessionValidator interface {
	GetSession(sessionID string) (interface{}, bool)
}

// Auth is a middleware that validates OIDC authentication.
// It checks for a valid session cookie and redirects to login if not authenticated.
func Auth(oidcEnabled bool, sessionValidator SessionValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication if OIDC is not enabled
		if !oidcEnabled {
			c.Next()
			return
		}

		// Skip authentication for public endpoints
		if isPublicEndpoint(c.FullPath()) {
			c.Next()
			return
		}

		// Check for session cookie
		sessionCookie, err := c.Cookie("session")
		if err != nil || sessionCookie == "" {
			// No session, redirect to login
			if isAPIRequest(c) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
				c.Abort()
				return
			}
			// For browser requests, redirect to login
			c.Redirect(http.StatusFound, "/api/v1/auth/login")
			c.Abort()
			return
		}

		// Validate session
		if sessionValidator != nil {
			sessionInfo, exists := sessionValidator.GetSession(sessionCookie)
			if !exists {
				if isAPIRequest(c) {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired session"})
					c.Abort()
					return
				}
				c.Redirect(http.StatusFound, "/api/v1/auth/login")
				c.Abort()
				return
			}
			// Store session info in context for handlers to use
			c.Set("session", sessionInfo)
		}

		c.Next()
	}
}

// isPublicEndpoint checks if the endpoint is public (no auth required).
func isPublicEndpoint(path string) bool {
	publicPaths := []string{
		"/api/v1/health",
		"/api/v1/auth/login",
		"/api/v1/auth/callback",
		"/api/v1/auth/userinfo",
	}

	for _, p := range publicPaths {
		if path == p {
			return true
		}
	}
	return false
}

// isAPIRequest checks if the request is an API request based on headers
func isAPIRequest(c *gin.Context) bool {
	// Check Accept header
	accept := c.GetHeader("Accept")
	if strings.Contains(accept, "application/json") {
		return true
	}

	// Check if it's an XHR request
	if c.GetHeader("X-Requested-With") == "XMLHttpRequest" {
		return true
	}

	return false
}
