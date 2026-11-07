package sway

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"time"
)

type Client struct {
	conn  net.Conn
	frame *Conn
	mu    sync.Mutex
}

func Dial(ctx context.Context, address string) (*Client, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", address)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, frame: NewConn(conn)}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Request(ctx context.Context, typ MessageType, payload []byte) (Message, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.setDeadline(ctx); err != nil {
		return Message{}, err
	}
	defer c.clearDeadline()
	if err := c.frame.WriteMessage(Message{Type: typ, Payload: payload}); err != nil {
		return Message{}, err
	}
	reply, err := c.frame.ReadMessage()
	if err != nil {
		return Message{}, err
	}
	return reply, nil
}

func (c *Client) setDeadline(ctx context.Context) error {
	if dl, ok := ctx.Deadline(); ok && !dl.IsZero() {
		return c.conn.SetDeadline(dl)
	}
	return nil
}

func (c *Client) clearDeadline() {
	_ = c.conn.SetDeadline(time.Time{})
}

func (c *Client) RunCommand(ctx context.Context, command string) (Message, error) {
	return c.Request(ctx, RunCommand, []byte(command))
}

func (c *Client) GetWorkspaces(ctx context.Context) (Message, error) {
	return c.Request(ctx, GetWorkspaces, nil)
}

func (c *Client) GetOutputs(ctx context.Context) (Message, error) {
	return c.Request(ctx, GetOutputs, nil)
}

func (c *Client) GetTree(ctx context.Context) (Message, error) {
	return c.Request(ctx, GetTree, nil)
}

func (c *Client) GetInputs(ctx context.Context) (Message, error) {
	return c.Request(ctx, GetInputs, nil)
}

func (c *Client) GetVersion(ctx context.Context) (*Version, error) {
	reply, err := c.Request(ctx, GetVersion, nil)
	if err != nil {
		return nil, err
	}
	var v Version
	if err := json.Unmarshal(reply.Payload, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

type subscribePayload struct {
	Events []string `json:"events"`
}

type subscribeReply struct {
	Success bool `json:"success"`
}

func (c *Client) Subscribe(ctx context.Context, events []string) (<-chan Message, error) {
	payload, err := json.Marshal(subscribePayload{Events: events})
	if err != nil {
		return nil, err
	}
	reply, err := c.Request(ctx, Subscribe, payload)
	if err != nil {
		return nil, err
	}
	var sr subscribeReply
	if err := json.Unmarshal(reply.Payload, &sr); err != nil || !sr.Success {
		return nil, errors.New("i3-ipc: subscribe failed")
	}
	ch := make(chan Message, 16)
	go func() {
		defer close(ch)
		for {
			m, err := c.frame.ReadMessage()
			if err != nil {
				return
			}
			if !IsEvent(m.Type) {
				continue
			}
			select {
			case ch <- m:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}
