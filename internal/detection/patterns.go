package detection

// Pattern defines a named detection rule with a regex or matching function.
type Pattern struct {
	Name        string
	Severity    string // "critical", "high", "medium"
	Description string
	Regex       string
}

// BuiltinPatterns returns the default set of detection patterns.
func BuiltinPatterns() []Pattern {
	// TODO: populate with prompt injection, tool shadowing,
	// cross-origin escalation, and data exfiltration patterns
	return []Pattern{}
}
