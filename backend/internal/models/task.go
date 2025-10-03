// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package models defines data structures for the Image Sync application.
package models

import (
	"sync"
	"time"
)

// SyncStatus represents the current state of a sync task.
type SyncStatus string

const (
	StatusPending   SyncStatus = "pending"   // Task created, not yet started
	StatusRunning   SyncStatus = "running"   // Task is currently executing
	StatusCompleted SyncStatus = "completed" // Task completed successfully
	StatusFailed    SyncStatus = "failed"    // Task failed with error
)

// SyncTask represents an image synchronization task.
// It tracks task metadata, status, logs, and provides real-time log streaming to clients.
type SyncTask struct {
	ID           string        `json:"id"`           // Unique task identifier (UUID)
	SourceImage  string        `json:"sourceImage"`  // Source image address
	DestImage    string        `json:"destImage"`    // Destination image address
	Architecture string        `json:"architecture"` // Target architecture (e.g., "linux/amd64", "all")
	Status       SyncStatus    `json:"status"`       // Current task status
	Message      string        `json:"message"`      // Human-readable status message
	Output       string        `json:"output"`       // Complete log output (set when task completes)
	ErrorOutput  string        `json:"errorOutput"`  // Error message (if task failed)
	StartTime    time.Time     `json:"startTime"`    // Task start timestamp
	EndTime      *time.Time    `json:"endTime,omitempty"` // Task end timestamp (nil if not completed)
	LogLines     []string      `json:"-"`            // In-memory log lines (not serialized)
	LogListeners []chan string `json:"-"`            // Active log stream subscribers (SSE)
	logMu        sync.Mutex    // Mutex for thread-safe log operations
}

// NewSyncTask creates a new sync task with initial pending status.
func NewSyncTask(id, sourceImage, destImage, architecture string) *SyncTask {
	return &SyncTask{
		ID:           id,
		SourceImage:  sourceImage,
		DestImage:    destImage,
		Architecture: architecture,
		Status:       StatusPending,
		Message:      "Task created",
		StartTime:    time.Now(),
		LogLines:     []string{},
		LogListeners: []chan string{},
	}
}

// AddLog appends a log line to the task and broadcasts it to all active listeners.
// Thread-safe for concurrent access.
func (t *SyncTask) AddLog(line string) {
	t.logMu.Lock()
	defer t.logMu.Unlock()

	t.LogLines = append(t.LogLines, line)
	// Broadcast to all SSE listeners
	for _, ch := range t.LogListeners {
		select {
		case ch <- line:
			// Successfully sent
		default:
			// Channel is full or closed, skip this listener
		}
	}
}

// AddLogListener creates a new log listener channel for SSE streaming.
// Returns a buffered channel that will receive new log lines.
func (t *SyncTask) AddLogListener() chan string {
	t.logMu.Lock()
	defer t.logMu.Unlock()

	ch := make(chan string, 100)
	t.LogListeners = append(t.LogListeners, ch)
	return ch
}

// RemoveLogListener removes and closes a log listener channel.
// Should be called when an SSE client disconnects.
func (t *SyncTask) RemoveLogListener(ch chan string) {
	t.logMu.Lock()
	defer t.logMu.Unlock()

	for i, listener := range t.LogListeners {
		if listener == ch {
			t.LogListeners = append(t.LogListeners[:i], t.LogListeners[i+1:]...)
			close(ch)
			break
		}
	}
}

// CloseAllLogListeners closes all active log listener channels.
// Called when task completes to notify all SSE clients.
func (t *SyncTask) CloseAllLogListeners() {
	t.logMu.Lock()
	defer t.logMu.Unlock()

	for _, ch := range t.LogListeners {
		close(ch)
	}
	t.LogListeners = []chan string{}
}

// GetLogLines returns a copy of all log lines.
// Thread-safe for concurrent access.
func (t *SyncTask) GetLogLines() []string {
	t.logMu.Lock()
	defer t.logMu.Unlock()

	logs := make([]string, len(t.LogLines))
	copy(logs, t.LogLines)
	return logs
}

// SyncRequest represents the request body for creating a sync task.
type SyncRequest struct {
	SourceImage    string `json:"sourceImage" binding:"required"` // Source image address (required)
	DestImage      string `json:"destImage" binding:"required"`   // Destination image address (required)
	SourceUsername string `json:"sourceUsername"`                 // Source registry username (optional)
	SourcePassword string `json:"sourcePassword"`                 // Source registry password (optional)
	DestUsername   string `json:"destUsername"`                   // Destination registry username (optional)
	DestPassword   string `json:"destPassword"`                   // Destination registry password (optional)
	Architecture   string `json:"architecture"`                   // Target architecture (optional, default: "all")
	SrcTLSVerify   *bool  `json:"srcTlsVerify"`                   // Source TLS verification (optional, default: false)
	DestTLSVerify  *bool  `json:"destTlsVerify"`                  // Destination TLS verification (optional, default: false)
	RetryTimes     *int   `json:"retryTimes"`                     // Retry times for network failures (optional, default: 3)
}

// InspectRequest represents the request body for inspecting an image.
type InspectRequest struct {
	Image     string `json:"image" binding:"required"` // Image address (required)
	Username  string `json:"username"`                 // Registry username (optional)
	Password  string `json:"password"`                 // Registry password (optional)
	TLSVerify *bool  `json:"tlsVerify"`                // TLS verification (optional, default: false)
}

// InspectResponse represents the response for image inspection.
type InspectResponse struct {
	Architectures []string `json:"architectures"` // List of available architectures
}

// EnvDefaults represents default registry configuration from environment variables.
type EnvDefaults struct {
	SourceRegistry string `json:"sourceRegistry"` // Default source registry prefix
	DestRegistry   string `json:"destRegistry"`   // Default destination registry prefix
}

// TaskListRequest represents query parameters for listing tasks.
type TaskListRequest struct {
	Page      int        `form:"page,default=1"`              // Page number (default: 1)
	PageSize  int        `form:"pageSize,default=20"`         // Items per page (default: 20, max: 100)
	Status    SyncStatus `form:"status"`                      // Filter by status (optional)
	SortBy    string     `form:"sortBy,default=startTime"`    // Sort field (default: startTime)
	SortOrder string     `form:"sortOrder,default=desc"`      // Sort order: asc/desc (default: desc)
}

// TaskSummary represents a summarized view of a task (without full logs).
type TaskSummary struct {
	ID           string     `json:"id"`
	SourceImage  string     `json:"sourceImage"`
	DestImage    string     `json:"destImage"`
	Architecture string     `json:"architecture"`
	Status       SyncStatus `json:"status"`
	Message      string     `json:"message"`
	StartTime    time.Time  `json:"startTime"`
	EndTime      *time.Time `json:"endTime,omitempty"`
}

// TaskListResponse represents the response for task list queries.
type TaskListResponse struct {
	Total    int            `json:"total"`    // Total number of tasks matching filter
	Page     int            `json:"page"`     // Current page number
	PageSize int            `json:"pageSize"` // Items per page
	Tasks    []*TaskSummary `json:"tasks"`    // Task summaries for current page
}