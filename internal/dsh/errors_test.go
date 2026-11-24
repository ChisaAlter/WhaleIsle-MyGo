package dsh

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"
)

func TestTransportErrorClassification(t *testing.T) {
	reset := &TransportError{Op: "session/list", Err: &net.OpError{Op: "read", Err: syscall.ECONNRESET}}
	if !reset.Temporary() {
		t.Fatal("connection reset must be temporary")
	}
	if got := reset.Error(); !strings.Contains(got, "dsh session/list:") {
		t.Fatalf("Error() = %q", got)
	}
	if !errors.Is(reset, syscall.ECONNRESET) {
		t.Fatal("Unwrap must expose the cause")
	}
	eof := &TransportError{Op: "x", Err: io.EOF}
	if !eof.Temporary() {
		t.Fatal("EOF must be temporary")
	}
	gone := &TransportError{Op: "x", Err: errors.New("malformed frame")}
	if gone.Temporary() {
		t.Fatal("a protocol failure is not transient")
	}
}

func TestHTTPErrorClassification(t *testing.T) {
	h := &HTTPError{Op: "chat/send", Status: 429, Body: "rate limited"}
	if !h.Temporary() {
		t.Fatal("429 must be transient")
	}
	if got := h.Error(); got != "dsh chat/send: HTTP 429: rate limited" {
		t.Fatalf("Error() = %q", got)
	}
	for _, status := range []int{400, 401, 403, 404} {
		if (&HTTPError{Status: status}).Temporary() {
			t.Fatalf("%d must not be transient", status)
		}
	}
	if !(&HTTPError{Status: 503}).Temporary() {
		t.Fatal("503 must be transient")
	}
}

func TestErrEndpointNotReadyCodes(t *testing.T) {
	for _, code := range []string{"gateway/invocation-unavailable", "gateway/service-unavailable"} {
		err := &Error{Code: code, Message: "unavailable"}
		if !errors.Is(err, ErrEndpointNotReady) {
			t.Fatalf("%s must classify as ErrEndpointNotReady", code)
		}
	}
	if errors.Is(&Error{Code: "NOT_FOUND", Message: "x"}, ErrEndpointNotReady) {
		t.Fatal("other codes must not match ErrEndpointNotReady")
	}
	if errors.Is(errors.New("x"), ErrEndpointNotReady) {
		t.Fatal("unrelated errors must not match")
	}
}

func TestCallReturnsTypedHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	}))
	defer srv.Close()
	c := &Client{Endpoint: Endpoint{Origin: srv.URL}}
	err := c.Call(context.Background(), "session/get", map[string]any{}, nil)
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) || httpErr.Status != 404 || httpErr.Op != "session/get" {
		t.Fatalf("got %v (%T)", err, err)
	}
}

func TestCallReturnsTypedTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // closed before the call: the dial is refused
	c := &Client{Endpoint: Endpoint{Origin: srv.URL}}
	err := c.Call(context.Background(), "session/get", map[string]any{}, nil)
	var transport *TransportError
	if !errors.As(err, &transport) || transport.Op != "session/get" {
		t.Fatalf("got %v (%T)", err, err)
	}
	if !transport.Temporary() {
		t.Fatal("connection refused must be temporary")
	}
}

func TestCallRPCErrEndpointNotReadyThroughWire(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":  "server-response",
			"rpcId": body["rpcId"],
			"result": map[string]any{
				"ok":    false,
				"error": map[string]any{"code": "gateway/service-unavailable", "message": "unavailable"},
			},
		})
	}))
	defer srv.Close()
	c := &Client{Endpoint: Endpoint{Origin: srv.URL}}
	err := c.Call(context.Background(), "im/status", map[string]any{}, nil)
	var rpcErr *Error
	if !errors.As(err, &rpcErr) || !errors.Is(err, ErrEndpointNotReady) {
		t.Fatalf("got %v", err)
	}
}
