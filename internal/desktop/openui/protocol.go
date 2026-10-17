// Package openui connects the embedded SDK-based OpenUI renderer to the isolated
// preview helper. Public resources never receive Host credentials.
package openui

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"
)

const MaxProgramBytes = 256 << 10
const MaxEventBytes = 64 << 10
const MaxStateBytes = 48 << 10
const MaxViewerBytes = 12 << 20

type Render struct {
	InstanceID string `json:"instanceId"`
	// Revision is assigned by the native owner, not inferred by the renderer.
	// Latest-only streaming delivery may skip revisions, but never go backwards.
	Revision  uint64          `json:"revision,omitempty"`
	Program   string          `json:"program"`
	Streaming bool            `json:"streaming"`
	State     json.RawMessage `json:"state"`
	// HostActions is a trusted Host opt-in, independently limited to the owned
	// embedded viewer by both the shell and helper-session event admission.
	HostActions bool `json:"hostActions,omitempty"`
	// Media enables only the owned viewer's typed, private native YouTube seat.
	// It does not grant network access or native bindings to the helper.
	Media bool `json:"media,omitempty"`
}
type Event struct {
	Media       *MediaRequest     `json:"media,omitempty"`
	Sequence    uint64            `json:"sequence,omitempty"`
	UserGesture bool              `json:"userGesture,omitempty"`
	Text        string            `json:"text,omitempty"`
	URL         string            `json:"url,omitempty"`
	RequestID   string            `json:"requestId,omitempty"`
	Path        string            `json:"path,omitempty"`
	Query       map[string]string `json:"query,omitempty"`
	Channel     string            `json:"channel"`
	Version     int               `json:"version"`
	InstanceID  string            `json:"instanceId"`
	Type        string            `json:"type"`
	State       json.RawMessage   `json:"state,omitempty"`
	Message     string            `json:"message,omitempty"`
	FormState   json.RawMessage   `json:"formState,omitempty"`
	Context     json.RawMessage   `json:"context,omitempty"`
	Program     string            `json:"program,omitempty"`
	Code        string            `json:"code,omitempty"`
}

func Identity(binding, session, message string, block int) string {
	raw, _ := json.Marshal([]any{binding, session, message, block})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func ValidateState(raw json.RawMessage) error {
	var object map[string]json.RawMessage
	if len(raw) > MaxStateBytes || json.Unmarshal(raw, &object) != nil || object == nil {
		return errors.New("invalid or oversized OpenUI state")
	}
	return nil
}
func ValidateRender(r Render) error {
	if r.Revision > 9007199254740991 || (r.Media && r.Revision == 0) {
		return errors.New("invalid OpenUI source revision")
	}
	if len(r.InstanceID) != 64 || len(r.Program) == 0 || len(r.Program) > MaxProgramBytes || !utf8.ValidString(r.Program) {
		return errors.New("invalid OpenUI instance/program")
	}
	if _, err := hex.DecodeString(r.InstanceID); err != nil {
		return errors.New("invalid OpenUI instance")
	}
	return ValidateState(r.State)
}
func ValidateEvent(raw json.RawMessage, instance string) (Event, error) {
	var e Event
	if len(raw) > MaxEventBytes || !utf8.Valid(raw) || json.Unmarshal(raw, &e) != nil || e.Channel != "whale-openui" || e.Version != 1 || e.InstanceID != instance {
		return e, errors.New("invalid OpenUI event envelope")
	}
	switch e.Type {
	case "media":
		if err := ValidateMediaRequest(e); err != nil {
			return e, err
		}
	case "clipboard", "open-url":
		if err := ValidateNativeAction(e); err != nil {
			return e, err
		}
	case "api-request":
		if err := validateAPIRequest(e); err != nil {
			return e, err
		}
	case "state":
		if err := ValidateState(e.State); err != nil {
			return e, err
		}
	case "assistant":
		if strings.TrimSpace(e.Message) == "" || len(e.Message) > 16<<10 {
			return e, errors.New("invalid OpenUI action")
		}
	case "error":
		if len(e.Message) > 4096 {
			return e, errors.New("oversized OpenUI error")
		}
	case "resolved":
		if len(e.Program) > MaxEventBytes/2 {
			return e, errors.New("oversized OpenUI resolved event")
		}
	default:
		return e, errors.New("unsupported OpenUI event")
	}
	return e, nil
}
func (e Event) ActionText() string {
	parts := []string{e.Message}
	if len(e.FormState) > 0 {
		parts = append(parts, "OpenUI form state:\n"+string(e.FormState))
	}
	if len(e.Context) > 0 {
		parts = append(parts, "OpenUI context:\n"+string(e.Context))
	}
	return strings.Join(parts, "\n\n")
}
