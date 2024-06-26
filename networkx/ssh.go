package networkx

import (
	"golang.org/x/crypto/ssh"
	"net"
	"time"
)

func NewJumpProxy(jumpAddr, user, password, target string) (conn net.Conn, err error) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	client, err := ssh.Dial("tcp", jumpAddr, config)
	if err != nil {
		return
	}
	conn, err = client.Dial("tcp", target)
	return
}
