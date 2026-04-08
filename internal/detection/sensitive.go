package detection

// SensitiveDataScanner detects PII, secrets, and other sensitive
// data in MCP tool call parameters and responses.
type SensitiveDataScanner struct{}

// NewSensitiveDataScanner creates a sensitive data scanner.
func NewSensitiveDataScanner() *SensitiveDataScanner {
	return &SensitiveDataScanner{}
}

// Scan checks the given text for sensitive data patterns and returns
// any matches found.
func (s *SensitiveDataScanner) Scan(text string) []Result {
	// TODO: implement PII and secret detection
	return nil
}
