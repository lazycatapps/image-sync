// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

package service

import (
	"testing"

	"github.com/lazycatapps/image-sync/internal/models"
	"github.com/lazycatapps/image-sync/internal/pkg/logger"
	"github.com/lazycatapps/image-sync/internal/repository"
)

func TestCreateSyncTask(t *testing.T) {
	repo := repository.NewInMemoryTaskRepository()
	log := logger.New()
	service := NewSyncService(repo, log, 600)

	req := &models.SyncRequest{
		SourceImage: "docker.io/library/nginx:latest",
		DestImage:   "registry.example.com/nginx:latest",
	}

	taskID, err := service.CreateSyncTask(req)
	if err != nil {
		t.Fatalf("CreateSyncTask failed: %v", err)
	}

	if taskID == "" {
		t.Fatal("Expected non-empty task ID")
	}

	task, err := repo.Get(taskID)
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}

	if task.SourceImage != req.SourceImage {
		t.Errorf("Expected source image %s, got %s", req.SourceImage, task.SourceImage)
	}

	if task.DestImage != req.DestImage {
		t.Errorf("Expected dest image %s, got %s", req.DestImage, task.DestImage)
	}

	if task.Architecture != "all" {
		t.Errorf("Expected architecture 'all', got %s", task.Architecture)
	}

	if task.Status != models.StatusPending {
		t.Errorf("Expected status pending, got %s", task.Status)
	}
}

func TestCreateSyncTaskWithArchitecture(t *testing.T) {
	repo := repository.NewInMemoryTaskRepository()
	log := logger.New()
	service := NewSyncService(repo, log, 600)

	req := &models.SyncRequest{
		SourceImage:  "docker.io/library/nginx:latest",
		DestImage:    "registry.example.com/nginx:latest",
		Architecture: "linux/amd64",
	}

	taskID, err := service.CreateSyncTask(req)
	if err != nil {
		t.Fatalf("CreateSyncTask failed: %v", err)
	}

	task, err := repo.Get(taskID)
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}

	if task.Architecture != "linux/amd64" {
		t.Errorf("Expected architecture 'linux/amd64', got %s", task.Architecture)
	}
}

func TestGetTask(t *testing.T) {
	repo := repository.NewInMemoryTaskRepository()
	log := logger.New()
	service := NewSyncService(repo, log, 600)

	req := &models.SyncRequest{
		SourceImage: "docker.io/library/nginx:latest",
		DestImage:   "registry.example.com/nginx:latest",
	}

	taskID, _ := service.CreateSyncTask(req)

	task, err := service.GetTask(taskID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if task.ID != taskID {
		t.Errorf("Expected task ID %s, got %s", taskID, task.ID)
	}
}

func TestGetTaskNotFound(t *testing.T) {
	repo := repository.NewInMemoryTaskRepository()
	log := logger.New()
	service := NewSyncService(repo, log, 600)

	_, err := service.GetTask("non-existent-id")
	if err != repository.ErrTaskNotFound {
		t.Errorf("Expected ErrTaskNotFound, got %v", err)
	}
}

func TestListTasks(t *testing.T) {
	repo := repository.NewInMemoryTaskRepository()
	log := logger.New()
	service := NewSyncService(repo, log, 600)

	// Create multiple tasks
	for i := 0; i < 5; i++ {
		req := &models.SyncRequest{
			SourceImage: "docker.io/library/nginx:latest",
			DestImage:   "registry.example.com/nginx:latest",
		}
		service.CreateSyncTask(req)
	}

	listReq := &models.TaskListRequest{
		Page:     1,
		PageSize: 10,
	}

	resp, err := service.ListTasks(listReq)
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}

	if resp.Total != 5 {
		t.Errorf("Expected total 5, got %d", resp.Total)
	}

	if len(resp.Tasks) != 5 {
		t.Errorf("Expected 5 tasks, got %d", len(resp.Tasks))
	}
}

func TestListTasksWithPagination(t *testing.T) {
	repo := repository.NewInMemoryTaskRepository()
	log := logger.New()
	service := NewSyncService(repo, log, 600)

	// Create 25 tasks
	for i := 0; i < 25; i++ {
		req := &models.SyncRequest{
			SourceImage: "docker.io/library/nginx:latest",
			DestImage:   "registry.example.com/nginx:latest",
		}
		service.CreateSyncTask(req)
	}

	listReq := &models.TaskListRequest{
		Page:     2,
		PageSize: 10,
	}

	resp, err := service.ListTasks(listReq)
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}

	if resp.Total != 25 {
		t.Errorf("Expected total 25, got %d", resp.Total)
	}

	if len(resp.Tasks) != 10 {
		t.Errorf("Expected 10 tasks on page 2, got %d", len(resp.Tasks))
	}
}

func TestListTasksFilterByStatus(t *testing.T) {
	repo := repository.NewInMemoryTaskRepository()
	log := logger.New()
	service := NewSyncService(repo, log, 600)

	// Create tasks with different statuses
	for i := 0; i < 3; i++ {
		req := &models.SyncRequest{
			SourceImage: "docker.io/library/nginx:latest",
			DestImage:   "registry.example.com/nginx:latest",
		}
		taskID, _ := service.CreateSyncTask(req)
		task, _ := repo.Get(taskID)
		if i == 0 {
			task.Status = models.StatusCompleted
			repo.Update(task)
		}
	}

	listReq := &models.TaskListRequest{
		Status: models.StatusPending,
	}

	resp, err := service.ListTasks(listReq)
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}

	if resp.Total != 2 {
		t.Errorf("Expected 2 pending tasks, got %d", resp.Total)
	}
}
