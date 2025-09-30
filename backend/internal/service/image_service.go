// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

package service

import (
	"context"
	"encoding/json"
	"fmt"
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
	args := []string{"inspect", "--raw"}

	// Add TLS verification flag
	tlsVerify := true
	if req.TLSVerify != nil {
		tlsVerify = *req.TLSVerify
	}
	args = append(args, fmt.Sprintf("--tls-verify=%v", tlsVerify))

	// Add credentials if provided
	if req.Username != "" && req.Password != "" {
		args = append(args, "--creds", fmt.Sprintf("%s:%s", req.Username, req.Password))
	}

	args = append(args, fmt.Sprintf("docker://%s", req.Image))

	// Execute skopeo inspect command
	s.logger.Info("Inspecting image: %s", req.Image)

	ctx, cancel := context.WithTimeout(context.Background(), imageInspectTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "skopeo", args...)
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
