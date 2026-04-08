// Package auth implements the device authentication flow for
// connecting the gateway to Clawkeeper Cloud.
package auth

// Credentials holds the stored authentication state.
type Credentials struct {
	DeviceID    string `json:"deviceId"`
	AccessToken string `json:"accessToken"`
	OrgID       string `json:"orgId"`
}

// Client manages device authentication with Clawkeeper Cloud.
type Client struct {
	baseURL string
}

// NewClient creates an auth client pointing at the given API base URL.
func NewClient(baseURL string) *Client {
	return &Client{baseURL: baseURL}
}

// Login initiates the device auth flow by opening the browser.
func (c *Client) Login() (*Credentials, error) {
	// TODO: implement device auth flow
	return nil, nil
}

// Status checks whether stored credentials are valid.
func (c *Client) Status() (*Credentials, error) {
	// TODO: implement credential validation
	return nil, nil
}

// Logout removes stored credentials.
func (c *Client) Logout() error {
	// TODO: implement credential removal
	return nil
}
