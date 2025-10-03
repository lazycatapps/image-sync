// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package handler provides HTTP request handlers for the Image Sync API.
package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/lazycatapps/image-sync/internal/models"
	apperrors "github.com/lazycatapps/image-sync/internal/pkg/errors"
	"github.com/lazycatapps/image-sync/internal/pkg/logger"
	"github.com/lazycatapps/image-sync/internal/pkg/validator"
	"github.com/lazycatapps/image-sync/internal/repository"
	"github.com/lazycatapps/image-sync/internal/service"
	"github.com/lazycatapps/image-sync/internal/types"

	"github.com/gin-gonic/gin"
)

// SyncHandler handles HTTP requests related to image synchronization tasks.
type SyncHandler struct {
	syncService service.SyncService
	config      *types.Config
	logger      logger.Logger
}

// NewSyncHandler creates a new SyncHandler instance.
func NewSyncHandler(syncService service.SyncService, cfg *types.Config, logger logger.Logger) *SyncHandler {
	return &SyncHandler{
		syncService: syncService,
		config:      cfg,
		logger:      logger,
	}
}

// handleError processes errors and sends appropriate HTTP responses.
// It checks if the error is an AppError with status code, otherwise returns 500.
func (h *SyncHandler) handleError(c *gin.Context, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.StatusCode, gin.H{"error": appErr.Message})
	} else {
		h.logger.Error("Unexpected error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
	}
}

// SyncImage creates a new image synchronization task.
// It validates the request, creates a task record, and starts sync execution asynchronously.
//
// Request body (JSON):
//   - sourceImage (required): Source image address
//   - destImage (required): Destination image address
//   - architecture (optional): Target architecture (e.g., "linux/amd64", "all")
//   - sourceUsername, sourcePassword (optional): Source registry credentials
//   - destUsername, destPassword (optional): Destination registry credentials
//   - srcTLSVerify, destTLSVerify (optional): TLS verification flags
//
// Response (200 OK):
//
//	{"message": "Sync started", "id": "task-uuid"}
//
// Error responses: 400 (invalid input), 500 (server error)
func (h *SyncHandler) SyncImage(c *gin.Context) {
	var req models.SyncRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind JSON request: %v", err)
		h.handleError(c, apperrors.WrapInvalidInput(err, "Invalid request body"))
		return
	}

	// Validate input fields for security
	if err := validator.ValidateImageName(req.SourceImage); err != nil {
		h.handleError(c, apperrors.WrapInvalidInput(err, "Invalid source image"))
		return
	}

	if err := validator.ValidateImageName(req.DestImage); err != nil {
		h.handleError(c, apperrors.WrapInvalidInput(err, "Invalid destination image"))
		return
	}

	if err := validator.ValidateArchitecture(req.Architecture); err != nil {
		h.handleError(c, apperrors.WrapInvalidInput(err, "Invalid architecture"))
		return
	}

	if err := validator.ValidateCredentials(req.SourceUsername, req.SourcePassword); err != nil {
		h.handleError(c, apperrors.WrapInvalidInput(err, "Invalid source credentials"))
		return
	}

	if err := validator.ValidateCredentials(req.DestUsername, req.DestPassword); err != nil {
		h.handleError(c, apperrors.WrapInvalidInput(err, "Invalid destination credentials"))
		return
	}

	if err := validator.ValidateRetryTimes(req.RetryTimes); err != nil {
		h.handleError(c, apperrors.WrapInvalidInput(err, "Invalid retry times"))
		return
	}

	taskID, err := h.syncService.CreateSyncTask(&req)
	if err != nil {
		h.logger.Error("Failed to create sync task: %v", err)
		h.handleError(c, apperrors.WrapInternal(err, "Failed to create sync task"))
		return
	}

	// Execute sync asynchronously
	go func() {
		if err := h.syncService.ExecuteSync(taskID, &req); err != nil {
			h.logger.Error("[%s] Sync execution failed: %v", taskID, err)
		}
	}()

	h.logger.Info("Sync task created: %s (source: %s, dest: %s)", taskID, req.SourceImage, req.DestImage)

	c.JSON(http.StatusOK, gin.H{
		"message": "Sync started",
		"id":      taskID,
	})
}

