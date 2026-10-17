package sshremote

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

// ForwardDirection selects a local or reverse SSH tunnel.
type ForwardDirection string

const (
	ForwardLocal   ForwardDirection = "local"
	ForwardReverse ForwardDirection = "reverse"
)

// ForwardSpec is a validated SSH port-forward definition. Local forwards
// listen on the desktop and dial targetHost:targetPort through SSH. Reverse
// forwards ask the remote sshd to listen and dial targetHost:targetPort on the
// desktop when a connection arrives.
type ForwardSpec struct {
	Direction  ForwardDirection `json:"direction"`
	ListenHost string           `json:"listenHost,omitempty"`
	ListenPort int              `json:"listenPort"`
	TargetHost string           `json:"targetHost"`
	TargetPort int              `json:"targetPort"`
}

// ParseForwardSpec accepts the legacy local form
// localPort:remoteHost:remotePort and an explicit local:/reverse: prefix.
// Hosts may be bracketed IPv6 literals when they contain colons.
func ParseForwardSpec(value string) (ForwardSpec, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return ForwardSpec{}, errors.New("forward spec required")
	}
	direction := ForwardLocal
	for _, prefix := range []struct {
		name string
		kind ForwardDirection
	}{
		{"local:", ForwardLocal}, {"reverse:", ForwardReverse},
	} {
		if strings.HasPrefix(strings.ToLower(raw), prefix.name) {
			direction = prefix.kind
			raw = strings.TrimSpace(raw[len(prefix.name):])
			break
		}
	}
	first := strings.IndexByte(raw, ':')
	last := strings.LastIndexByte(raw, ':')
	if first <= 0 || last <= first || last == len(raw)-1 {
		return ForwardSpec{}, errors.New("forward spec must be localPort:remoteHost:remotePort")
	}
	listenPort, err := parseForwardPort(raw[:first])
	if err != nil {
		return ForwardSpec{}, fmt.Errorf("forward listen port: %w", err)
	}
	targetPort, err := parseForwardPort(raw[last+1:])
	if err != nil {
		return ForwardSpec{}, fmt.Errorf("forward target port: %w", err)
	}
	targetHost := strings.TrimSpace(raw[first+1 : last])
	if strings.HasPrefix(targetHost, "[") && strings.HasSuffix(targetHost, "]") {
		targetHost = strings.TrimSuffix(strings.TrimPrefix(targetHost, "["), "]")
	}
	return validateForward(ForwardSpec{Direction: direction, ListenHost: "127.0.0.1", ListenPort: listenPort, TargetHost: targetHost, TargetPort: targetPort})
}

func parseForwardPort(value string) (int, error) {
	port, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || port < 1 || port > 65535 {
		return 0, errors.New("port out of range")
	}
	return port, nil
}

func validateForward(spec ForwardSpec) (ForwardSpec, error) {
	if spec.Direction != ForwardLocal && spec.Direction != ForwardReverse {
		return ForwardSpec{}, errors.New("unknown forward direction")
	}
	if spec.ListenHost == "" {
		spec.ListenHost = "127.0.0.1"
	}
	if strings.TrimSpace(spec.TargetHost) == "" {
		return ForwardSpec{}, errors.New("forward target host required")
	}
	if spec.ListenPort < 1 || spec.ListenPort > 65535 || spec.TargetPort < 1 || spec.TargetPort > 65535 {
		return ForwardSpec{}, errors.New("forward port out of range")
	}
	return spec, nil
}

// String emits the same compact form accepted by ParseForwardSpec.
func (f ForwardSpec) String() string {
	if f.Direction == ForwardReverse {
		return "reverse:" + strconv.Itoa(f.ListenPort) + ":" + net.JoinHostPort(f.TargetHost, strconv.Itoa(f.TargetPort))
	}
	return strconv.Itoa(f.ListenPort) + ":" + net.JoinHostPort(f.TargetHost, strconv.Itoa(f.TargetPort))
}

// StartForward starts one validated forward on an existing SSH client. The
// returned stop function is idempotent and closes listeners and active
// channels when called.
func StartForward(ctx context.Context, client *ssh.Client, spec ForwardSpec) (func(), error) {
	if client == nil {
		return nil, errors.New("ssh client required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	validated, err := validateForward(spec)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	var once sync.Once
	stop := func() { once.Do(cancel) }
	if validated.Direction == ForwardReverse {
		// x/crypto/ssh exposes remote TCP forwarding through ListenTCP. The
		// listener is consumed by the copy loop below and closes on stop.
		remoteListener, err := client.ListenTCP(&net.TCPAddr{IP: net.ParseIP(validated.ListenHost), Port: validated.ListenPort})
		if err != nil {
			cancel()
			return nil, err
		}
		go func() {
			<-ctx.Done()
			_ = remoteListener.Close()
		}()
		go bridgeForwardConnections(ctx, remoteListener, validated.TargetHost, validated.TargetPort)
		return func() {
			stop()
			_ = remoteListener.Close()
		}, nil
	}
	localListener, err := net.Listen("tcp", net.JoinHostPort(validated.ListenHost, strconv.Itoa(validated.ListenPort)))
	if err != nil {
		cancel()
		return nil, err
	}
	go func() {
		<-ctx.Done()
		_ = localListener.Close()
	}()
	go bridgeForwardConnections(ctx, localListener, validated.TargetHost, validated.TargetPort, client)
	return func() {
		stop()
		_ = localListener.Close()
	}, nil
}

func bridgeForwardConnections(ctx context.Context, listener net.Listener, targetHost string, targetPort int, clients ...*ssh.Client) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		go func(local net.Conn) {
			defer local.Close()
			var remote net.Conn
			var dialErr error
			if len(clients) > 0 {
				remote, dialErr = clients[0].Dial("tcp", net.JoinHostPort(targetHost, strconv.Itoa(targetPort)))
			} else {
				remote, dialErr = net.Dial("tcp", net.JoinHostPort(targetHost, strconv.Itoa(targetPort)))
			}
			if dialErr != nil {
				return
			}
			defer remote.Close()
			done := make(chan struct{}, 2)
			go func() { _, _ = io.Copy(remote, local); done <- struct{}{} }()
			go func() { _, _ = io.Copy(local, remote); done <- struct{}{} }()
			select {
			case <-done:
			case <-ctx.Done():
			}
		}(conn)
	}
}
