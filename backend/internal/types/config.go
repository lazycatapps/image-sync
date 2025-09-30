// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package types defines configuration types for the Image Sync application.
package types

// Config represents the complete application configuration.
type Config struct {
	Server   ServerConfig   // HTTP server configuration
	Registry RegistryConfig // Default registry configuration
	Sync     SyncConfig     // Sync operation configuration
	CORS     CORSConfig     // CORS policy configuration
	Storage  StorageConfig  // Storage configuration
	OIDC     OIDCConfig     // OIDC authentication configuration
}

// ServerConfig defines HTTP server listening configuration.
type ServerConfig struct {
	Host string // Server listening address (e.g., "0.0.0.0", "127.0.0.1")
	Port int    // Server listening port (e.g., 8080)
}

// RegistryConfig defines default container registry addresses.
// These are used to pre-fill the frontend UI fields.
type RegistryConfig struct {
	DefaultSourceRegistry string // Default source registry prefix (e.g., "registry.example.com/")
	DefaultDestRegistry   string // Default destination registry prefix
}

// SyncConfig defines sync operation behavior.
type SyncConfig struct {
	Timeout int // Sync operation timeout in seconds (default: 600)
}

// CORSConfig defines Cross-Origin Resource Sharing policy.
type CORSConfig struct {
	AllowedOrigins []string // Allowed origins (e.g., ["*"], ["https://app.example.com"])
}

// StorageConfig defines storage configuration.
type StorageConfig struct {
	ConfigDir string // Directory for storing configuration files (default: "/configs")
}

// OIDCConfig defines OIDC authentication configuration.
type OIDCConfig struct {
	ClientID     string // OIDC client ID
	ClientSecret string // OIDC client secret
	Issuer       string // OIDC issuer URL
	RedirectURL  string // OIDC redirect URL after authentication
	Enabled      bool   // Whether OIDC authentication is enabled
}
