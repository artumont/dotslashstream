package redis

import (
	"context"
	"time"
)

// Task represents the payload for an async job
type Task struct {
	Type    string
	Payload []byte
}

// QueueClient is injected into your HTTP handlers to dispatch background jobs.
type QueueClient interface {
	// Enqueue pushes a task onto the queue for background processing.
	// The delay determines how long to wait before the task becomes available for execution.
	Enqueue(ctx context.Context, task *Task, delay time.Duration) error

	// Close gracefully shuts down the client and releases any underlying resources.
	Close() error
}
