package settings

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// ProductAnalyticsEnabled calls productAnalytics/enabled.
func ProductAnalyticsEnabled(ctx context.Context, client *dsh.Client) (bool, error) {
	if client == nil {
		return false, nil
	}
	var out api.EnabledResult
	err := client.Call(ctx, "productAnalytics/enabled", map[string]any{}, &out)
	return bool(out), err
}

// ReportProductEvent calls productAnalytics/report with a raw event payload.
func ReportProductEvent(ctx context.Context, client *dsh.Client, event api.ProductEvent) error {
	if client == nil || len(event) == 0 {
		return nil
	}
	var out api.ReportResult
	return client.Call(ctx, "productAnalytics/report", map[string]any{"event": event}, &out)
}

// WatchAnalyticsPolicy opens productAnalytics/watchPolicy and invokes onPolicy for each frame.
func WatchAnalyticsPolicy(ctx context.Context, mux *dsh.Mux, onPolicy func(enabled bool)) {
	if mux == nil || onPolicy == nil {
		return
	}
	if err := mux.Dial(ctx); err != nil {
		return
	}
	stream, err := mux.Open(ctx, "productAnalytics/watchPolicy", map[string]any{"args": map[string]any{}})
	if err != nil {
		return
	}
	defer stream.Cancel()
	for {
		raw, err := stream.Recv(ctx)
		if err != nil {
			if errors.Is(err, dsh.ErrStreamEnd) || errors.Is(err, context.Canceled) ||
				strings.Contains(err.Error(), "context canceled") {
				return
			}
			return
		}
		var on api.WatchPolicyResult
		if json.Unmarshal(raw, &on) == nil {
			onPolicy(bool(on))
			continue
		}
		var b bool
		if json.Unmarshal(raw, &b) == nil {
			onPolicy(b)
		}
	}
}
