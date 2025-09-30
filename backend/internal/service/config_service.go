// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

package service

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/lazycatapps/image-sync/internal/models"
	"github.com/lazycatapps/image-sync/internal/pkg/errors"
	"github.com/lazycatapps/image-sync/internal/pkg/logger"
	"github.com/lazycatapps/image-sync/internal/pkg/validator"
)

const (
	configFilePrefix = "config_"
	configFileSuffix = ".json"
	lastUsedFileName = "last_used.txt"
)

// ConfigService handles user configuration persistence
type ConfigService struct {
	baseConfigDir     string // Base config directory
	allowPasswordSave bool   // Whether to allow saving passwords in config files
	maxConfigSize     int    // Maximum config file size in bytes
	maxConfigFiles    int    // Maximum number of configs per user
	mu                sync.RWMutex
	logger            logger.Logger
}

// NewConfigService creates a new config service
// configDir is the base directory where config files will be stored (default: /configs)
// allowPasswordSave controls whether passwords can be saved in config files (default: false for maximum security)
// maxConfigSize is the maximum size of a single config file in bytes (default: 4096)
// maxConfigFiles is the maximum number of config files per user (default: 1000)
func NewConfigService(configDir string, allowPasswordSave bool, maxConfigSize, maxConfigFiles int, log logger.Logger) *ConfigService {
	service := &ConfigService{
		baseConfigDir:     configDir,
		allowPasswordSave: allowPasswordSave,
		maxConfigSize:     maxConfigSize,
		maxConfigFiles:    maxConfigFiles,
		logger:            log,
	}

	if err := os.MkdirAll(service.baseConfigDir, 0700); err != nil {
		service.logger.Error("Failed to initialize config directory %s: %v", service.baseConfigDir, err)
	}

	log.Info("ConfigService initialized with allowPasswordSave=%v, maxConfigSize=%d, maxConfigFiles=%d",
		allowPasswordSave, maxConfigSize, maxConfigFiles)
	return service
}

// getUserConfigDir returns the config directory for a specific user
// If userIdentifier is empty, returns the shared config directory
// userIdentifier can be email, userID, username, etc. - comes from handler layer
func (s *ConfigService) getUserConfigDir(userIdentifier string) string {
	if userIdentifier == "" {
		// No user identifier, use shared directory
		return s.baseConfigDir
	}
	// User-specific directory: /configs/users/{userIdentifier}
	safeIdentifier := sanitizeUserIdentifier(userIdentifier)
	return filepath.Join(s.baseConfigDir, "users", safeIdentifier)
}

// getConfigPath returns the file path for a given config name and user
func (s *ConfigService) getConfigPath(userIdentifier, name string) string {
	// Use filepath.Clean to prevent path traversal
	cleanName := filepath.Clean(name)
	// Double-check that the clean name doesn't contain path separators
	if strings.Contains(cleanName, string(os.PathSeparator)) {
		cleanName = filepath.Base(cleanName)
	}
	filename := configFilePrefix + cleanName + configFileSuffix
	configDir := s.getUserConfigDir(userIdentifier)
	return filepath.Join(configDir, filename)
}

// getLastUsedPath returns the path to the last used config name file for a user
func (s *ConfigService) getLastUsedPath(userIdentifier string) string {
	configDir := s.getUserConfigDir(userIdentifier)
	return filepath.Join(configDir, lastUsedFileName)
}

// ListConfigs returns a list of all saved configuration names for a user
func (s *ConfigService) ListConfigs(userIdentifier string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	configDir := s.getUserConfigDir(userIdentifier)

	// Ensure directory exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		// No configs exist yet
		return []string{}, nil
	}

	// Read directory
	entries, err := os.ReadDir(configDir)
	if err != nil {
		s.logger.Error("Failed to read config directory: %v", err)
		return nil, errors.WrapInternal(err, "Failed to read config directory")
	}

	// Extract config names from filenames
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filename := entry.Name()
		if strings.HasPrefix(filename, configFilePrefix) && strings.HasSuffix(filename, configFileSuffix) {
			// Extract name from "config_NAME.json"
			name := strings.TrimPrefix(filename, configFilePrefix)
			name = strings.TrimSuffix(name, configFileSuffix)
			names = append(names, name)
		}
	}

	// Sort alphabetically
	sort.Strings(names)

	s.logger.Info("Listed %d configs for user %s", len(names), userIdentifier)
	return names, nil
}

