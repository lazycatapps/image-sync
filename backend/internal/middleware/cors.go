// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package middleware provides HTTP middleware for the Image Sync server.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS creates a Cross-Origin Resource Sharing (CORS) middleware.
// It configures allowed origins, methods, and headers for cross-origin requests.
//
// Supported origins:
//   - "*": Allow all origins (reflects the request origin to support credentials)
//   - Specific origins: Only allow exact matches
//
// Allowed methods: GET, POST, PUT, DELETE, OPTIONS
// Allowed headers: Content-Type, Authorization
//
// Note: When using credentials (cookies, authorization headers), the wildcard "*"
// is not allowed by CORS spec. This middleware automatically reflects the request
// origin when wildcard is configured, allowing credentials to work properly.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if request origin is in allowed list
		allowed := false
		allowCredentials := false
		for _, allowedOrigin := range allowedOrigins {
			if allowedOrigin == "*" {
				// Wildcard: allow all origins by reflecting the request origin
				// This is required when using credentials (CORS spec forbids "*" with credentials)
				allowed = true
				if origin != "" {
					c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
					allowCredentials = true
				} else {
					// No origin header (e.g., same-origin request or tools like curl)
					c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
					allowCredentials = false
				}
				break
			} else if allowedOrigin == origin {
				// Exact match: allow this specific origin
				allowed = true
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				allowCredentials = true
				break
			}
		}

		// Only set CORS headers if origin is allowed
		if allowed {
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if allowCredentials {
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}

		// Handle preflight OPTIONS requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// CORSWithOrigins creates a CORS middleware from a comma-separated origins string.
// Empty or whitespace-only origins default to wildcard "*".
func CORSWithOrigins(originsCSV string) gin.HandlerFunc {
	var origins []string
	if originsCSV == "" {
		origins = []string{"*"}
	} else {
		parts := strings.Split(originsCSV, ",")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				origins = append(origins, trimmed)
			}
		}
		if len(origins) == 0 {
			origins = []string{"*"}
		}
	}
	return CORS(origins)
}
