// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

package repository

import (
	"testing"

	"github.com/lazycatapps/image-sync/internal/models"
)

func TestInMemoryTaskRepository_Create(t *testing.T) {
	repo := NewInMemoryTaskRepository()
	task := models.NewSyncTask("test-id", "src", "dest", "all")

	err := repo.Create(task)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	retrieved, err := repo.Get("test-id")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if retrieved.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got '%s'", retrieved.ID)
	}
}

func TestInMemoryTaskRepository_Get_NotFound(t *testing.T) {
	repo := NewInMemoryTaskRepository()

	_, err := repo.Get("non-existent")
	if err != ErrTaskNotFound {
		t.Errorf("Expected ErrTaskNotFound, got %v", err)
	}
}

func TestInMemoryTaskRepository_Update(t *testing.T) {
	repo := NewInMemoryTaskRepository()
	task := models.NewSyncTask("test-id", "src", "dest", "all")

	repo.Create(task)

	task.Status = models.StatusCompleted
	task.Message = "Done"

	err := repo.Update(task)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	retrieved, _ := repo.Get("test-id")
	if retrieved.Status != models.StatusCompleted {
		t.Errorf("Expected status 'completed', got '%s'", retrieved.Status)
	}
	if retrieved.Message != "Done" {
		t.Errorf("Expected message 'Done', got '%s'", retrieved.Message)
	}
}

func TestInMemoryTaskRepository_Delete(t *testing.T) {
	repo := NewInMemoryTaskRepository()
	task := models.NewSyncTask("test-id", "src", "dest", "all")

	repo.Create(task)

	err := repo.Delete("test-id")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	_, err = repo.Get("test-id")
	if err != ErrTaskNotFound {
		t.Errorf("Expected ErrTaskNotFound after delete, got %v", err)
	}
}

func TestInMemoryTaskRepository_List(t *testing.T) {
	repo := NewInMemoryTaskRepository()

	task1 := models.NewSyncTask("id1", "src1", "dest1", "all")
	task2 := models.NewSyncTask("id2", "src2", "dest2", "linux/amd64")

	repo.Create(task1)
	repo.Create(task2)

	tasks, err := repo.List()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}
}