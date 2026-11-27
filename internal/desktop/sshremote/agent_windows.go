//go:build windows

package sshremote

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const defaultWindowsSSHAgentPipe = `\\.\pipe\openssh-ssh-agent`

var (
	kernel32DLL       = windows.NewLazySystemDLL("kernel32.dll")
	procWaitNamedPipe = kernel32DLL.NewProc("WaitNamedPipeW")
)

func waitNamedPipe(name *uint16, timeoutMS uint32) error {
	r, _, err := procWaitNamedPipe.Call(uintptr(unsafe.Pointer(name)), uintptr(timeoutMS))
	if r == 0 {
		return err
	}
	return nil
}

func dialSSHAgent(timeout time.Duration) (net.Conn, error) {
	sock := strings.TrimSpace(os.Getenv("SSH_AUTH_SOCK"))
	if sock != "" {
		if strings.HasPrefix(sock, `\\.\pipe\`) || strings.HasPrefix(sock, `\\?\pipe\`) {
			return dialWindowsNamedPipe(sock, timeout)
		}
		return net.DialTimeout("unix", sock, timeout)
	}
	return dialWindowsNamedPipe(defaultWindowsSSHAgentPipe, timeout)
}

func dialWindowsNamedPipe(pipePath string, timeout time.Duration) (net.Conn, error) {
	pathPtr, err := windows.UTF16PtrFromString(pipePath)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(timeout)
	for {
		h, err := windows.CreateFile(
			pathPtr,
			windows.GENERIC_READ|windows.GENERIC_WRITE,
			0,
			nil,
			windows.OPEN_EXISTING,
			0,
			0,
		)
		if err == nil {
			f := os.NewFile(uintptr(h), pipePath)
			return &windowsPipeConn{file: f, path: pipePath}, nil
		}
		if err != windows.ERROR_PIPE_BUSY {
			return nil, fmt.Errorf("windows openssh agent (%s): %w", pipePath, err)
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, fmt.Errorf("windows openssh agent (%s): busy timeout", pipePath)
		}
		waitMS := uint32(remaining.Milliseconds())
		if waitMS == 0 {
			waitMS = 1
		}
		if werr := waitNamedPipe(pathPtr, waitMS); werr != nil {
			return nil, fmt.Errorf("windows openssh agent (%s): %w", pipePath, werr)
		}
	}
}

type windowsPipeAddr string

func (a windowsPipeAddr) Network() string { return "pipe" }
func (a windowsPipeAddr) String() string  { return string(a) }

type windowsPipeConn struct {
	file *os.File
	path string
}

func (c *windowsPipeConn) Read(p []byte) (int, error)         { return c.file.Read(p) }
func (c *windowsPipeConn) Write(p []byte) (int, error)        { return c.file.Write(p) }
func (c *windowsPipeConn) Close() error                       { return c.file.Close() }
func (c *windowsPipeConn) LocalAddr() net.Addr                { return windowsPipeAddr(c.path) }
func (c *windowsPipeConn) RemoteAddr() net.Addr               { return windowsPipeAddr(c.path) }
func (c *windowsPipeConn) SetDeadline(t time.Time) error      { return c.file.SetDeadline(t) }
func (c *windowsPipeConn) SetReadDeadline(t time.Time) error  { return c.file.SetReadDeadline(t) }
func (c *windowsPipeConn) SetWriteDeadline(t time.Time) error { return c.file.SetWriteDeadline(t) }