// GetSyncStatus retrieves the status and details of a sync task by ID.
//
// Path parameter:
//   - id: Task UUID
//
// Response (200 OK): Task object with all details (status, logs, timestamps, etc.)
// Error responses: 404 (task not found), 500 (server error)
func (h *SyncHandler) GetSyncStatus(c *gin.Context) {
	id := c.Param("id")

	task, err := h.syncService.GetTask(id)
	if err != nil {
		if errors.Is(err, repository.ErrTaskNotFound) {
			h.handleError(c, apperrors.WrapTaskNotFound(err))
			return
		}
		h.logger.Error("Failed to get task %s: %v", id, err)
		h.handleError(c, apperrors.WrapInternal(err, "Failed to get task"))
		return
	}

	c.JSON(http.StatusOK, task)
}

// StreamLogs streams task logs to the client using Server-Sent Events (SSE).
// It sends historical logs first, then streams new logs in real-time until task completes.
//
// Path parameter:
//   - id: Task UUID
//
// Response headers:
//   - Content-Type: text/event-stream
//   - Cache-Control: no-cache
//   - Connection: keep-alive
//
// Response format: SSE (data: <log line>\n\n)
// Error responses: 404 (task not found), 500 (server error)
func (h *SyncHandler) StreamLogs(c *gin.Context) {
	id := c.Param("id")

	task, err := h.syncService.GetTask(id)
	if err != nil {
		if errors.Is(err, repository.ErrTaskNotFound) {
			h.handleError(c, apperrors.WrapTaskNotFound(err))
			return
		}
		h.logger.Error("Failed to get task %s for log streaming: %v", id, err)
		h.handleError(c, apperrors.WrapInternal(err, "Failed to get task"))
		return
	}

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	// Send existing logs first
	existingLogs := task.GetLogLines()
	taskStatus := task.Status

	for _, line := range existingLogs {
		fmt.Fprintf(c.Writer, "data: %s\n\n", line)
		c.Writer.Flush()
	}

	// If task is already completed, no need to stream further
	if taskStatus == models.StatusCompleted || taskStatus == models.StatusFailed {
		return
	}

	// Subscribe to new logs
	logChan := task.AddLogListener()
	defer task.RemoveLogListener(logChan)

	// Stream new logs until task completes or client disconnects
	clientGone := c.Request.Context().Done()
	for {
		select {
		case line, ok := <-logChan:
			if !ok {
				// Channel closed, task completed
				return
			}
			fmt.Fprintf(c.Writer, "data: %s\n\n", line)
			c.Writer.Flush()
		case <-clientGone:
			// Client disconnected
			return
		}
	}
}

// GetEnvDefaults returns default registry configuration from environment variables.
//
// Response (200 OK):
//
//	{"sourceRegistry": "registry.example.com/", "destRegistry": "registry2.example.com/"}
func (h *SyncHandler) GetEnvDefaults(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"sourceRegistry": h.config.Registry.DefaultSourceRegistry,
		"destRegistry":   h.config.Registry.DefaultDestRegistry,
	})
}

// ListTasks lists sync tasks with pagination, filtering, and sorting.
//
// Query parameters:
//   - page (optional): Page number, default 1
//   - pageSize (optional): Items per page, default 20, max 100
//   - status (optional): Filter by status (pending/running/completed/failed)
//   - sortBy (optional): Sort field (startTime/endTime), default startTime
//   - sortOrder (optional): Sort direction (asc/desc), default desc
//
// Response (200 OK):
//
//	{"total": 100, "page": 1, "pageSize": 20, "tasks": [...]}
//
// Error responses: 400 (invalid parameters), 500 (server error)
func (h *SyncHandler) ListTasks(c *gin.Context) {
	var req models.TaskListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.handleError(c, apperrors.WrapInvalidInput(err, "Invalid query parameters"))
		return
	}

	resp, err := h.syncService.ListTasks(&req)
	if err != nil {
		h.logger.Error("Failed to list tasks: %v", err)
		h.handleError(c, apperrors.WrapInternal(err, "Failed to list tasks"))
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Health performs a health check and returns service status.
//
// Response (200 OK):
//
//	{"status": "ok"}
func (h *SyncHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}