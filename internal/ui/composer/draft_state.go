package composer

import (
	"encoding/json"
	"errors"
	"strings"
	"whaleisle/internal/store/session"
)

// DraftState is a stable UI-owned draft. Async work returns through Window.Update.
// Each session keeps its own text input client, selection, attachments and error.
type DraftState struct {
	ModelCommand       ModelCommandState
	flightSeq          uint64
	flights            map[uint64]DraftFlight
	failedSends        map[uint64]DraftFlight
	failedRestored     *DraftFlight
	Text               string
	Attachments        []session.ComposerAttachment
	Client             *DraftClient
	Sending            bool
	Preparing          int
	Error              string
	TargetID           string
	EditSeq            float64
	EditSessionID      string
	EditReturnSeq      float64
	EditFocus          bool
	stashedText        string
	stashedAttachments []session.ComposerAttachment
}

func NewDraftState(text string) *DraftState {
	d := &DraftState{Text: text}
	d.Client = NewDraftClient(&d.Text)
	return d
}

// SaveDraft persists only user inputs. Staged Host receipts and image bytes are
// deliberately excluded, since a restarted Host needs a fresh upload.
func (d *DraftState) SaveDraft() json.RawMessage {
	if d.EditSeq > 0 {
		return (&DraftState{Text: d.stashedText, Attachments: d.stashedAttachments}).SaveDraft()
	}
	type file struct {
		Path string
		Name string
	}
	value := struct {
		Text  string
		Files []file
		Sends []savedComposerSend
	}{Text: d.Text, Sends: d.recoverySends()}
	for _, a := range d.Attachments {
		value.Files = append(value.Files, file{a.Path, a.Name})
	}
	raw, _ := json.Marshal(value)
	return raw
}

func RestoreDraft(raw json.RawMessage) *DraftState {
	d, err := RestoreDraftChecked(raw)
	if err != nil {
		return NewDraftState("")
	}
	return d
}

// RestoreDraftChecked lets durable recovery retain a corrupt snapshot instead
// of accidentally replacing it with an empty draft on the next save.
func RestoreDraftChecked(raw json.RawMessage) (*DraftState, error) {
	var value struct {
		Text  *string
		Sends []savedComposerSend
		Files []struct {
			Path string
			Name string
		}
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	if value.Text == nil {
		return nil, errors.New("saved draft is missing Text")
	}
	d := NewDraftState(*value.Text)
	for _, a := range value.Files {
		if strings.TrimSpace(a.Path) == "" {
			return nil, errors.New("saved attachment is missing its path")
		}
		d.Attachments = append(d.Attachments, session.ComposerAttachment{Path: a.Path, Name: a.Name})
	}
	d.restoreRecoverySends(value.Sends)
	return d, nil
}
