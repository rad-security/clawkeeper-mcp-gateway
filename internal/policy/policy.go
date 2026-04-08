// Package policy implements the policy engine that evaluates MCP
// tool calls against user-defined rules to determine allow/deny
// decisions.
package policy

// Rule defines a single policy rule.
type Rule struct {
	Name       string   `json:"name"`
	Action     string   `json:"action"` // "allow", "deny", "warn"
	Tools      []string `json:"tools"`  // tool name patterns
	Conditions []string `json:"conditions,omitempty"`
}

// Policy is a collection of rules evaluated in order.
type Policy struct {
	Rules []Rule
}

// NewPolicy creates an empty policy.
func NewPolicy() *Policy {
	return &Policy{}
}

// Evaluate checks a tool call against all policy rules and returns
// the action to take.
func (p *Policy) Evaluate(toolName string, params map[string]interface{}) string {
	// TODO: implement policy evaluation
	return "allow"
}

// LoadFromFile reads policy rules from a YAML or JSON file.
func LoadFromFile(path string) (*Policy, error) {
	// TODO: implement policy loading
	return NewPolicy(), nil
}
