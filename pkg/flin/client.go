package flin

import (
	"fmt"

	"github.com/skshohagmiah/flin/pkg/client"
	"github.com/skshohagmiah/flin/pkg/queue"
)

// Client is the unified Flin client
type Client struct {
	KV    *client.TCPClient
	Queue *queue.Client
}

// NewClient creates a new unified Flin client
func NewClient(kvAddr string, queueStoragePath string) (*Client, error) {
	// Create KV client
	kvClient, err := client.NewTCP(kvAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create KV client: %w", err)
	}

	// Create Queue client
	queueClient, err := queue.NewClient(queueStoragePath)
	if err != nil {
		kvClient.Close()
		return nil, fmt.Errorf("failed to create queue client: %w", err)
	}

	return &Client{
		KV:    kvClient,
		Queue: queueClient,
	}, nil
}

// Close closes the KV client
func (c *Client) Close() error {
	if c.KV != nil {
		return c.KV.Close()
	}
	return nil
}
