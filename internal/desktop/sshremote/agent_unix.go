//go:build !windows

package sshremote

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func dialSSHAgent(timeout time.Duration) (net.Conn, error) {
	sock := strings.TrimSpace(os.Getenv("SSH_AUTH_SOCK"))
	if sock == "" {
		return nil, fmt.Errorf("SSH_AUTH_SOCK is not set")
	}
	return net.DialTimeout("unix", sock, timeout)
}
