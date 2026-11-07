package sway

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync"
)

type Client struct {
	conn  net.Conn
	frame *Conn
	mu    sync.Mutex

	writeMu sync.Mutex

	pendingMu sync.Mutex
	pending   map[MessageType][]chan Message

	events []chan Message

	readerDone chan struct{}
	closed     bool
	failOnce   sync.Once
}

func Dial(ctx context.Context, address string) (*Client, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", address)
	if err != nil {
		return nil, err
	}
	c := &Client{
		conn:       conn,
		frame:      NewConn(conn),
		pending:    map[MessageType][]chan Message{},
		readerDone: make(chan struct{}),
	}
	go c.reader()
	return c, nil
}

func (c *Client) reader() {
	defer close(c.readerDone)
	for {
		m, err := c.frame.ReadMessage()
		if err != nil {
			c.failAll()
			return
		}
		if IsEvent(m.Type) {
			c.pendingMu.Lock()
			subs := append([]chan Message(nil), c.events...)
			c.pendingMu.Unlock()
			for _, ch := range subs {
				select {
				case ch <- m:
				default:
				}
			}
			continue
		}
		c.pendingMu.Lock()
		q := c.pending[m.Type]
		var ch chan Message
		if len(q) > 0 {
			ch = q[0]
			c.pending[m.Type] = q[1:]
			if len(c.pending[m.Type]) == 0 {
				delete(c.pending, m.Type)
			}
		}
		c.pendingMu.Unlock()
		if ch != nil {
			select {
			case ch <- m:
			default:
			}
		}
	}
}

func (c *Client) failAll() {
	c.failOnce.Do(func() {
		c.pendingMu.Lock()
		for _, q := range c.pending {
			for _, ch := range q {
				close(ch)
			}
		}
		c.pending = map[MessageType][]chan Message{}
		events := c.events
		c.events = nil
		c.pendingMu.Unlock()
		for _, ch := range events {
			close(ch)
		}
	})
}

func (c *Client) Close() error {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	err := c.conn.Close()
	c.failAll()
	return err
}

func (c *Client) Request(ctx context.Context, typ MessageType, payload []byte) (Message, error) {
	ch := make(chan Message, 1)
	c.pendingMu.Lock()
	c.pending[typ] = append(c.pending[typ], ch)
	c.pendingMu.Unlock()

	c.writeMu.Lock()
	err := c.frame.WriteMessage(Message{Type: typ, Payload: payload})
	c.writeMu.Unlock()
	if err != nil {
		c.pendingMu.Lock()
		// remove ch
		q := c.pending[typ]
		for i, x := range q {
			if x == ch {
				c.pending[typ] = append(q[:i], q[i+1:]...)
				break
			}
		}
		if len(c.pending[typ]) == 0 {
			delete(c.pending, typ)
		}
		c.pendingMu.Unlock()
		return Message{}, err
	}

	select {
	case reply, ok := <-ch:
		if !ok {
			return Message{}, errors.New("sway: connection closed")
		}
		return reply, nil
	case <-ctx.Done():
		c.pendingMu.Lock()
		q := c.pending[typ]
		for i, x := range q {
			if x == ch {
				c.pending[typ] = append(q[:i], q[i+1:]...)
				break
			}
		}
		if len(c.pending[typ]) == 0 {
			delete(c.pending, typ)
		}
		c.pendingMu.Unlock()
		return Message{}, ctx.Err()
	}
}

// convenience methods unchanged
func (c *Client) RunCommand(ctx context.Context, command string) (Message, error) {
	m, err := c.Request(ctx, RunCommand, []byte(command))
	if err != nil {
		return Message{}, err
	}
	var replies []struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(m.Payload, &replies); err == nil && len(replies) > 0 && !replies[0].Success {
		if replies[0].Error != "" {
			return m, errors.New(replies[0].Error)
		}
		return m, errors.New("sway: command failed")
	}
	return m, nil
}
func (c *Client) GetWorkspaces(ctx context.Context) ([]Workspace, error) {
	reply, err := c.Request(ctx, GetWorkspaces, nil)
	if err != nil {
		return nil, err
	}
	return UnmarshalWorkspaces(reply.Payload)
}
func (c *Client) GetInputs(ctx context.Context) ([]Input, error) {
	reply, err := c.Request(ctx, GetInputs, nil)
	if err != nil {
		return nil, err
	}
	return UnmarshalInputs(reply.Payload)
}

func (c *Client) GetOutputs(ctx context.Context) ([]Output, error) {
	reply, err := c.Request(ctx, GetOutputs, nil)
	if err != nil {
		return nil, err
	}
	// sway's get_outputs IPC may report {"success": false}; fall back to the
	// output nodes reported by get_tree, which carries all output fields.
	var succ struct {
		Success bool `json:"success"`
	}
	if json.Unmarshal(reply.Payload, &succ) == nil && !succ.Success {
		return outputsFromTree(c, ctx)
	}
	return UnmarshalOutputs(reply.Payload)
}

func outputsFromTree(c *Client, ctx context.Context) ([]Output, error) {
	reply, err := c.Request(ctx, GetTree, nil)
	if err != nil {
		return nil, err
	}
	var nodes []Node
	if err := json.Unmarshal(reply.Payload, &nodes); err != nil {
		return nil, err
	}
	var outs []Output
	for i := range nodes {
		if nodes[i].Type != "output" {
			continue
		}
		n := &nodes[i]
		outs = append(outs, Output{
			ID:               n.ID,
			Name:             n.Name,
			Active:           n.Active,
			CurrentWorkspace: n.CurrentWorkspace,
			Rect:             n.Rect,
			Scale:            n.Scale,
			Transform:        n.Transform,
			Make:             n.Make,
			Model:            n.Model,
			Serial:           n.Serial,
		})
	}
	return outs, nil
}

func (c *Client) GetTree(ctx context.Context) (*Node, error) {
	reply, err := c.Request(ctx, GetTree, nil)
	if err != nil {
		return nil, err
	}
	return UnmarshalTree(reply.Payload)
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
	ch := make(chan Message, 16)
	c.pendingMu.Lock()
	c.events = append(c.events, ch)
	c.pendingMu.Unlock()
	reply, err := c.Request(ctx, Subscribe, payload)
	if err != nil {
		return nil, err
	}
	var sr subscribeReply
	if err := json.Unmarshal(reply.Payload, &sr); err != nil || !sr.Success {
		return nil, errors.New("i3-ipc: subscribe failed")
	}
	return ch, nil
}
