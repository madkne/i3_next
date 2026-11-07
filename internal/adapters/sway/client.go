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
	conn   net.Conn
	frame  *Conn
	mu     sync.Mutex

	writeMu sync.Mutex

	pendingMu sync.Mutex
	pending   map[MessageType]chan Message

	eventCh chan Message
	events  []chan Message

	readerDone chan struct{}
	closed     bool
}

func Dial(ctx context.Context, address string) (*Client, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", address)
	if err != nil {
		return nil, err
	}
	c := &Client{
		conn:      conn,
		frame:     NewConn(conn),
		pending:   map[MessageType]chan Message{},
		eventCh:   make(chan Message, 64),
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
			c.failAll(err)
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
		ch, ok := c.pending[m.Type]
		if ok {
			delete(c.pending, m.Type)
		}
		c.pendingMu.Unlock()
		if ok {
			select {
			case ch <- m:
			default:
			}
		}
	}
}

func (c *Client) failAll(err error) {
	c.pendingMu.Lock()
	for typ, ch := range c.pending {
		select {
		case ch <- Message{}:
		default:
		}
		delete(c.pending, typ)
	}
	c.pendingMu.Unlock()
}

func (c *Client) Close() error {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	return c.conn.Close()
}

func (c *Client) Request(ctx context.Context, typ MessageType, payload []byte) (Message, error) {
	if err := c.setDeadline(ctx); err != nil {
		return Message{}, err
	}
	defer c.clearDeadline()

	ch := make(chan Message, 1)
	c.pendingMu.Lock()
	c.pending[typ] = ch
	c.pendingMu.Unlock()

	c.writeMu.Lock()
	err := c.frame.WriteMessage(Message{Type: typ, Payload: payload})
	c.writeMu.Unlock()
	if err != nil {
		c.pendingMu.Lock()
		delete(c.pending, typ)
		c.pendingMu.Unlock()
		return Message{}, err
	}

	select {
	case reply := <-ch:
		if reply.Type == 0 && reply.Payload == nil {
			// could be failAll sentinel; check conn
		}
		return reply, nil
	case <-ctx.Done():
		c.pendingMu.Lock()
		delete(c.pending, typ)
		c.pendingMu.Unlock()
		return Message{}, ctx.Err()
	}
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

// convenience methods unchanged
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
