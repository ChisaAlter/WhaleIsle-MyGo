package dsh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"syscall"
)

// TransportError is a failure below the HTTP envelope: the request never
// reached a well-formed Host answer (dial, reset, EOF, a read that stopped
// mid-body). Callers tell it apart from a rejection the Host itself wrote.
type TransportError struct {
	// Op is the RPC method, e.g. "session/list".
	Op  string
	Err error
}

func (e *TransportError) Error() string {
	return fmt.Sprintf("dsh %s: %v", e.Op, e.Err)
}

func (e *TransportError) Unwrap() error { return e.Err }

// Temporary reports whether the failure is transient by nature — a
// connection reset, refusal, EOF, timeout or unreachable host — so a
// caller may retry without treating the endpoint as broken.
func (e *TransportError) Temporary() bool {
	return temporaryTransport(e.Err)
}

func temporaryTransport(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var ne net.Error
	if errors.As(err, &ne) && (ne.Timeout() || ne.Temporary()) {
		return true
	}
	for _, errno := range []syscall.Errno{
		syscall.ECONNRESET,
		syscall.ECONNREFUSED,
		syscall.ECONNABORTED,
		syscall.ETIMEDOUT,
		syscall.EPIPE,
		syscall.EHOSTUNREACH,
		syscall.ENETUNREACH,
	} {
		if errors.Is(err, errno) {
			return true
		}
	}
	// Windows surfaces refusal and reset as WSA-codes Errno values whose text
	// is the platform's own wording, so the same vocabulary the message shows
	// is the last resort when no structured claim matched.
	return temporaryText.MatchString(err.Error())
}

// temporaryText is the transports' own vocabulary for a transient failure:
// winsock words on Windows ("actively refused", "forcibly closed"), the BSD
// spellings on unix. It never decides the cause — only that a retry may work.
var temporaryText = regexp.MustCompile(`(?i)connection (?:refused|reset|aborted)|actively refused|forcibly closed|no such host|host is unreachable|network is unreachable|broken pipe|timed out|timeout|i/o timeout`)

// HTTPError is a completed HTTP exchange whose status the Host rejected.
// The status is the fact a caller needs; Body keeps a clipped excerpt for
// the log, not for the screen.
type HTTPError struct {
	Op     string
	Status int
	Body   string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("dsh %s: HTTP %d: %s", e.Op, e.Status, e.Body)
}

// Temporary reports whether the status describes a transient rejection:
// rate limiting or a server-side failure, never a malformed request.
func (e *HTTPError) Temporary() bool {
	return e.Status == 429 || e.Status >= 500
}

// ErrEndpointNotReady marks a Host answer that a lazily activated service
// has not started yet. It is transient by design: the caller should retry
// after the service warms up rather than report a broken endpoint. errors.Is
// reaches it through Error.Code on the wire codes below.
var ErrEndpointNotReady = errors.New("dsh endpoint not ready")

// Is lets errors.Is(err, ErrEndpointNotReady) reach through a *Error when
// the Host's code is one of the lazy-activation answers.
func (e *Error) Is(target error) bool {
	if target != ErrEndpointNotReady {
		return false
	}
	switch e.Code {
	case "gateway/invocation-unavailable", "gateway/service-unavailable":
		return true
	}
	return false
}
