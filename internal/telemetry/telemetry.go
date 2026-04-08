// Package telemetry handles batched upload of anonymized usage
// metrics to Clawkeeper Cloud for authenticated users.
package telemetry

// Client uploads telemetry data to Clawkeeper Cloud.
type Client struct {
	endpoint string
	apiKey   string
	enabled  bool
}

// NewClient creates a telemetry client.
func NewClient(endpoint string, apiKey string) *Client {
	return &Client{
		endpoint: endpoint,
		apiKey:   apiKey,
		enabled:  apiKey != "",
	}
}

// Send uploads a batch of events to the telemetry endpoint.
func (c *Client) Send(events []map[string]interface{}) error {
	if !c.enabled {
		return nil
	}
	// TODO: implement batch upload
	return nil
}
