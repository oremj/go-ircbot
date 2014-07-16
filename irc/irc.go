package irc

import (
	"bufio"
	"crypto/tls"
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
	Prefix  *Prefix
	Raw     string
	Txt     string
}

type Prefix struct {
	Name string
	User string
	Host string
}

// Parses an incoming IRC message in the format:
//    message    =  [ ":" prefix SPACE ] command [ params ] crlf
//    prefix     =  servername / ( nickname [ [ "!" user ] "@" host ] )
//    command    =  1*letter / 3digit
//    params     =  *14( SPACE middle ) [ SPACE ":" trailing ]
//               =/ 14( SPACE middle ) [ SPACE [ ":" ] trailing ]
//
//    nospcrlfcl =  %x01-09 / %x0B-0C / %x0E-1F / %x21-39 / %x3B-FF
//                    ; any octet except NUL, CR, LF, " " and ":"
//    middle     =  nospcrlfcl *( ":" / nospcrlfcl )
//    trailing   =  *( ":" / " " / nospcrlfcl )
//
//    SPACE      =  %x20        ; space character
//    crlf       =  %x0D %x0A   ; "carriage return" "linefeed"
func ParseMessage(l string) *Message {
	split2 := func(s, sep string) (string, string) {
		parts := strings.SplitN(s, sep, 2)
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
		return parts[0], ""
	}
	parsePrefix := func(l string) (prefix *Prefix, tail string) {
		prefix = new(Prefix)
		if l == "" || l[0] != ':' {
			return prefix, l
		}

		head, tail := split2(l[1:], " ")
		user := strings.Index(head, "!")
		host := strings.Index(head, "@")
		if host > user {
			prefix.Host = head[host+1:]
			head = head[:host]
		}
		if user > 0 {
			prefix.User = head[user+1:]
			head = head[:user]
		}
		prefix.Name = head
		return prefix, tail
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
type Conn struct {
	rwc    io.ReadWriteCloser
	reader *bufio.Reader

	rl sync.Mutex
	wl sync.Mutex
}

// Create a new IRC connection
func NewConn(rwc io.ReadWriteCloser) *Conn {
	return &Conn{
		reader: bufio.NewReader(rwc),
		rwc:    rwc,
	}
}

// Establish a connection
func Dial(addr string) (*Conn, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return NewConn(conn), nil
}

//  Establish a secure connection
func DialTLS(addr string, config *tls.Config) (*Conn, error) {
	conn, err := tls.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}
	return NewConn(conn), nil
}

// Reads the next message
func (c *Conn) ReadMsg() (*Message, error) {
	c.rl.Lock()
	defer c.rl.Unlock()
	l, err := c.reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	return ParseMessage(l), nil
}

// Writes Raw bytes
func (c *Conn) Write(msg []byte) (int, error) {
	c.wl.Lock()
	defer c.wl.Unlock()

	return c.rwc.Write(msg)
}

// Writes string. "\r\n" will be appended
func (c *Conn) Send(l string) error {
	_, err := c.Write([]byte(l + "\r\n"))
	return err
}

// Closes connection
func (c *Conn) Close() error {
	return c.rwc.Close()
}
