// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

package handler

import (
	"net/http"

	"github.com/lazycatapps/image-sync/internal/models"
	apperrors "github.com/lazycatapps/image-sync/internal/pkg/errors"
	"github.com/lazycatapps/image-sync/internal/pkg/logger"
	"github.com/lazycatapps/image-sync/internal/service"

	"github.com/gin-gonic/gin"
)

// getUserIdentifier extracts the user identifier from the session stored in the context.
// The identifier is used as the subdirectory name for user-specific config files.
// Returns empty string if OIDC is not enabled or session is not found.
//
// Current implementation: Uses session.Email for better readability in file system.
// If you need to change the identifier (e.g., back to UserID), just modify the return line below.
func getUserIdentifier(c *gin.Context) string {
	sessionInfo, exists := c.Get("session")
	if !exists {
		return ""
	}

	session, ok := sessionInfo.(*service.SessionInfo)
	if !ok {
		return ""
	}

	// Change this line if you want to use a different identifier
	return session.Email + "_" + session.UserID // Could also be: session.UserID, session.Username, etc.
}

// ConfigHandler handles HTTP requests for user configuration management.
type ConfigHandler struct {
	configService *service.ConfigService
	logger        logger.Logger
}

// NewConfigHandler creates a new configuration handler.
func NewConfigHandler(configService *service.ConfigService, log logger.Logger) *ConfigHandler {
	return &ConfigHandler{
		configService: configService,
		logger:        log,
	}
}

// ListConfigs handles GET /api/v1/configs
// Returns a list of all saved configuration names for the current user
func (h *ConfigHandler) ListConfigs(c *gin.Context) {
	userIdentifier := getUserIdentifier(c)

	configs, err := h.configService.ListConfigs(userIdentifier)
	if err != nil {
		appErr, ok := err.(*apperrors.AppError)
		if ok {
			c.JSON(appErr.StatusCode, gin.H{"error": appErr.Message})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"configs": configs})
}

// GetConfig handles GET /api/v1/config/:name
// Retrieves a saved user configuration by name for the current user
func (h *ConfigHandler) GetConfig(c *gin.Context) {
	userIdentifier := getUserIdentifier(c)
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "config name is required"})
		return
	}

	config, err := h.configService.GetConfig(userIdentifier, name)
	if err != nil {
		appErr, ok := err.(*apperrors.AppError)
		if ok {
			c.JSON(appErr.StatusCode, gin.H{"error": appErr.Message})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, config)
}

// SaveConfig handles POST /api/v1/config/:name
// Saves user configuration with the given name for the current user
func (h *ConfigHandler) SaveConfig(c *gin.Context) {
	userIdentifier := getUserIdentifier(c)
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "config name is required"})
		return
	}

	var config models.UserConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Save configuration
	if err := h.configService.SaveConfig(userIdentifier, name, &config); err != nil {
		appErr, ok := err.(*apperrors.AppError)
		if ok {
			c.JSON(appErr.StatusCode, gin.H{"error": appErr.Message})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Configuration saved successfully"})
}

// DeleteConfig handles DELETE /api/v1/config/:name
// Deletes a saved user configuration by name for the current user
func (h *ConfigHandler) DeleteConfig(c *gin.Context) {
	userIdentifier := getUserIdentifier(c)
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "config name is required"})
		return
	}

	if err := h.configService.DeleteConfig(userIdentifier, name); err != nil {
		appErr, ok := err.(*apperrors.AppError)
		if ok {
			c.JSON(appErr.StatusCode, gin.H{"error": appErr.Message})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Configuration deleted successfully"})
}

// GetLastUsedConfig handles GET /api/v1/config/last-used
// Returns the name of the last used configuration for the current user
func (h *ConfigHandler) GetLastUsedConfig(c *gin.Context) {
	userIdentifier := getUserIdentifier(c)

	name, err := h.configService.GetLastUsedConfig(userIdentifier)
	if err != nil {
		appErr, ok := err.(*apperrors.AppError)
		if ok {
			c.JSON(appErr.StatusCode, gin.H{"error": appErr.Message})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"name": name})
}
