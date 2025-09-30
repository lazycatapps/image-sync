// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package router provides HTTP routing configuration for the Image Sync server.
package router

import (
	"github.com/lazycatapps/image-sync/internal/handler"
	"github.com/lazycatapps/image-sync/internal/middleware"
	"github.com/lazycatapps/image-sync/internal/types"

	"github.com/gin-gonic/gin"
)

// Router manages HTTP request routing and handler registration.
// It holds references to all HTTP handlers (sync, image inspection, config, etc.).
type Router struct {
	syncHandler      *handler.SyncHandler
	imageHandler     *handler.ImageHandler
	configHandler    *handler.ConfigHandler
	authHandler      *handler.AuthHandler
	sessionValidator middleware.SessionValidator
}

// New creates a new Router instance with the provided handlers.
func New(syncHandler *handler.SyncHandler, imageHandler *handler.ImageHandler, configHandler *handler.ConfigHandler, authHandler *handler.AuthHandler, sessionValidator middleware.SessionValidator) *Router {
	return &Router{
		syncHandler:      syncHandler,
		imageHandler:     imageHandler,
		configHandler:    configHandler,
		authHandler:      authHandler,
		sessionValidator: sessionValidator,
	}
}

// Setup initializes the Gin engine with middleware and routes.
// It configures the following middleware in order:
//  1. gin.Logger() - HTTP request logging
//  2. gin.Recovery() - Panic recovery
//  3. CORS - Cross-Origin Resource Sharing
//  4. Auth - OIDC authentication (if enabled)
//
// Returns a configured *gin.Engine ready to serve HTTP requests.
func (r *Router) Setup(cfg *types.Config) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())
	engine.Use(middleware.CORS(cfg.CORS.AllowedOrigins))
	engine.Use(middleware.Auth(cfg.OIDC.Enabled, r.sessionValidator))

	// Disable trusted proxy feature for security
	engine.SetTrustedProxies(nil)

	r.registerRoutes(engine)

	return engine
}

// registerRoutes registers all API routes under /api/v1 prefix.
// Available endpoints:
//   - GET    /health               - Health check
//   - GET    /auth/login           - Redirect to OIDC provider for login
//   - GET    /auth/callback        - OIDC callback handler
//   - POST   /auth/logout          - Logout current user
//   - GET    /auth/userinfo        - Get current user information
//   - GET    /sync                 - List sync tasks with pagination and filtering
//   - POST   /sync                 - Create a new sync task
//   - GET    /sync/:id             - Get sync task status and details
//   - GET    /sync/:id/logs        - Stream sync task logs via SSE
//   - GET    /env/defaults         - Get default registry configuration
//   - POST   /inspect              - Inspect image and list available architectures
//   - GET    /configs              - List all saved configuration names
//   - GET    /config/:name         - Get a saved user configuration by name
//   - POST   /config/:name         - Save user configuration with name
//   - DELETE /config/:name         - Delete a saved user configuration by name
//   - GET    /config/last-used     - Get the name of the last used configuration
func (r *Router) registerRoutes(engine *gin.Engine) {
	api := engine.Group("/api/v1")
	{
		// Public endpoints (no auth required)
		api.GET("/health", r.syncHandler.Health)

		// Auth endpoints
		auth := api.Group("/auth")
		{
			auth.GET("/login", r.authHandler.Login)
			auth.GET("/callback", r.authHandler.Callback)
			auth.POST("/logout", r.authHandler.Logout)
			auth.GET("/userinfo", r.authHandler.UserInfo)
		}

		// Protected endpoints (require auth if OIDC enabled)
		api.GET("/sync", r.syncHandler.ListTasks)
		api.POST("/sync", r.syncHandler.SyncImage)
		api.GET("/sync/:id", r.syncHandler.GetSyncStatus)
		api.GET("/sync/:id/logs", r.syncHandler.StreamLogs)
		api.GET("/env/defaults", r.syncHandler.GetEnvDefaults)
		api.POST("/inspect", r.imageHandler.InspectImage)

		// Config management endpoints
		api.GET("/configs", r.configHandler.ListConfigs)
		api.GET("/config/last-used", r.configHandler.GetLastUsedConfig)
		api.GET("/config/:name", r.configHandler.GetConfig)
		api.POST("/config/:name", r.configHandler.SaveConfig)
		api.DELETE("/config/:name", r.configHandler.DeleteConfig)
	}
}