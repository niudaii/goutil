package networkx

import (
	"crypto/tls"
	"errors"
	"golang.org/x/net/proxy"
	"net"
	"syscall"
	"time"
)

func NewConn(addr, proxyAddr string, timeout time.Duration) (conn net.Conn, err error) {
	if proxyAddr == "" {
		conn, err = net.DialTimeout("tcp", addr, timeout)
	} else {
		var proxyDialer proxy.Dialer
		proxyDialer, err = proxyDialerFromURL(proxyAddr, timeout)
		if err != nil {
			return
		}
		conn, err = proxyDialer.Dial("tcp", addr)
	}
	return
}

func NewTLSConn(conn net.Conn) (tlsConn *tls.Conn, err error) {
	config := &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS10,
		MaxVersion:         tls.VersionTLS13,
	}
	tlsConn = tls.Client(conn, config)
	err = tlsConn.Handshake()
	return
}

func Send(conn net.Conn, data []byte, timeout time.Duration) (err error) {
	err = conn.SetWriteDeadline(time.Now().Add(timeout))
	if err != nil {
		return
	}
	_, err = conn.Write(data)
	return
}

func Recv(conn net.Conn, timeout time.Duration) ([]byte, error) {
	response := make([]byte, 4096)
	err := conn.SetReadDeadline(time.Now().Add(timeout))
	if err != nil {
		return []byte{}, err
	}
	length, err := conn.Read(response)
	if err != nil {
		var netErr net.Error
		if (errors.As(err, &netErr) && netErr.Timeout()) ||
			errors.Is(err, syscall.ECONNREFUSED) { // timeout error or connection refused
			return []byte{}, nil
		}
		return response[:length], err
	}
	return response[:length], nil
}
