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
	split2 := func(s, sep string) (string, string) {
		parts := strings.SplitN(s, sep, 2)
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
		return parts[0], ""
	}
	parsePrefix := func(l string) (head string, tail string) {
		if l == "" || l[0] != ':' {
			return "", l
		}

		head, tail = split2(l, " ")
		return head[1:], tail
	}
	parseCommand := func(l string) (res string, tail string) {
		return split2(l, " ")
	}
	parseParams := func(l string) (res []string, tail string) {
		if l == "" || l[0] == ':' {
			return []string{}, l
		}

		head, tail := split2(l, " :")
		return strings.Split(head, " "), tail
	}

	msg := &Message{Raw: l}
	msg.Prefix, l = parsePrefix(l)
	msg.Command, l = parseCommand(l)
	msg.Params, l = parseParams(l)
	if l != "" && l[0] == ':' {
		l = l[1:]
	}
	msg.Txt = l
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