// GetConfig retrieves a saved configuration by name for a user
func (s *ConfigService) GetConfig(userIdentifier, name string) (*models.UserConfig, error) {
	// Validate config name
	if err := validator.ValidateConfigName(name); err != nil {
		return nil, errors.NewInvalidInput(err.Error())
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	configPath := s.getConfigPath(userIdentifier, name)

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return empty config if file doesn't exist
		return &models.UserConfig{}, nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		s.logger.Error("Failed to read config file %s: %v", name, err)
		return nil, errors.WrapInternal(err, "Failed to read config file")
	}

	// Parse JSON
	var config models.UserConfig
	if err := json.Unmarshal(data, &config); err != nil {
		s.logger.Error("Failed to parse config file %s: %v", name, err)
		return nil, errors.WrapInternal(err, "Failed to parse config file")
	}

	// Config data is stored in base64 format, return it as-is for secure transmission
	// Frontend will decode it for display
	if !s.allowPasswordSave {
		config.SourcePassword = ""
		config.DestPassword = ""
	}

	s.logger.Info("Config '%s' loaded successfully", name)
	return &config, nil
}

// SaveConfig saves a configuration with the given name for a user
func (s *ConfigService) SaveConfig(userIdentifier, name string, config *models.UserConfig) error {
	// Validate config name
	if err := validator.ValidateConfigName(name); err != nil {
		return errors.NewInvalidInput(err.Error())
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	configDir := s.getUserConfigDir(userIdentifier)

	// Check max configs limit (only for new configs)
	configPath := s.getConfigPath(userIdentifier, name)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// This is a new config, check limit
		configs, _ := s.listConfigsNoLock(userIdentifier)
		if len(configs) >= s.maxConfigFiles {
			return errors.NewInvalidInput(fmt.Sprintf("Maximum number of configs (%d) reached", s.maxConfigFiles))
		}
	}

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0700); err != nil {
		s.logger.Error("Failed to create config directory: %v", err)
		return errors.WrapInternal(err, "Failed to create config directory")
	}

	// Config data is already base64 encoded by frontend for secure transmission
	// We save it directly without re-encoding
	configToSave := *config

	if !s.allowPasswordSave {
		configToSave.SourcePassword = ""
		configToSave.DestPassword = ""
		s.logger.Info("Passwords removed from config before saving (allowPasswordSave=false)")
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(&configToSave, "", "  ")
	if err != nil {
		s.logger.Error("Failed to marshal config: %v", err)
		return errors.WrapInternal(err, "Failed to marshal config")
	}

	// Check config size limit
	if len(data) > s.maxConfigSize {
		s.logger.Error("Config size exceeds limit: %d bytes (max: %d bytes)", len(data), s.maxConfigSize)
		return errors.NewInvalidInput(fmt.Sprintf("Configuration size (%d bytes) exceeds maximum allowed size (%d bytes)", len(data), s.maxConfigSize))
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		s.logger.Error("Failed to write config file %s: %v", name, err)
		return errors.WrapInternal(err, "Failed to write config file")
	}

	// Update last used config
	s.setLastUsedConfigNoLock(userIdentifier, name)

	s.logger.Info("Config '%s' saved successfully to %s for user %s", name, configPath, userIdentifier)
	return nil
}

// DeleteConfig removes a saved configuration by name for a user
func (s *ConfigService) DeleteConfig(userIdentifier, name string) error {
	// Validate config name
	if err := validator.ValidateConfigName(name); err != nil {
		return errors.NewInvalidInput(err.Error())
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	configPath := s.getConfigPath(userIdentifier, name)

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// File doesn't exist, nothing to delete
		return nil
	}

	// Remove file
	if err := os.Remove(configPath); err != nil {
		s.logger.Error("Failed to delete config file %s: %v", name, err)
		return errors.WrapInternal(err, "Failed to delete config file")
	}

	// Clear last used if it was this config
	lastUsed, _ := s.getLastUsedConfigNoLock(userIdentifier)
	if lastUsed == name {
		_ = os.Remove(s.getLastUsedPath(userIdentifier))
	}

	s.logger.Info("Config '%s' deleted successfully for user %s", name, userIdentifier)
	return nil
}

