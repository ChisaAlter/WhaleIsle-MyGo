package dshbot

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"whaleisle/internal/desktop/pluginrpc"
)

type RoutineAdvancedDraft struct {
	ContextFrom                                   []string
	SilentAllowed                                 bool
	WatchKind, WatchValue, WatchTimeout, WatchRaw string
	WatchUnknown                                  bool
	ContextChanged, SilentChanged, WatchChanged   bool
}

func cloneRoutineAdvanced(d RoutineAdvancedDraft) RoutineAdvancedDraft {
	d.ContextFrom = append([]string(nil), d.ContextFrom...)
	return d
}
func routineAdvancedFromRecord(r pluginrpc.DshbotRoutine) RoutineAdvancedDraft {
	d := RoutineAdvancedDraft{ContextFrom: append([]string(nil), r.ContextFrom...), SilentAllowed: r.SilentAllowed, WatchTimeout: "15000", WatchRaw: strings.TrimSpace(string(r.Watch))}
	if d.WatchRaw == "" || d.WatchRaw == "null" {
		return d
	}
	var w struct {
		Kind      string `json:"kind"`
		Value     string `json:"value"`
		TimeoutMS *int   `json:"timeoutMs"`
	}
	if json.Unmarshal(r.Watch, &w) != nil || (w.Kind != "url" && w.Kind != "command") {
		d.WatchUnknown = true
		return d
	}
	d.WatchKind = w.Kind
	d.WatchValue = w.Value
	if w.TimeoutMS != nil {
		d.WatchTimeout = strconv.Itoa(*w.TimeoutMS)
	}
	return d
}
func routineContextCandidate(id, botID, currentID string, rows []pluginrpc.DshbotRoutine) bool {
	if id == "self" {
		return true
	}
	for _, r := range rows {
		if r.ID == id && r.ID != currentID && r.BotID == botID {
			return true
		}
	}
	return false
}
func editRoutineAdvanced(d *RoutineAdvancedDraft, field, value, botID, currentID string, rows []pluginrpc.DshbotRoutine) (bool, error) {
	switch field {
	case "contextToggle":
		index := slices.Index(d.ContextFrom, value)
		if index >= 0 {
			d.ContextFrom = append(append([]string(nil), d.ContextFrom[:index]...), d.ContextFrom[index+1:]...)
		} else {
			if !routineContextCandidate(value, botID, currentID, rows) {
				return false, fmt.Errorf("上下文来源不属于此 Bot / Context source is not owned by this Bot")
			}
			if len(d.ContextFrom) >= 3 {
				return false, fmt.Errorf("最多选择 3 个上下文来源 / At most 3 context sources")
			}
			d.ContextFrom = append(append([]string(nil), d.ContextFrom...), value)
		}
		d.ContextChanged = true
	case "silentAllowed":
		d.SilentAllowed = value == "true"
		d.SilentChanged = true
	case "watchKind":
		if value != "" && value != "url" && value != "command" {
			return false, fmt.Errorf("不支持的 watch 类型 / Unsupported watch kind")
		}
		d.WatchKind = value
		d.WatchUnknown = false
		d.WatchChanged = true
	case "watchValue":
		if d.WatchUnknown {
			return false, fmt.Errorf("未知 watch 配置须先明确选择已支持类型 / Select a supported watch type first")
		}
		d.WatchValue = value
		d.WatchChanged = true
	case "watchTimeout":
		if d.WatchUnknown {
			return false, fmt.Errorf("未知 watch 配置须先明确选择已支持类型 / Select a supported watch type first")
		}
		d.WatchTimeout = value
		d.WatchChanged = true
	default:
		return false, nil
	}
	return true, nil
}
func prepareRoutineAdvanced(d RoutineAdvancedDraft, botID, currentID string, rows []pluginrpc.DshbotRoutine) (pluginrpc.DshbotRoutineAdvancedPatch, error) {
	p := pluginrpc.DshbotRoutineAdvancedPatch{ContextSet: d.ContextChanged, SilentSet: d.SilentChanged, WatchSet: d.WatchChanged, SilentAllowed: d.SilentAllowed}
	// The Host normalizes even omitted contextFrom during save. Refuse to save
	// an unresolved list rather than silently dropping an untouched source.
	contextCheck := pluginrpc.DshbotRoutineAdvancedPatch{ContextSet: true, ContextFrom: d.ContextFrom}
	if err := pluginrpc.ValidateDshbotRoutineAdvanced(contextCheck); err != nil {
		return p, err
	}
	for _, id := range d.ContextFrom {
		if !routineContextCandidate(id, botID, currentID, rows) {
			return p, fmt.Errorf("上下文来源 %s 已不可用；明确移除后再审阅 / Remove the unavailable context source before review", id)
		}
	}
	if p.ContextSet {
		p.ContextFrom = append([]string{}, d.ContextFrom...)
	}
	if p.WatchSet && d.WatchKind != "" {
		timeout, err := strconv.Atoi(strings.TrimSpace(d.WatchTimeout))
		if err != nil {
			return p, fmt.Errorf("watch 超时须为整数毫秒 / Watch timeout must be integer milliseconds")
		}
		p.Watch = &pluginrpc.DshbotRoutineWatch{Kind: d.WatchKind, Value: strings.TrimSpace(d.WatchValue), TimeoutMS: timeout}
	}
	return p, pluginrpc.ValidateDshbotRoutineAdvanced(p)
}

func cloneRoutineDraft(d RoutineDraft) RoutineDraft {
	d.Advanced = cloneRoutineAdvanced(d.Advanced)
	return d
}
func cloneRoutineReview(r RoutineReview) RoutineReview {
	r.Advanced = cloneRoutineAdvanced(r.Advanced)
	r.Command.Advanced = pluginrpc.CloneDshbotRoutineAdvanced(r.Command.Advanced)
	return r
}
func RoutineReviewNeedsWatchAcknowledgement(r RoutineReview) bool {
	enabled := r.Command.Action == "save" && r.Command.Edit.Enabled || r.Command.Action == "toggle" && !r.Command.EnabledBefore
	return enabled && (r.Advanced.WatchKind == "command" || r.Advanced.WatchUnknown)
}
func (c *RoutineControls) AcknowledgeWatch(s RoutineScope, expected *RoutineReview, ack bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.valid(s) && !c.state.Busy && c.state.Review != nil && expected != nil && c.state.Review.Sequence == expected.Sequence {
		c.state.Review.WatchAcknowledged = ack
	}
}

// Show the exact canonical values submitted to the Host, keeping raw draft text
// intact so validation failures never destroy the user's input.
func routineAdvancedReviewValue(d RoutineAdvancedDraft, p pluginrpc.DshbotRoutineAdvancedPatch) RoutineAdvancedDraft {
	d = cloneRoutineAdvanced(d)
	if p.WatchSet {
		d.WatchUnknown = false
		if p.Watch == nil {
			d.WatchKind = ""
			d.WatchValue = ""
		} else {
			d.WatchKind = p.Watch.Kind
			d.WatchValue = strings.TrimSpace(p.Watch.Value)
			d.WatchTimeout = strconv.Itoa(p.Watch.TimeoutMS)
		}
	}
	return d
}
