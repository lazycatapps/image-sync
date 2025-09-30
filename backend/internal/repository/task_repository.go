// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package repository provides data access layer for sync tasks.
package repository

import (
	"errors"
	"sync"

	"github.com/lazycatapps/image-sync/internal/models"
)

var (
	// ErrTaskNotFound is returned when a requested task does not exist.
	ErrTaskNotFound = errors.New("task not found")
)

// TaskRepository defines the interface for task persistence operations.
type TaskRepository interface {
	Create(task *models.SyncTask) error
	Get(id string) (*models.SyncTask, error)
	Update(task *models.SyncTask) error
	Delete(id string) error
	List() ([]*models.SyncTask, error)
}

// InMemoryTaskRepository implements TaskRepository using in-memory storage.
// It uses a map for storage and a RWMutex for thread-safe access.
// Note: All data is lost when the process restarts.
type InMemoryTaskRepository struct {
	tasks map[string]*models.SyncTask
	mu    sync.RWMutex
}

// NewInMemoryTaskRepository creates a new in-memory task repository.
func NewInMemoryTaskRepository() *InMemoryTaskRepository {
	return &InMemoryTaskRepository{
		tasks: make(map[string]*models.SyncTask),
	}
}

// Create adds a new task to the repository.
func (r *InMemoryTaskRepository) Create(task *models.SyncTask) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tasks[task.ID] = task
	return nil
}

// Get retrieves a task by ID.
func (r *InMemoryTaskRepository) Get(id string) (*models.SyncTask, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, exists := r.tasks[id]
	if !exists {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

// Update modifies an existing task.
func (r *InMemoryTaskRepository) Update(task *models.SyncTask) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[task.ID]; !exists {
		return ErrTaskNotFound
	}
	r.tasks[task.ID] = task
	return nil
}

// Delete removes a task from the repository.
func (r *InMemoryTaskRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[id]; !exists {
		return ErrTaskNotFound
	}
	delete(r.tasks, id)
	return nil
}

// List returns all tasks in the repository.
func (r *InMemoryTaskRepository) List() ([]*models.SyncTask, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tasks := make([]*models.SyncTask, 0, len(r.tasks))
	for _, task := range r.tasks {
		tasks = append(tasks, task)
	}
	return tasks, nil
}