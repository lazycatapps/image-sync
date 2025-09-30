// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package handler provides HTTP request handlers.
package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/lazycatapps/image-sync/internal/pkg/logger"
	"github.com/lazycatapps/image-sync/internal/service"
	"github.com/lazycatapps/image-sync/internal/types"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

// AuthHandler handles OIDC authentication requests.
type AuthHandler struct {
	config         *types.OIDCConfig
	sessionService *service.SessionService
	provider       *oidc.Provider
	oauth2Config   *oauth2.Config
	log            logger.Logger
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(cfg *types.OIDCConfig, sessionService *service.SessionService, log logger.Logger) (*AuthHandler, error) {
	// If OIDC is not enabled, return handler without initialization
	if !cfg.Enabled {
		return &AuthHandler{
			config:         cfg,
			sessionService: sessionService,
			log:            log,
		}, nil
	}

	// Initialize OIDC provider
	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, err
	}

	// Configure OAuth2
	oauth2Config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email", "groups"},
	}

	return &AuthHandler{
		config:         cfg,
		sessionService: sessionService,
		provider:       provider,
		oauth2Config:   oauth2Config,
		log:            log,
	}, nil
}

// Login redirects to OIDC provider for authentication.
func (h *AuthHandler) Login(c *gin.Context) {
	if !h.config.Enabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OIDC authentication is not enabled"})
		return
	}

	// Generate random state
	state, err := generateState()
	if err != nil {
		h.log.Error("Failed to generate state: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate state"})
		return
	}

	// Store state in cookie for verification
	c.SetCookie("oauth_state", state, 600, "/", "", true, true)

	// Redirect to OIDC provider
	authURL := h.oauth2Config.AuthCodeURL(state)
	c.Redirect(http.StatusFound, authURL)
}

// Callback handles the OIDC callback.
func (h *AuthHandler) Callback(c *gin.Context) {
	if !h.config.Enabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "OIDC authentication is not enabled"})
		return
	}

	// Verify state
	stateCookie, err := c.Cookie("oauth_state")
	if err != nil {
		h.log.Error("Missing state cookie: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing state"})
		return
	}

	state := c.Query("state")
	if state != stateCookie {
		h.log.Error("State mismatch: expected %s, got %s", stateCookie, state)
		c.JSON(http.StatusBadRequest, gin.H{"error": "State mismatch"})
		return
	}

	// Clear state cookie
	c.SetCookie("oauth_state", "", -1, "/", "", true, true)

	// Exchange code for token
	code := c.Query("code")
	ctx := context.Background()
	oauth2Token, err := h.oauth2Config.Exchange(ctx, code)
	if err != nil {
		h.log.Error("Failed to exchange token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token"})
		return
	}

	// Extract ID token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		h.log.Error("No id_token in token response")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "No id_token"})
		return
	}

	// Verify ID token
	verifier := h.provider.Verifier(&oidc.Config{ClientID: h.config.ClientID})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		h.log.Error("Failed to verify ID token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify token"})
		return
	}

	// Extract claims
	var claims struct {
		Sub    string   `json:"sub"`
		Email  string   `json:"email"`
		Groups []string `json:"groups"`
	}
	if err := idToken.Claims(&claims); err != nil {
		h.log.Error("Failed to extract claims: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to extract claims"})
		return
	}

	// Create session
	sessionID, err := h.sessionService.CreateSession(claims.Sub, claims.Email, claims.Groups)
	if err != nil {
		h.log.Error("Failed to create session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	// Set session cookie
	c.SetCookie("session", sessionID, 86400*7, "/", "", true, true)

	h.log.Info("User authenticated: %s (%s)", claims.Email, claims.Sub)

	// Redirect to home page
	c.Redirect(http.StatusFound, "/")
}

// Logout logs out the user.
func (h *AuthHandler) Logout(c *gin.Context) {
	// Get session cookie
	sessionCookie, err := c.Cookie("session")
	if err == nil && sessionCookie != "" {
		h.sessionService.DeleteSession(sessionCookie)
	}

	// Clear session cookie
	c.SetCookie("session", "", -1, "/", "", true, true)

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// UserInfo returns current user information.
func (h *AuthHandler) UserInfo(c *gin.Context) {
	if !h.config.Enabled {
		c.JSON(http.StatusOK, gin.H{
			"authenticated": false,
			"oidc_enabled":  false,
		})
		return
	}

	// Get session cookie
	sessionCookie, err := c.Cookie("session")
	if err != nil || sessionCookie == "" {
		c.JSON(http.StatusOK, gin.H{
			"authenticated": false,
			"oidc_enabled":  true,
		})
		return
	}

	// Get session info
	session, exists := h.sessionService.GetSessionInfo(sessionCookie)
	if !exists {
		c.JSON(http.StatusOK, gin.H{
			"authenticated": false,
			"oidc_enabled":  true,
		})
		return
	}

	// Check if user is admin
	isAdmin := false
	for _, group := range session.Groups {
		if group == "ADMIN" {
			isAdmin = true
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"authenticated": true,
		"oidc_enabled":  true,
		"user_id":       session.UserID,
		"email":         session.Email,
		"groups":        session.Groups,
		"is_admin":      isAdmin,
	})
}

// generateState generates a random state string for CSRF protection.
func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
