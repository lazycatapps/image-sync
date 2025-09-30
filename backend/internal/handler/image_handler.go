// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

package handler

import (
	"errors"
	"net/http"

	"github.com/lazycatapps/image-sync/internal/models"
	apperrors "github.com/lazycatapps/image-sync/internal/pkg/errors"
	"github.com/lazycatapps/image-sync/internal/pkg/logger"
	"github.com/lazycatapps/image-sync/internal/pkg/validator"
	"github.com/lazycatapps/image-sync/internal/service"

	"github.com/gin-gonic/gin"
)

// ImageHandler handles HTTP requests related to image inspection.
type ImageHandler struct {
	imageService service.ImageService
	logger       logger.Logger
}

// NewImageHandler creates a new ImageHandler instance.
func NewImageHandler(imageService service.ImageService, logger logger.Logger) *ImageHandler {
	return &ImageHandler{
		imageService: imageService,
		logger:       logger,
	}
}

// handleError processes errors and sends appropriate HTTP responses.
func (h *ImageHandler) handleError(c *gin.Context, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.StatusCode, gin.H{"error": appErr.Message})
	} else {
		h.logger.Error("Unexpected error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
	}
}

// InspectImage inspects a container image and returns available architectures.
// It uses skopeo to fetch the image manifest and extract platform information.
//
// Request body (JSON):
//   - image (required): Image address (e.g., "docker.io/library/nginx:latest")
//   - username (optional): Registry username
//   - password (optional): Registry password
//   - tlsVerify (optional): TLS verification flag
//
// Response (200 OK):
//
//	{"architectures": ["linux/amd64", "linux/arm64", "linux/arm/v7"]}
//
// Error responses: 400 (invalid input), 500 (inspection failed)
func (h *ImageHandler) InspectImage(c *gin.Context) {
	var req models.InspectRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleError(c, apperrors.WrapInvalidInput(err, "Invalid request body"))
		return
	}

	// Validate input fields for security
	if err := validator.ValidateImageName(req.Image); err != nil {
		h.handleError(c, apperrors.WrapInvalidInput(err, "Invalid image name"))
		return
	}

	if err := validator.ValidateCredentials(req.Username, req.Password); err != nil {
		h.handleError(c, apperrors.WrapInvalidInput(err, "Invalid credentials"))
		return
	}

	h.logger.Info("Inspecting image: %s", req.Image)

	resp, err := h.imageService.InspectImage(&req)
	if err != nil {
		h.logger.Error("Failed to inspect image %s: %v", req.Image, err)
		h.handleError(c, apperrors.WrapCommandFailed(err, "Failed to inspect image"))
		return
	}

	h.logger.Info("Image inspection completed: %s (%d architectures)", req.Image, len(resp.Architectures))

	c.JSON(http.StatusOK, resp)
}