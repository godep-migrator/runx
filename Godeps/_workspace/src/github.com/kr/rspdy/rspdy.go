/*

Package rspdy provides support for the RSPDY protocol.

RSPDY is exactly like SPDY, except the roles of client and server
are reversed. The initiating endpoint serves incoming requests
on the connection it dials; the listening endpoint sends outgoing
requests over accepted connections.

*/
package rspdy

import (
	"crypto/tls"
	"fmt"
	"github.com/kr/spdy"
	"net"
	"net/http"
)

// DialAndServeTLS dials a TLS connection, then serves incoming HTTP
// requests. If h is nil, it uses http.DefaultHandler.
// If config.NextProtos is nil, it uses "rspdy/3".
func DialAndServeTLS(network, addr string, config *tls.Config, h http.Handler) error {
	var srv spdy.Server
	srv.Handler = h
	config1 := new(tls.Config)
	if config != nil {
		*config1 = *config
	}
	if config1.NextProtos == nil {
		config1.NextProtos = []string{"rspdy/3"}
	}
	// TODO(kr): cert stuff
	conn, err := tls.Dial(network, addr, config1)
	if err != nil {
		return err
	}
	return srv.ServeConn(conn)
}

type Listener struct {
	l net.Listener
}

func NewListener(l net.Listener) *Listener {
	return &Listener{l}
}

// Listen listens for incoming TLS connections and
// returns a Listener that accepts SPDY sessions.
func ListenTLS(addr, certFile, keyFile string) (*Listener, error) {
	var config tls.Config
	config.NextProtos = []string{"rspdy/3"}
	var err error
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	l, err := tls.Listen("tcp", addr, &config)
	if err != nil {
		return nil, err
	}
	return &Listener{l}, nil
}

func (l *Listener) accept() (net.Conn, error) {
	c, err := l.l.Accept()
	if err != nil {
		return nil, err
	}
	if tlsConn, ok := c.(*tls.Conn); ok {
		if err := tlsConn.Handshake(); err != nil {
			return nil, err
		}
		tlsState := tlsConn.ConnectionState()
		if s := tlsState.NegotiatedProtocol; s != "rspdy/3" {
			c.Close()
			return nil, fmt.Errorf("unknown protocol: %s", s)
		}
	}
	return c, nil
}

func (l *Listener) AcceptSPDY() (*spdy.Conn, error) {
	c, err := l.accept()
	if err != nil {
		return nil, err
	}
	return &spdy.Conn{Conn: c}, nil
}
