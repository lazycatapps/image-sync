// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package main is the entry point for the Image Sync server application.
// It initializes all dependencies, configures the server, and starts the HTTP service.
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/lazycatapps/image-sync/internal/handler"
	"github.com/lazycatapps/image-sync/internal/pkg/logger"
	"github.com/lazycatapps/image-sync/internal/repository"
	"github.com/lazycatapps/image-sync/internal/router"
	"github.com/lazycatapps/image-sync/internal/service"
	"github.com/lazycatapps/image-sync/internal/types"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd is the root command for the CLI application.
// It defines the application name, description, and the main execution function.
var rootCmd = &cobra.Command{
	Use:   "image-sync",
	Short: "Image Sync - Container image synchronization tool",
	Long:  `A web service for synchronizing container images between registries using Skopeo.`,
	Run:   runServer,
}

// init initializes command-line flags and environment variable bindings.
// It sets up the following configuration options:
//   - --host: Server listening address (default: 0.0.0.0)
//   - --port: Server listening port (default: 8080)
//   - --timeout: Sync operation timeout in seconds (default: 600)
//   - --default-source-registry: Default source registry prefix
//   - --default-dest-registry: Default destination registry prefix
//   - --cors-allowed-origins: CORS allowed origins (default: *)
//   - --config-dir: Directory for storing configuration files (default: /configs)
//
// Environment variables are supported with SYNC_ prefix and underscores replacing hyphens.
// For example: SYNC_DEFAULT_SOURCE_REGISTRY for --default-source-registry.
func init() {
	rootCmd.Flags().String("host", "0.0.0.0", "Server host")
	rootCmd.Flags().IntP("port", "p", 8080, "Server port")
	rootCmd.Flags().IntP("timeout", "t", 600, "Sync timeout in seconds")
	rootCmd.Flags().String("default-source-registry", "", "Default source registry")
	rootCmd.Flags().String("default-dest-registry", "", "Default destination registry")
	rootCmd.Flags().StringSlice("cors-allowed-origins", []string{"*"}, "CORS allowed origins")
	rootCmd.Flags().String("config-dir", "./configs", "Directory for storing configuration files")
	rootCmd.Flags().Bool("allow-password-save", false, "Allow saving passwords in configuration files (default: false for security)")
	rootCmd.Flags().Int("max-config-size", 4096, "Maximum configuration file size in bytes (default: 4096)")
	rootCmd.Flags().Int("max-config-files", 1000, "Maximum number of configuration files per user (default: 1000)")
	rootCmd.Flags().String("oidc-client-id", "", "OIDC client ID")
	rootCmd.Flags().String("oidc-client-secret", "", "OIDC client secret")
	rootCmd.Flags().String("oidc-issuer", "", "OIDC issuer URL")
	rootCmd.Flags().String("oidc-redirect-url", "", "OIDC redirect URL")

	viper.BindPFlags(rootCmd.Flags())

	// Set environment variable prefix to "SYNC"
	viper.SetEnvPrefix("SYNC")
	viper.AutomaticEnv()
	// Replace hyphens with underscores in environment variable names
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
}

// runServer is the main server execution function.
// It performs the following steps:
//  1. Loads configuration from command-line flags and environment variables
//  2. Initializes logger
//  3. Creates repository for task storage (in-memory)
//  4. Initializes services (sync, image inspection, session, config)
//  5. Sets up HTTP handlers (including auth handler if OIDC enabled)
//  6. Configures routing and middleware
//  7. Starts the HTTP server
func runServer(cmd *cobra.Command, args []string) {
	// Load configuration from viper
	oidcClientID := viper.GetString("oidc-client-id")
	oidcClientSecret := viper.GetString("oidc-client-secret")
	oidcIssuer := viper.GetString("oidc-issuer")
	oidcRedirectURL := viper.GetString("oidc-redirect-url")

	cfg := &types.Config{
		Server: types.ServerConfig{
			Host: viper.GetString("host"),
			Port: viper.GetInt("port"),
		},
		Registry: types.RegistryConfig{
			DefaultSourceRegistry: viper.GetString("default-source-registry"),
			DefaultDestRegistry:   viper.GetString("default-dest-registry"),
		},
		Sync: types.SyncConfig{
			Timeout: viper.GetInt("timeout"),
		},
		CORS: types.CORSConfig{
			AllowedOrigins: viper.GetStringSlice("cors-allowed-origins"),
		},
		Storage: types.StorageConfig{
			ConfigDir: viper.GetString("config-dir"),
		},
		OIDC: types.OIDCConfig{
			ClientID:     oidcClientID,
			ClientSecret: oidcClientSecret,
			Issuer:       oidcIssuer,
			RedirectURL:  oidcRedirectURL,
			Enabled:      oidcClientID != "" && oidcClientSecret != "" && oidcIssuer != "",
		},
	}

	// Initialize logger
	log := logger.New()

	// Log OIDC configuration status
	if cfg.OIDC.Enabled {
		log.Info("OIDC authentication enabled")
		log.Info("  Issuer: %s", cfg.OIDC.Issuer)
		log.Info("  Client ID: %s", cfg.OIDC.ClientID)
		log.Info("  Redirect URL: %s", cfg.OIDC.RedirectURL)
		log.Info("  Client Secret: %s", maskSecret(cfg.OIDC.ClientSecret))
	} else {
		log.Info("OIDC authentication disabled")
		log.Debug("  OIDC_CLIENT_ID: %s", maskSecret(oidcClientID))
		log.Debug("  OIDC_CLIENT_SECRET: %s", maskSecret(oidcClientSecret))
		log.Debug("  OIDC_ISSUER: %s", oidcIssuer)
		log.Debug("  OIDC_REDIRECT_URL: %s", oidcRedirectURL)
	}

	// Initialize repository (in-memory task storage)
	taskRepo := repository.NewInMemoryTaskRepository()

	// Initialize services
	syncService := service.NewSyncService(taskRepo, log, cfg.Sync.Timeout)
	imageService := service.NewImageService(log)
	allowPasswordSave := viper.GetBool("allow-password-save")
	maxConfigSize := viper.GetInt("max-config-size")
	maxConfigFiles := viper.GetInt("max-config-files")
	configService := service.NewConfigService(cfg.Storage.ConfigDir, allowPasswordSave, maxConfigSize, maxConfigFiles, log)
	sessionService := service.NewSessionService(7 * 24 * time.Hour) // 7 days session TTL

	// Initialize HTTP handlers
	syncHandler := handler.NewSyncHandler(syncService, cfg, log)
	imageHandler := handler.NewImageHandler(imageService, log)
	configHandler := handler.NewConfigHandler(configService, log)

	// Initialize auth handler
	authHandler, err := handler.NewAuthHandler(&cfg.OIDC, sessionService, log)
	if err != nil {
		log.Error("Failed to initialize auth handler: %v", err)
		return
	}

	// Set up router and middleware
	router := router.New(syncHandler, imageHandler, configHandler, authHandler, sessionService)
	engine := router.Setup(cfg)

	// Start HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Info("Server starting on %s (timeout: %ds)", addr, cfg.Sync.Timeout)
	if err := engine.Run(addr); err != nil {
		log.Error("Failed to start server: %v", err)
	}
}

// maskSecret masks a secret string for logging.
// Shows first 4 characters if length > 8, otherwise shows masked string.
func maskSecret(secret string) string {
	if secret == "" {
		return "(empty)"
	}
	if len(secret) <= 8 {
		return "***"
	}
	return secret[:4] + "***"
}

// main is the application entry point.
func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}
