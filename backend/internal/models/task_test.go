// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

package models

import (
	"testing"
	"time"
)

func TestNewSyncTask(t *testing.T) {
	task := NewSyncTask("test-id", "nginx:latest", "registry.example.com/nginx:latest", "all")

	if task.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got '%s'", task.ID)
	}

	if task.SourceImage != "nginx:latest" {
		t.Errorf("Expected SourceImage 'nginx:latest', got '%s'", task.SourceImage)
	}

	if task.Status != StatusPending {
		t.Errorf("Expected status 'pending', got '%s'", task.Status)
	}

	if task.Message != "Task created" {
		t.Errorf("Expected message 'Task created', got '%s'", task.Message)
	}

	if len(task.LogLines) != 0 {
		t.Errorf("Expected empty LogLines, got %d items", len(task.LogLines))
	}
}

func TestSyncTask_AddLog(t *testing.T) {
	task := NewSyncTask("test-id", "src", "dest", "all")

	task.AddLog("First log")
	task.AddLog("Second log")

	logs := task.GetLogLines()
	if len(logs) != 2 {
		t.Errorf("Expected 2 log lines, got %d", len(logs))
	}

	if logs[0] != "First log" {
		t.Errorf("Expected first log 'First log', got '%s'", logs[0])
	}

	if logs[1] != "Second log" {
		t.Errorf("Expected second log 'Second log', got '%s'", logs[1])
	}
}

func TestSyncTask_AddLogListener(t *testing.T) {
	task := NewSyncTask("test-id", "src", "dest", "all")

	ch := task.AddLogListener()
	if ch == nil {
		t.Error("Expected non-nil channel")
	}

	go func() {
		task.AddLog("Test message")
	}()

	select {
	case msg := <-ch:
		if msg != "Test message" {
			t.Errorf("Expected 'Test message', got '%s'", msg)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for log message")
	}

	task.RemoveLogListener(ch)
}

func TestSyncTask_CloseAllLogListeners(t *testing.T) {
	task := NewSyncTask("test-id", "src", "dest", "all")

	ch1 := task.AddLogListener()
	ch2 := task.AddLogListener()

	task.CloseAllLogListeners()

	_, ok1 := <-ch1
	_, ok2 := <-ch2

	if ok1 || ok2 {
		t.Error("Expected all channels to be closed")
	}
}