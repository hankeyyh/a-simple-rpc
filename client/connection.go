package client

import (
	"crypto/tls"
	"errors"
	"net"
)

func newDirectConn(c *Client, network, address string) (net.Conn, error) {
	if c == nil {
		return nil, errors.New("client is nil")
	}

	var tlsConn *tls.Conn
	var conn net.Conn
	var err error

	if c.option.TlsConfig != nil {
		dialer := &net.Dialer{
			Timeout: c.option.ConnectTimeout,
		}
		tlsConn, err = tls.DialWithDialer(dialer, network, address, c.option.TlsConfig)
		conn = net.Conn(tlsConn)
	} else {
		conn, err = net.DialTimeout(network, address, c.option.ConnectTimeout)
	}

	return conn, err
}
