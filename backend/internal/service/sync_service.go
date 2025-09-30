// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package service provides business logic for image synchronization operations.
package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/lazycatapps/image-sync/internal/models"
	"github.com/lazycatapps/image-sync/internal/pkg/logger"
	"github.com/lazycatapps/image-sync/internal/repository"

	"github.com/google/uuid"
)

// SyncService defines the interface for image synchronization operations.
type SyncService interface {
	CreateSyncTask(req *models.SyncRequest) (string, error)
	GetTask(id string) (*models.SyncTask, error)
	ExecuteSync(taskID string, req *models.SyncRequest) error
	ListTasks(req *models.TaskListRequest) (*models.TaskListResponse, error)
}

// syncService implements the SyncService interface.
type syncService struct {
	repo    repository.TaskRepository
	logger  logger.Logger
	timeout int // Sync operation timeout in seconds
}

// NewSyncService creates a new SyncService instance.
func NewSyncService(repo repository.TaskRepository, logger logger.Logger, timeout int) SyncService {
	return &syncService{
		repo:    repo,
		logger:  logger,
		timeout: timeout,
	}
}

// CreateSyncTask creates a new sync task record in the repository.
// It generates a unique task ID and initializes the task with pending status.
func (s *syncService) CreateSyncTask(req *models.SyncRequest) (string, error) {
	taskID := uuid.New().String()

	// Default to "all" architectures if not specified
	architecture := req.Architecture
	if architecture == "" {
		architecture = "all"
	}

	task := models.NewSyncTask(taskID, req.SourceImage, req.DestImage, architecture)

	if err := s.repo.Create(task); err != nil {
		return "", fmt.Errorf("failed to create task: %w", err)
	}

	return taskID, nil
}

// GetTask retrieves a task by ID from the repository.
func (s *syncService) GetTask(id string) (*models.SyncTask, error) {
	return s.repo.Get(id)
}

