package irc

import (
	"bufio"
	"io"
	"net"
	"strings"
	"sync"
)

// An IRC message in the format:
//   [:Prefix] Command [ { Param } ] [:Txt]
type Message struct {
	Command string
	Params  []string
	Prefix  string
	Raw     string
	User    string
	Txt     string
}

// Parses an incoming IRC message in the format:
//   [:Prefix] Command [ { Param } ] [:Txt]
func ParseMessage(l string) *Message {
	msg := &Message{Raw: l}
	if l[0] == ':' {
		if i := strings.Index(l, " "); i > -1 {
			msg.Prefix = l[1:i]
			l = l[i+1 : len(l)]
		}
		if i := strings.Index(msg.Prefix, "!"); i > -1 {
			msg.User = msg.Prefix[i+1 : strings.Index(msg.Prefix, "@")]
		}
	}
	parts := strings.SplitN(l, " ", 2)
	msg.Command = parts[0]
	if len(parts) == 1 {
		return msg
	}

	l = parts[1]
	if l[0] != ':' {
		parts := strings.SplitN(l, " :", 2)
		msg.Params = strings.Split(parts[0], " ")
		if len(parts) == 1 {
			return msg
		}
		l = parts[1]
	}
	if len(l) > 0 {
		if l[0] == ':' && len(l) > 1 {
			msg.Txt = l[1:]
		} else {
			msg.Txt = l
		}
	}
	return msg
}

// IRC connection struct
type Connection struct {
	SSL bool

	rwc    io.ReadWriteCloser
	reader *bufio.Reader

	rl sync.Mutex
	wl sync.Mutex
}

// Establish a connection
func (c *Connection) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	c.rwc = conn
	c.reader = bufio.NewReader(c.rwc)
	return nil
}

// Reads the next message
func (c *Connection) ReadMsg() (*Message, error) {
	c.rl.Lock()
	defer c.rl.Unlock()
	l, err := c.reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	return ParseMessage(l), nil
}

// Writes Raw bytes
func (c *Connection) Write(msg []byte) (int, error) {
	c.wl.Lock()
	defer c.wl.Unlock()

	return c.rwc.Write(msg)
}

// Writes string. "\r\n" will be appended
func (c *Connection) Send(l string) error {
	_, err := c.Write([]byte(l + "\r\n"))
	return err
}

// Closes connection
func (c *Connection) Close() error {
	return c.rwc.Close()
}
