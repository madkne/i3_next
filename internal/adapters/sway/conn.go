package sway

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
)

type Conn struct {
	conn net.Conn
	buf  bytes.Buffer
}

func NewConn(conn net.Conn) *Conn {
	return &Conn{conn: conn}
}

func (c *Conn) WriteMessage(m Message) error {
	data, err := MarshalMessage(m)
	if err != nil {
		return err
	}
	_, err = c.conn.Write(data)
	return err
}

func (c *Conn) ReadMessage() (Message, error) {
	if err := c.fill(frameSize); err != nil {
		return Message{}, err
	}
	data := c.buf.Bytes()
	if string(data[:magicSize]) != magic {
		return Message{}, errors.New("i3-ipc: bad magic")
	}
	length, typ, err := unmarshalHeader(data[magicSize:frameSize])
	if err != nil {
		return Message{}, err
	}
	if length > maxPayload {
		return Message{}, fmt.Errorf("i3-ipc: payload too large: %d bytes (max %d)", length, maxPayload)
	}
	if err := c.fill(frameSize + int(length)); err != nil {
		return Message{}, err
	}
	data = c.buf.Bytes()
	m := Message{
		Type:    typ,
		Payload: append([]byte(nil), data[frameSize:frameSize+int(length)]...),
	}
	c.buf.Next(frameSize + int(length))
	return m, nil
}

func (c *Conn) fill(n int) error {
	for c.buf.Len() < n {
		tmp := make([]byte, 256)
		nn, err := c.conn.Read(tmp)
		if nn > 0 {
			c.buf.Write(tmp[:nn])
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				if c.buf.Len() > 0 {
					return io.ErrUnexpectedEOF
				}
				return io.EOF
			}
			return err
		}
		if nn == 0 {
			return io.ErrUnexpectedEOF
		}
	}
	return nil
}
