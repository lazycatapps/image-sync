// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/lazycatapps/image-sync/internal/models"
	"github.com/lazycatapps/image-sync/internal/pkg/logger"
)

const imageInspectTimeout = 30 * time.Second

// ImageService defines the interface for image inspection operations.
type ImageService interface {
	InspectImage(req *models.InspectRequest) (*models.InspectResponse, error)
}

// imageService implements the ImageService interface.
type imageService struct {
	logger logger.Logger
}

// NewImageService creates a new ImageService instance.
func NewImageService(logger logger.Logger) ImageService {
	return &imageService{
		logger: logger,
	}
}

// InspectImage uses skopeo to fetch image manifest and extract available architectures.
// It supports both multi-arch manifest lists and single-arch manifests.
func (s *imageService) InspectImage(req *models.InspectRequest) (*models.InspectResponse, error) {
	// Create temporary auth file if credentials are provided
	authFile, err := createAuthFileForInspect(req.Image, req.Username, req.Password)
	if err != nil {
		s.logger.Error("Failed to create auth file for %s: %v", req.Image, err)
		return nil, fmt.Errorf("failed to create auth file: %w", err)
	}
	// Ensure auth file is deleted after use
	if authFile != "" {
		defer func() {
			if err := os.Remove(authFile); err != nil {
				s.logger.Error("Failed to remove auth file: %v", err)
			}
		}()
	}

	args := []string{"inspect", "--raw"}

	// Add TLS verification flag
	tlsVerify := true
	if req.TLSVerify != nil {
		tlsVerify = *req.TLSVerify
	}
	args = append(args, fmt.Sprintf("--tls-verify=%v", tlsVerify))

	// Credentials are now handled via REGISTRY_AUTH_FILE environment variable
	// No longer adding --creds to command line

	args = append(args, fmt.Sprintf("docker://%s", req.Image))

	// Execute skopeo inspect command
	s.logger.Info("Inspecting image: %s", req.Image)

	ctx, cancel := context.WithTimeout(context.Background(), imageInspectTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "skopeo", args...)

	// Set REGISTRY_AUTH_FILE environment variable if auth file exists
	if authFile != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("REGISTRY_AUTH_FILE=%s", authFile))
	}

	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		s.logger.Error("Image inspection timed out after %s", imageInspectTimeout)
		return nil, fmt.Errorf("image inspection timed out after %s", imageInspectTimeout)
	}
	if err != nil {
		s.logger.Error("Failed to inspect image %s: %v", req.Image, err)
		return nil, fmt.Errorf("failed to inspect image: %w", err)
	}

	// Parse JSON manifest
	var inspectResult map[string]interface{}
	if err := json.Unmarshal(output, &inspectResult); err != nil {
		s.logger.Error("Failed to parse inspect result for %s: %v", req.Image, err)
		return nil, fmt.Errorf("failed to parse inspect result: %w", err)
	}

	// Extract architectures from manifest
	architectures := s.extractArchitectures(inspectResult)

	s.logger.Info("Image %s has %d architecture(s)", req.Image, len(architectures))

	return &models.InspectResponse{
		Architectures: architectures,
	}, nil
}

// extractArchitectures parses the skopeo inspect result and extracts architecture information.
// It handles both multi-arch manifest lists and single-arch manifests.
func (s *imageService) extractArchitectures(inspectResult map[string]interface{}) []string {
	architectures := []string{}

	// Check for multi-arch manifest list
	if manifests, ok := inspectResult["manifests"].([]interface{}); ok {
		for _, manifest := range manifests {
			if m, ok := manifest.(map[string]interface{}); ok {
				if platform, ok := m["platform"].(map[string]interface{}); ok {
					if os, ok := platform["os"].(string); ok {
						if arch, ok := platform["architecture"].(string); ok {
							archStr := fmt.Sprintf("%s/%s", os, arch)
							// Include variant if present (e.g., arm/v7)
							if variant, ok := platform["variant"].(string); ok && variant != "" {
								archStr = fmt.Sprintf("%s/%s", archStr, variant)
							}
							architectures = append(architectures, archStr)
						}
					}
				}
			}
		}
	} else {
		// Single-arch manifest
		if arch, ok := inspectResult["Architecture"].(string); ok {
			if os, ok := inspectResult["Os"].(string); ok {
				architectures = append(architectures, fmt.Sprintf("%s/%s", strings.ToLower(os), arch))
			}
		}
	}

	return architectures
}

// extractRegistryForInspect extracts the registry domain from an image URL.
// This is a duplicate of the function in sync_service.go for image inspection.
func extractRegistryForInspect(imageURL string) string {
	// Remove "docker://" prefix if present
	imageURL = strings.TrimPrefix(imageURL, "docker://")

	// Split by "/" to get the first part
	parts := strings.Split(imageURL, "/")
	if len(parts) == 0 {
		return ""
	}

	// The first part is the registry (may include port)
	registry := parts[0]

	// If the first part doesn't contain a "." or ":", it's likely just the image name (e.g., "nginx")
	// In this case, default to "docker.io"
	if !strings.Contains(registry, ".") && !strings.Contains(registry, ":") {
		return "docker.io"
	}

	return registry
}

// createAuthFileForInspect creates a temporary Docker-compatible auth file for skopeo inspect.
// It returns the file path and an error if any.
// The caller is responsible for deleting the file after use.
func createAuthFileForInspect(image, username, password string) (string, error) {
	// If no credentials provided, return empty string (no auth file needed)
	if username == "" || password == "" {
		return "", nil
	}

	// Build auth config
	authConfig := map[string]interface{}{
		"auths": map[string]interface{}{},
	}

	registry := extractRegistryForInspect(image)
	authStr := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	authConfig["auths"].(map[string]interface{})[registry] = map[string]string{
		"auth": authStr,
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "skopeo-auth-*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temp auth file: %w", err)
	}
	defer tmpFile.Close()

	// Set restrictive permissions (0600 = rw-------)
	if err := os.Chmod(tmpFile.Name(), 0600); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to set auth file permissions: %w", err)
	}

	// Write auth config to file
	encoder := json.NewEncoder(tmpFile)
	if err := encoder.Encode(authConfig); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write auth file: %w", err)
	}

	return tmpFile.Name(), nil
}
