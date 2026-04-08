package telemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rad-security/clawkeeper-mcp-gateway/internal/logging"
)

// Client handles batch event upload to the Clawkeeper API.
type Client struct {
	apiURL   string
	apiKey   string
	hostname string
	logger   *logging.Logger
	done     chan struct{}
}

// NewClient creates a telemetry client.
func NewClient(apiURL, apiKey string, logger *logging.Logger) *Client {
	hostname, _ := os.Hostname()
	return &Client{
		apiURL:   apiURL,
		apiKey:   apiKey,
		hostname: hostname,
		logger:   logger,
		done:     make(chan struct{}),
	}
}

// Start begins the background flush loop (every 5 seconds).
func (c *Client) Start() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.flush()
			case <-c.done:
				c.flush() // Final flush
				return
			}
		}
	}()
}

// Stop signals the flush loop to stop.
func (c *Client) Stop() {
	close(c.done)
}

func (c *Client) flush() {
	events := c.logger.FlushBuffer()
	if len(events) == 0 {
		return
	}

	payload := map[string]interface{}{
		"events":   events,
		"hostname": c.hostname,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", c.apiURL+"/api/v1/mcp/events", bytes.NewReader(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// Network error — events stay in local JSONL (already written by logger)
		fmt.Fprintf(os.Stderr, "[clawkeeper] telemetry upload failed: %v\n", err)
		return
	}
	resp.Body.Close()
}
