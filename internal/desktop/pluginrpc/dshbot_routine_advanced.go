package pluginrpc

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode/utf16"
)

type DshbotRoutineWatch struct {
	Kind      string `json:"kind"`
	Value     string `json:"value"`
	TimeoutMS int    `json:"timeoutMs"`
}

// Presence is intentional: untouched advanced fields must remain omitted.
// WatchSet with Watch=nil is the original explicit watch:null removal.
type DshbotRoutineAdvancedPatch struct {
	ContextSet, SilentSet, WatchSet bool
	ContextFrom                     []string
	SilentAllowed                   bool
	Watch                           *DshbotRoutineWatch
}

func CloneDshbotRoutineAdvanced(p DshbotRoutineAdvancedPatch) DshbotRoutineAdvancedPatch {
	p.ContextFrom = append([]string(nil), p.ContextFrom...)
	if p.Watch != nil {
		w := *p.Watch
		p.Watch = &w
	}
	return p
}
func ValidateDshbotRoutineAdvanced(p DshbotRoutineAdvancedPatch) error {
	if p.ContextSet {
		if len(p.ContextFrom) > 3 {
			return fmt.Errorf("contextFrom permits at most 3 selected sources")
		}
		seen := map[string]bool{}
		for _, id := range p.ContextFrom {
			if strings.TrimSpace(id) == "" || strings.TrimSpace(id) != id || seen[id] {
				return fmt.Errorf("contextFrom requires unique explicit source IDs")
			}
			seen[id] = true
		}
	}
	if p.WatchSet && p.Watch != nil {
		w := p.Watch
		if w.Kind != "url" && w.Kind != "command" {
			return fmt.Errorf("watch kind must be url or command")
		}
		if strings.TrimSpace(w.Value) == "" || len(utf16.Encode([]rune(strings.TrimSpace(w.Value)))) > 2000 {
			return fmt.Errorf("watch target must contain 1–2000 UTF-16 units")
		}
		if w.TimeoutMS < 1 || w.TimeoutMS > 300000 {
			return fmt.Errorf("watch timeout must be 1–300000 milliseconds")
		}
	}
	return nil
}
func applyRoutineAdvancedPayload(payload map[string]any, p DshbotRoutineAdvancedPatch) {
	if p.ContextSet {
		payload["contextFrom"] = append([]string{}, p.ContextFrom...)
	}
	if p.SilentSet {
		payload["silentAllowed"] = p.SilentAllowed
	}
	if p.WatchSet {
		payload["watch"] = nil
		if p.Watch != nil {
			w := *p.Watch
			w.Value = strings.TrimSpace(w.Value)
			payload["watch"] = w
		}
	}
}
func validateRoutineAdvancedReceipt(p DshbotRoutineAdvancedPatch, r DshbotRoutine) error {
	if p.ContextSet && !slices.Equal(p.ContextFrom, r.ContextFrom) {
		return fmt.Errorf("Host returned different context sources; refresh before continuing")
	}
	if p.SilentSet && p.SilentAllowed != r.SilentAllowed {
		return fmt.Errorf("Host returned a different silent-response setting")
	}
	if p.WatchSet {
		raw := strings.TrimSpace(string(r.Watch))
		if p.Watch == nil {
			if raw != "" && raw != "null" {
				return fmt.Errorf("Host did not confirm watch removal")
			}
		} else {
			var got DshbotRoutineWatch
			if json.Unmarshal(r.Watch, &got) != nil || got.Kind != p.Watch.Kind || got.Value != strings.TrimSpace(p.Watch.Value) || got.TimeoutMS != p.Watch.TimeoutMS {
				return fmt.Errorf("Host returned a different watch configuration")
			}
		}
	}
	return nil
}
