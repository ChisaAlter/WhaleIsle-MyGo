package im

import (
	"context"
	"encoding/json"
	"strings"

	"whaleisle/internal/dsh/api"
)

// ReportCordisClientGuardFailure calls dynamicCordisRunner/reportClientGuardFailure.
func (s *Store) ReportCordisClientGuardFailure(ctx context.Context, pluginID, pluginRunID string, failure api.CordisErrorDetails) {
	s.reportCordis(ctx, "dynamicCordisRunner/reportClientGuardFailure", pluginID, pluginRunID, failure)
}

// ReportCordisRenderFailure calls dynamicCordisRunner/reportRenderFailure.
func (s *Store) ReportCordisRenderFailure(ctx context.Context, pluginID, pluginRunID string, failure api.DynamicCordisRenderFailure) {
	s.reportCordis(ctx, "dynamicCordisRunner/reportRenderFailure", pluginID, pluginRunID, failure)
}

func (s *Store) reportCordis(ctx context.Context, method, pluginID, pluginRunID string, failure any) {
	pluginID = strings.TrimSpace(pluginID)
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil || pluginID == "" {
		return
	}
	go func() {
		err := client.Call(ctx, method, map[string]any{
			"pluginId":    pluginID,
			"pluginRunId": pluginRunID,
			"failure":     failure,
		}, &json.RawMessage{})
		s.mu.Lock()
		if err != nil {
			s.pluginDiag = method + ": " + err.Error()
		} else {
			s.pluginDiag = method + ": ok"
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// ResolveCordisInspectQuery calls dynamicCordisRunner/resolveInspectQuery.
func (s *Store) ResolveCordisInspectQuery(ctx context.Context, requestID string, resolution api.CordisInspectQueryResolution) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	requestID = strings.TrimSpace(requestID)
	if client == nil || requestID == "" {
		return
	}
	go func() {
		var out api.CordisInspectResolveAck
		err := client.Call(ctx, "dynamicCordisRunner/resolveInspectQuery", map[string]any{
			"requestId":  requestID,
			"resolution": resolution,
		}, &out)
		s.mu.Lock()
		if err != nil {
			s.pluginDiag = "cordis resolveInspect: " + err.Error()
		} else {
			s.pluginDiag = "cordis resolveInspect ok"
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// ResolveCordisRequestRun calls dynamicCordisRunner/resolveRequestRun.
func (s *Store) ResolveCordisRequestRun(ctx context.Context, requestID string, resolution api.DynamicCordisRunResolution) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	requestID = strings.TrimSpace(requestID)
	if client == nil || requestID == "" {
		return
	}
	go func() {
		err := client.Call(ctx, "dynamicCordisRunner/resolveRequestRun", map[string]any{
			"requestId":  requestID,
			"resolution": resolution,
		}, &json.RawMessage{})
		s.mu.Lock()
		if err != nil {
			s.pluginDiag = "cordis resolveRun: " + err.Error()
		} else {
			s.pluginDiag = "cordis resolveRun ok"
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// RunCordisHostHalf calls dynamicCordisRunner/runHostHalf.
func (s *Store) RunCordisHostHalf(ctx context.Context, packageID, mode, requestID string, approveFuture bool) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	packageID = strings.TrimSpace(packageID)
	if client == nil || packageID == "" {
		return
	}
	go func() {
		var out api.DynamicCordisHostHalfResult
		err := client.Call(ctx, "dynamicCordisRunner/runHostHalf", map[string]any{
			"packageId":             packageID,
			"mode":                  mode,
			"requestId":             requestID,
			"approveFutureVersions": approveFuture,
		}, &out)
		s.mu.Lock()
		if err != nil {
			s.pluginDiag = "cordis runHostHalf: " + err.Error()
		} else {
			s.pluginDiag = "cordis runHostHalf ok"
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// SettleCordisUserRun calls dynamicCordisRunner/settleUserRun.
func (s *Store) SettleCordisUserRun(ctx context.Context, pluginID string, resolution api.DynamicCordisRunResolution) {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	pluginID = strings.TrimSpace(pluginID)
	if client == nil || pluginID == "" {
		return
	}
	go func() {
		var out api.DynamicCordisRunResponse
		err := client.Call(ctx, "dynamicCordisRunner/settleUserRun", map[string]any{
			"pluginId":   pluginID,
			"resolution": resolution,
		}, &out)
		s.mu.Lock()
		if err != nil {
			s.pluginDiag = "cordis settle: " + err.Error()
		} else {
			s.pluginDiag = "cordis settle ok"
		}
		s.mu.Unlock()
		s.fire()
	}()
}
