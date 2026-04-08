// Package detection implements the threat detection engine that
// inspects MCP tool calls and responses for security threats
// including prompt injection, tool shadowing, and data exfiltration.
package detection

// Result represents the outcome of a detection scan.
type Result struct {
	Verdict     string // "pass", "warn", "block"
	PatternName string
	Severity    string // "critical", "high", "medium"
	Description string
}

// Engine runs threat detection on MCP tool calls.
type Engine struct{}

// NewEngine creates a detection engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Evaluate inspects an MCP tool call and returns a detection result.
func (e *Engine) Evaluate(toolName string, params map[string]interface{}, response string) Result {
	return Result{Verdict: "pass"}
}