// ExecuteSync executes the image synchronization operation using skopeo.
// It builds the skopeo command, executes it with timeout, captures output, and updates task status.
// This method runs asynchronously and should be called in a goroutine.
func (s *syncService) ExecuteSync(taskID string, req *models.SyncRequest) error {
	task, err := s.repo.Get(taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Update task status to running
	task.Status = models.StatusRunning
	task.Message = "Syncing image..."
	if err := s.repo.Update(task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	task.AddLog(fmt.Sprintf("Task started at %s", time.Now().Format(time.RFC3339)))

	// Build skopeo command arguments
	args := s.buildSkopeoArgs(task, req)

	// Log sanitized command (credentials masked)
	sanitizedCmd := sanitizeCommand(args)
	s.logger.Info("[%s] Starting sync: %s -> %s", taskID, req.SourceImage, req.DestImage)
	task.AddLog(fmt.Sprintf("Executing: %s", sanitizedCmd))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.timeout)*time.Second)
	defer cancel()

	// Execute skopeo command
	cmd := exec.CommandContext(ctx, "skopeo", args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return s.handleTaskError(task, "Failed to create stdout pipe", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return s.handleTaskError(task, "Failed to create stderr pipe", err)
	}

	if err := cmd.Start(); err != nil {
		return s.handleTaskError(task, "Failed to start command", err)
	}

	// Read command output in parallel goroutines
	var outputWg sync.WaitGroup
	outputWg.Add(2)

	go s.readOutput(task, stdoutPipe, &outputWg)
	go s.readOutput(task, stderrPipe, &outputWg)

	// Wait for command to complete
	err = cmd.Wait()

	// Check if timeout occurred
	if ctx.Err() == context.DeadlineExceeded {
		task.AddLog(fmt.Sprintf("Timeout exceeded (%ds)", s.timeout))
		s.logger.Error("[%s] Sync timeout after %ds", taskID, s.timeout)
		err = fmt.Errorf("command timeout after %ds", s.timeout)
	}

	// Wait for output goroutines to finish (with timeout)
	done := make(chan struct{})
	go func() {
		outputWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Output reading completed
	case <-time.After(5 * time.Second):
		s.logger.Error("[%s] WARNING: Output reading timed out", taskID)
	}

	endTime := time.Now()

	// Finalize task based on result
	if err != nil {
		task.AddLog(fmt.Sprintf("Sync failed: %v", err))
		s.logger.Error("[%s] Sync failed: %v", taskID, err)
	} else {
		task.AddLog(fmt.Sprintf("Sync completed at %s", endTime.Format(time.RFC3339)))
		s.logger.Info("[%s] Sync completed successfully", taskID)
	}

	// Close all log listeners (SSE connections)
	task.CloseAllLogListeners()

	// Update task with final status
	task.EndTime = &endTime
	task.Output = strings.Join(task.GetLogLines(), "\n")

	if err != nil {
		task.Status = models.StatusFailed
		task.Message = "Sync failed"
		task.ErrorOutput = err.Error()
	} else {
		task.Status = models.StatusCompleted
		task.Message = "Sync completed successfully"
	}

	if updateErr := s.repo.Update(task); updateErr != nil {
		s.logger.Error("[%s] Failed to update task status: %v", taskID, updateErr)
	}

	return nil
}

// buildSkopeoArgs constructs the skopeo command arguments based on the sync request.
// It handles TLS verification, credentials, architecture selection, and image addresses.
func (s *syncService) buildSkopeoArgs(task *models.SyncTask, req *models.SyncRequest) []string {
	args := []string{"copy"}

	// Add TLS verification flags
	srcTLSVerify := true
	if req.SrcTLSVerify != nil {
		srcTLSVerify = *req.SrcTLSVerify
	}
	destTLSVerify := true
	if req.DestTLSVerify != nil {
		destTLSVerify = *req.DestTLSVerify
	}

	args = append(args, fmt.Sprintf("--src-tls-verify=%v", srcTLSVerify))
	args = append(args, fmt.Sprintf("--dest-tls-verify=%v", destTLSVerify))

	// Add source registry credentials if provided
	if req.SourceUsername != "" && req.SourcePassword != "" {
		args = append(args, "--src-creds", fmt.Sprintf("%s:%s", req.SourceUsername, req.SourcePassword))
		task.AddLog("Using source credentials")
	}

	// Add destination registry credentials if provided
	if req.DestUsername != "" && req.DestPassword != "" {
		args = append(args, "--dest-creds", fmt.Sprintf("%s:%s", req.DestUsername, req.DestPassword))
		task.AddLog("Using destination credentials")
	}

	// Handle architecture selection
	if task.Architecture == "all" {
		args = append(args, "--all")
		task.AddLog("Copying all architectures")
	} else if task.Architecture != "" {
		// Parse architecture format: os/arch or os/arch/variant
		parts := strings.Split(task.Architecture, "/")
		if len(parts) >= 2 {
			args = append(args, "--override-os", parts[0])
			args = append(args, "--override-arch", parts[1])
			if len(parts) > 2 {
				args = append(args, "--override-variant", parts[2])
			}
			task.AddLog(fmt.Sprintf("Copying architecture: %s", task.Architecture))
		}
	}

	// Add source and destination image addresses
	args = append(args, fmt.Sprintf("docker://%s", task.SourceImage))
	args = append(args, fmt.Sprintf("docker://%s", task.DestImage))

	return args
}

// readOutput reads command output from a pipe and adds it to the task log.
// It runs in a separate goroutine and signals completion via WaitGroup.
func (s *syncService) readOutput(task *models.SyncTask, pipe io.ReadCloser, wg *sync.WaitGroup) {
	defer wg.Done()

	reader := bufio.NewReader(pipe)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			// Handle any remaining partial line
			if line != "" {
				task.AddLog(strings.TrimSpace(line))
			}
			// EOF is normal, only log other errors
			if err != io.EOF {
				s.logger.Error("[%s] Output read error: %v", task.ID, err)
			}
			break
		}
		line = strings.TrimSpace(line)
		if line != "" {
			task.AddLog(line)
		}
	}
}

