// Copyright (c) 2025 Lazycat Apps
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package logger provides structured logging interfaces for the Image Sync application.
package logger

import (
	"log"
	"os"
)

// Logger defines the logging interface with three severity levels.
type Logger interface {
	Info(format string, args ...interface{})  // Informational messages
	Error(format string, args ...interface{}) // Error messages
	Debug(format string, args ...interface{}) // Debug messages
}

// StandardLogger implements the Logger interface using Go's standard log package.
// It outputs INFO and DEBUG logs to stdout, and ERROR logs to stderr.
type StandardLogger struct {
	infoLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
}

// New creates a new StandardLogger instance with predefined log formats.
// Log format: [LEVEL] timestamp message
func New() *StandardLogger {
	return &StandardLogger{
		infoLogger:  log.New(os.Stdout, "[INFO] ", log.LstdFlags),
		errorLogger: log.New(os.Stderr, "[ERROR] ", log.LstdFlags),
		debugLogger: log.New(os.Stdout, "[DEBUG] ", log.LstdFlags),
	}
}

// Info logs an informational message to stdout.
func (l *StandardLogger) Info(format string, args ...interface{}) {
	l.infoLogger.Printf(format, args...)
}

// Error logs an error message to stderr.
func (l *StandardLogger) Error(format string, args ...interface{}) {
	l.errorLogger.Printf(format, args...)
}

// Debug logs a debug message to stdout.
func (l *StandardLogger) Debug(format string, args ...interface{}) {
	l.debugLogger.Printf(format, args...)
}