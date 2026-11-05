package dsh

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type deliveryRoundTrip func(*http.Request) (*http.Response, error)

func (f deliveryRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestEventDeliveryBoundAndNeverSerialized(t *testing.T) {
	var calls atomic.Int32
	c := &Client{Endpoint: Endpoint{Origin: "http://fixture.invalid"}, HTTP: &http.Client{Transport: deliveryRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		_, rpcID := decodeEventResultRPC(t, r)
		body, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": rpcID, "result": map[string]any{"ok": true}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body))), Header: make(http.Header)}, nil
	})}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	valid := true
	delivery := &EventDelivery{Client: c, EventID: "e", Context: ctx, Valid: func() bool { return valid }}
	result := EventResult{ClientID: "listener", EventID: "e", Outcome: EventResultOutcome{Kind: "result", Value: json.RawMessage(`"allowed-once"`), Delivery: delivery}}
	wire, err := json.Marshal(result)
	if err != nil || strings.Contains(string(wire), "Delivery") || strings.Contains(string(wire), "Context") {
		t.Fatalf("authority leaked %s %v", wire, err)
	}
	if err = c.PostEventResult(context.Background(), result); err != nil {
		t.Fatal(err)
	}
	valid = false
	if err = c.PostEventResult(context.Background(), result); err == nil {
		t.Fatal("stale scope sent")
	}
	valid = true
	delivery.EventID = "foreign"
	if err = c.PostEventResult(context.Background(), result); err == nil {
		t.Fatal("foreign event sent")
	}
	delivery.EventID = "e"
	delivery.Client = &Client{}
	if err = c.PostEventResult(context.Background(), result); err == nil {
		t.Fatal("foreign Host sent")
	}
	delivery.Client = c
	cancel()
	if err = c.PostEventResult(context.Background(), result); err == nil {
		t.Fatal("revoked scope sent")
	}
	if calls.Load() != 1 {
		t.Fatal("invalid result reached transport")
	}
}
func TestEventDeliveryRevocationCancelsHTTP(t *testing.T) {
	entered := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &Client{Endpoint: Endpoint{Origin: "http://fixture.invalid"}, HTTP: &http.Client{Transport: deliveryRoundTrip(func(r *http.Request) (*http.Response, error) {
		close(entered)
		<-r.Context().Done()
		return nil, r.Context().Err()
	})}}
	result := EventResult{ClientID: "listener", EventID: "e", Outcome: EventResultOutcome{Kind: "result", Delivery: &EventDelivery{Client: c, EventID: "e", Context: ctx, Valid: func() bool { return true }}}}
	done := make(chan error, 1)
	go func() { done <- c.PostEventResult(context.Background(), result) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("request never entered")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("revoked request succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("revoked HTTP not cancelled")
	}
}
