// Package logging provides structured JSONL event logging for
// all MCP gateway activity including tool calls, detection results,
// and policy decisions.
package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Event represents a single logged gateway event.
type Event struct {
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"` // "tool_call", "detection", "policy", "error"
	Server    string                 `json:"server"`
	Data      map[string]interface{} `json:"data"`
}

// DetectionResult is a minimal interface for detection results
// so that the logging package does not import detection directly.
type DetectionResult interface {
	GetVerdict() string
	GetPatternName() string
	GetSeverity() string
	GetDescription() string
	GetCategory() string
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
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = f.Write(data)
	return err
}

// Read returns the most recent n events from the log file.
func (l *Logger) Read(n int) ([]Event, error) {
	// TODO: implement log reading
	return nil, nil
}

// Warn logs a warning message to stderr and to the event log.
func (l *Logger) Warn(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "[WARN] %s\n", msg)
	l.Log(Event{
		Timestamp: time.Now(),
		Type:      "warning",
		Data:      map[string]interface{}{"message": msg},
	})
}

// LogDetection logs a detection result for a given server and tool.
func (l *Logger) LogDetection(serverName, toolName string, result interface{}) {
	data := map[string]interface{}{
		"server": serverName,
		"tool":   toolName,
	}

	// Use type assertion to extract fields from detection.Result
	// without importing the detection package directly.
	if r, ok := result.(interface{ GetVerdict() string }); ok {
		data["verdict"] = r.GetVerdict()
	}
	if r, ok := result.(interface{ GetPatternName() string }); ok {
		data["pattern"] = r.GetPatternName()
	}
	if r, ok := result.(interface{ GetSeverity() string }); ok {
		data["severity"] = r.GetSeverity()
	}
	if r, ok := result.(interface{ GetDescription() string }); ok {
		data["description"] = r.GetDescription()
	}
	if r, ok := result.(interface{ GetCategory() string }); ok {
		data["category"] = r.GetCategory()
	}

	// Also handle the concrete struct via map conversion as fallback
	if b, err := json.Marshal(result); err == nil {
		var m map[string]interface{}
		if json.Unmarshal(b, &m) == nil {
			for k, v := range m {
				if _, exists := data[k]; !exists {
					data[k] = v
				}
			}
		}
	}

	fmt.Fprintf(os.Stderr, "[DETECTION] server=%s tool=%s result=%v\n", serverName, toolName, data)
	l.Log(Event{
		Timestamp: time.Now(),
		Type:      "detection",
		Server:    serverName,
		Data:      data,
	})
}

// LogToolCall logs a tool call and its detection result.
func (l *Logger) LogToolCall(serverName, toolName string, arguments map[string]interface{}, result interface{}) {
	data := map[string]interface{}{
		"server":    serverName,
		"tool":      toolName,
		"arguments": arguments,
	}

	// Extract detection result fields via JSON round-trip
	if b, err := json.Marshal(result); err == nil {
		var m map[string]interface{}
		if json.Unmarshal(b, &m) == nil {
			data["detection"] = m
		}
	}

	l.Log(Event{
		Timestamp: time.Now(),
		Type:      "tool_call",
		Server:    serverName,
		Data:      data,
	})
}
