// Package logging provides structured JSONL event logging for
// all MCP gateway activity including tool calls, detection results,
// and policy decisions.
package logging

import (
	"time"
)

// Event represents a single logged gateway event.
type Event struct {
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"` // "tool_call", "detection", "policy", "error"
	Server    string                 `json:"server"`
	Data      map[string]interface{} `json:"data"`
}

// Logger writes structured events to a JSONL file.
type Logger struct {
	path string
}

// NewLogger creates a new event logger writing to the given file path.
func NewLogger(path string) *Logger {
	return &Logger{path: path}
}

// Log writes an event to the log file.
func (l *Logger) Log(event Event) error {
	// TODO: implement JSONL writing
	return nil
}

// Read returns the most recent n events from the log file.
func (l *Logger) Read(n int) ([]Event, error) {
	// TODO: implement log reading
	return nil, nil
}