// handleTaskError updates the task with error information and marks it as failed.
func (s *syncService) handleTaskError(task *models.SyncTask, message string, err error) error {
	task.AddLog(fmt.Sprintf("Error: %v", err))
	task.Status = models.StatusFailed
	task.Message = message
	task.ErrorOutput = err.Error()
	endTime := time.Now()
	task.EndTime = &endTime

	if updateErr := s.repo.Update(task); updateErr != nil {
		s.logger.Error("[%s] Failed to update task: %v", task.ID, updateErr)
	}

	s.logger.Error("[%s] %s: %v", task.ID, message, err)
	return fmt.Errorf("%s: %w", message, err)
}

// ListTasks retrieves a paginated and filtered list of sync tasks.
// It supports filtering by status, sorting, and pagination.
func (s *syncService) ListTasks(req *models.TaskListRequest) (*models.TaskListResponse, error) {
	tasks, err := s.repo.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	// Filter by status if specified
	filtered := tasks
	if req.Status != "" {
		filtered = []*models.SyncTask{}
		for _, task := range tasks {
			if task.Status == req.Status {
				filtered = append(filtered, task)
			}
		}
	}

	// Sort tasks
	sortTasks(filtered, req.SortBy, req.SortOrder)

	// Pagination
	total := len(filtered)
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	pagedTasks := filtered[start:end]

	// Convert to summary format (excludes full logs)
	summaries := make([]*models.TaskSummary, len(pagedTasks))
	for i, task := range pagedTasks {
		summaries[i] = &models.TaskSummary{
			ID:           task.ID,
			SourceImage:  task.SourceImage,
			DestImage:    task.DestImage,
			Architecture: task.Architecture,
			Status:       task.Status,
			Message:      task.Message,
			StartTime:    task.StartTime,
			EndTime:      task.EndTime,
		}
	}

	return &models.TaskListResponse{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Tasks:    summaries,
	}, nil
}

// sortTasks sorts a slice of tasks in-place using bubble sort.
// Supports sorting by startTime or endTime, in ascending or descending order.
func sortTasks(tasks []*models.SyncTask, sortBy, sortOrder string) {
	if len(tasks) <= 1 {
		return
	}

	// Simple bubble sort for small datasets (sufficient for task lists)
	for i := 0; i < len(tasks)-1; i++ {
		for j := 0; j < len(tasks)-i-1; j++ {
			shouldSwap := false
			if sortBy == "endTime" {
				t1 := tasks[j].EndTime
				t2 := tasks[j+1].EndTime
				// Handle nil endTime (for running tasks)
				if t1 == nil && t2 != nil {
					shouldSwap = sortOrder == "asc"
				} else if t1 != nil && t2 == nil {
					shouldSwap = sortOrder == "desc"
				} else if t1 != nil && t2 != nil {
					if sortOrder == "desc" {
						shouldSwap = t1.Before(*t2)
					} else {
						shouldSwap = t1.After(*t2)
					}
				}
			} else {
				// Default to startTime
				if sortOrder == "desc" {
					shouldSwap = tasks[j].StartTime.Before(tasks[j+1].StartTime)
				} else {
					shouldSwap = tasks[j].StartTime.After(tasks[j+1].StartTime)
				}
			}
			if shouldSwap {
				tasks[j], tasks[j+1] = tasks[j+1], tasks[j]
			}
		}
	}
}

// sanitizeCommand replaces credentials in command arguments with "***:***" for safe logging.
func sanitizeCommand(args []string) string {
	sanitized := make([]string, len(args))
	skipNext := false
	for i, arg := range args {
		if skipNext {
			// Replace credentials argument
			sanitized[i] = "***:***"
			skipNext = false
		} else if strings.HasPrefix(arg, "--src-creds=") || strings.HasPrefix(arg, "--dest-creds=") {
			// Handle --src-creds=user:pass format
			sanitized[i] = strings.Split(arg, "=")[0] + "=***:***"
		} else if arg == "--creds" || arg == "--src-creds" || arg == "--dest-creds" {
			// Keep flag, but mark next argument for sanitization
			sanitized[i] = arg
			skipNext = true
		} else {
			sanitized[i] = arg
		}
	}
	return "skopeo " + strings.Join(sanitized, " ")
}