// GetLastUsedConfig returns the name of the last used config for a user
func (s *ConfigService) GetLastUsedConfig(userIdentifier string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getLastUsedConfigNoLock(userIdentifier)
}

// getLastUsedConfigNoLock returns the last used config name without locking
func (s *ConfigService) getLastUsedConfigNoLock(userIdentifier string) (string, error) {
	lastUsedPath := s.getLastUsedPath(userIdentifier)

	// Check if file exists
	if _, err := os.Stat(lastUsedPath); os.IsNotExist(err) {
		return "", nil // No last used config
	}

	// Read file
	data, err := os.ReadFile(lastUsedPath)
	if err != nil {
		s.logger.Error("Failed to read last used config: %v", err)
		return "", errors.WrapInternal(err, "Failed to read last used config")
	}

	name := strings.TrimSpace(string(data))

	// Validate the name
	if err := validator.ValidateConfigName(name); err != nil {
		s.logger.Error("Invalid last used config name: %v", err)
		return "", nil // Ignore invalid names
	}

	return name, nil
}

// SetLastUsedConfig saves the name of the last used config for a user
func (s *ConfigService) SetLastUsedConfig(userIdentifier, name string) error {
	// Validate config name
	if err := validator.ValidateConfigName(name); err != nil {
		return errors.NewInvalidInput(err.Error())
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.setLastUsedConfigNoLock(userIdentifier, name)
}

// setLastUsedConfigNoLock saves the last used config name without locking
func (s *ConfigService) setLastUsedConfigNoLock(userIdentifier, name string) error {
	configDir := s.getUserConfigDir(userIdentifier)

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0700); err != nil {
		s.logger.Error("Failed to create config directory: %v", err)
		return errors.WrapInternal(err, "Failed to create config directory")
	}

	lastUsedPath := s.getLastUsedPath(userIdentifier)

	// Write config name to file
	if err := os.WriteFile(lastUsedPath, []byte(name), 0600); err != nil {
		s.logger.Error("Failed to write last used config: %v", err)
		return errors.WrapInternal(err, "Failed to write last used config")
	}

	return nil
}

// listConfigsNoLock lists configs without locking (for internal use)
func (s *ConfigService) listConfigsNoLock(userIdentifier string) ([]string, error) {
	configDir := s.getUserConfigDir(userIdentifier)

	// Ensure directory exists
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(configDir)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filename := entry.Name()
		if strings.HasPrefix(filename, configFilePrefix) && strings.HasSuffix(filename, configFileSuffix) {
			name := strings.TrimPrefix(filename, configFilePrefix)
			name = strings.TrimSuffix(name, configFileSuffix)
			names = append(names, name)
		}
	}

	return names, nil
}

func sanitizeUserIdentifier(identifier string) string {
	if identifier == "" {
		return ""
	}

	const maxLength = 128
	const allowed = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._-@"

	var builder strings.Builder
	for _, r := range identifier {
		if strings.ContainsRune(allowed, r) {
			builder.WriteRune(r)
		} else {
			builder.WriteRune('-')
		}
	}

	safe := builder.String()
	safe = strings.ReplaceAll(safe, "..", "-")
	safe = strings.Trim(safe, "-_.")

	if safe == "" {
		sum := sha256.Sum256([]byte(identifier))
		return hex.EncodeToString(sum[:8])
	}

	if len(safe) > maxLength {
		safe = safe[:maxLength]
	}

	return safe
}

func encodeToBase64(value string) string {
	if value == "" {
		return ""
	}
	return base64.StdEncoding.EncodeToString([]byte(value))
}

func decodeFromBase64(value string) string {
	if value == "" {
		return ""
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return value
	}
	return string(decoded)
}
