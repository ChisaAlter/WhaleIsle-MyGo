package main

import (
	"context"
	"errors"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
)

// This opt-in mode starts only the existing owned blank Host and reads its
// authenticated declaration stream. It creates no sessions and never asks for
// model catalogs, providers, account state or command execution.
func checkNativeContributions(endpoint harness.Endpoint, r *report) error {
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: endpoint.Origin, Cookie: endpoint.Cookie}}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	values := make(chan dsh.NativeContributions, 1)
	done := make(chan error, 1)
	go func() {
		done <- client.WatchNativeContributions(ctx, func(value dsh.NativeContributions) {
			select {
			case values <- value:
			case <-ctx.Done():
			}
		})
	}()
	var snapshot dsh.NativeContributions
	select {
	case snapshot = <-values:
	case err := <-done:
		if err == nil {
			return errors.New("native contribution stream ended before a declaration")
		}
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
	checks := []struct {
		name string
		pass bool
	}{
		{"native-graph-known", snapshot.Known && snapshot.Error == "" && snapshot.Revision != ""},
		{"native-file-registration", snapshot.FileCommand && snapshot.CommandRevision("file") != ""},
		{"native-model-registration", snapshot.ModelCommand && snapshot.CommandRevision("model") != ""},
		{"native-permission-decoration", snapshot.PermissionCommand && snapshot.CommandRevision("permission") != ""},
		{"native-feedback-decoration", snapshot.FeedbackCommand && snapshot.CommandRevision("feedback") != ""},
		{"native-1017-label-registration", snapshot.ModelLabel1017},
	}
	passed := true
	for _, item := range checks {
		r.Checks = append(r.Checks, check{Name: item.name, Pass: item.pass})
		passed = passed && item.pass
	}
	// Keep only the content identity. Never persist the Host's cookie, raw boot
	// document, graph resource URLs or third-party source text.
	r.Checks = append(r.Checks, check{Name: "native-graph-identity", Pass: snapshot.Known, SHA256: snapshot.Revision})
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			return errors.New("native contribution stream did not retire on cancellation")
		}
		r.Checks = append(r.Checks, check{Name: "native-stream-cancel", Pass: true})
	case <-time.After(3 * time.Second):
		return errors.New("native contribution stream remained active after cancellation")
	}
	if !passed {
		return errors.New("the owned default Host did not admit every fixed native contribution")
	}
	return nil
}
