package im

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// RefreshCordisInventory calls dynamicCordisRunner/inventory and stores a short note.
func (s *Store) RefreshCordisInventory(ctx context.Context) {
	s.mu.Lock()
	client, epoch, generation := s.client, s.connectionEpoch, s.refreshGeneration
	s.mu.Unlock()
	s.refreshCordisInventory(ctx, client, epoch, generation)
}

func (s *Store) refreshCordisInventory(ctx context.Context, client *dsh.Client, epoch, generation uint64) {
	if client == nil {
		return
	}
	go func() {
		var out api.InventoryResult
		err := client.Call(ctx, "dynamicCordisRunner/inventory", map[string]any{}, &out)
		note := ""
		if err != nil {
			note = "dynamicCordisRunner/inventory: " + err.Error()
		} else {
			note = fmt.Sprintf("dynamicCordisRunner/inventory: %d plugins", len(out))
			if len(out) > 0 {
				item := out[0]
				pid := item.PluginId
				runID := ""
				if item.ActiveRun != nil {
					runID = item.ActiveRun.PluginRunId
				}
				args := map[string]any{"pluginId": pid, "pluginRunId": runID}
				var code api.DynamicCordisClientSource
				if cerr := client.Call(ctx, "dynamicCordisRunner/getClientCode", args, &code); cerr == nil {
					note += " · clientCode ok"
				}
			}
		}
		s.mu.Lock()
		if s.client != client || s.connectionEpoch != epoch || s.refreshGeneration != generation {
			s.mu.Unlock()
			return
		}
		if s.pluginDiag != "" {
			s.pluginDiag = s.pluginDiag + " | " + note
		} else {
			s.pluginDiag = note
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// StopCordisPlugin calls dynamicCordisRunner/stopFromPanel for a plugin id.
func (s *Store) StopCordisPlugin(ctx context.Context, pluginID string) {
	pluginID = strings.TrimSpace(pluginID)
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil || pluginID == "" {
		return
	}
	go func() {
		var out api.DynamicCordisStopResponse
		err := client.Call(ctx, "dynamicCordisRunner/stopFromPanel", map[string]any{
			"pluginId": pluginID,
		}, &out)
		s.mu.Lock()
		if err != nil {
			s.pluginDiag = "cordis stop: " + err.Error()
		} else {
			s.pluginDiag = "cordis stopped: " + pluginID
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// InvokeCordisMethod calls dynamicCordisRunner/invoke for a plugin run.
func (s *Store) InvokeCordisMethod(ctx context.Context, pluginID, pluginRunID, method string, args json.RawMessage) {
	pluginID = strings.TrimSpace(pluginID)
	method = strings.TrimSpace(method)
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil || pluginID == "" || method == "" {
		return
	}
	go func() {
		req := map[string]any{
			"pluginId":    pluginID,
			"pluginRunId": pluginRunID,
			"method":      method,
		}
		if len(args) > 0 {
			req["args"] = args
		}
		var out api.DynamicCordisInvokeResult
		err := client.Call(ctx, "dynamicCordisRunner/invoke", req, &out)
		s.mu.Lock()
		if err != nil {
			s.pluginDiag = "cordis invoke: " + err.Error()
		} else {
			s.pluginDiag = "cordis invoke ok: " + method
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// UndefineCordisPlugin calls dynamicCordisRunner/undefineFromPanel.
func (s *Store) UndefineCordisPlugin(ctx context.Context, pluginID string) {
	pluginID = strings.TrimSpace(pluginID)
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil || pluginID == "" {
		return
	}
	go func() {
		var out api.DynamicCordisUndefineReceipt
		err := client.Call(ctx, "dynamicCordisRunner/undefineFromPanel", map[string]any{
			"pluginId": pluginID,
		}, &out)
		s.mu.Lock()
		if err != nil {
			s.pluginDiag = "cordis undefine: " + err.Error()
		} else {
			s.pluginDiag = "cordis undefined: " + pluginID
		}
		s.mu.Unlock()
		s.fire()
	}()
}
