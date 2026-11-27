package realtime

import (
	"sync"

	"github.com/gofiber/websocket/v2"
)

// Client wraps one WebSocket connection. All writes to the connection go through it:
// the websocket library allows only one concurrent writer per connection, and both the
// hub broadcast and the connection handler write to the same client.
type Client struct {
	conn *websocket.Conn
	mu   sync.Mutex // serializes writes

	// closeMu guards released. The server recycles conn once the handler returns,
	// so a Close that arrives after Release must not touch it.
	closeMu  sync.Mutex
	released bool
}

// NewClient wraps conn. The caller keeps reading from conn directly.
func NewClient(conn *websocket.Conn) *Client {
	return &Client{conn: conn}
}

// SendText writes one text message.
func (c *Client) SendText(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// SendJSON writes v as one JSON text message.
func (c *Client) SendJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteJSON(v)
}

// Close closes the connection. It does not take the write lock: closing may run
// concurrently with a write. After Release it is a no-op.
func (c *Client) Close() error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	if c.released {
		return nil
	}
	return c.conn.Close()
}

// Release marks the connection as handed back to the server. The handler calls it
// after unregistering, before it returns; a Close in progress finishes first.
func (c *Client) Release() {
	c.closeMu.Lock()
	c.released = true
	c.closeMu.Unlock()
}
