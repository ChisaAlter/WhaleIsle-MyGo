package composer

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"whaleisle/internal/store/session"
)

// DraftState is a stable UI-owned draft. Async work returns through Window.Update.
// Each session keeps its own text input client, selection, attachments and error.
type DraftState struct {
	Suggestions        SuggestionState
	GroupReply         *GroupReplyDraft
	ModelCommand       ModelCommandState
	PermissionCommand  ModelCommandState
	VoiceSetupOpen     bool
	flightSeq          uint64
	flights            map[uint64]DraftFlight
	failedSends        map[uint64]DraftFlight
	failedRestored     *DraftFlight
	Text               string
	Attachments        []session.ComposerAttachment
	Client             *DraftClient
	Sending            bool
	Preparing          int
	attachmentCancels  map[uint64]context.CancelFunc
	Error              string
	TargetID           string
	EditSeq            float64
	EditSessionID      string
	EditReturnSeq      float64
	EditFocus          bool
	VoiceStarted       time.Time // recording clock, reset when the phase leaves recording
	VoiceTick          time.Time // last tick that scheduled a clock repaint
	stashedText        string
	stashedReferences  []draftReference
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
		stashed := NewDraftState(d.stashedText)
		stashed.Attachments = d.stashedAttachments
		stashed.Client.references = append([]draftReference(nil), d.stashedReferences...)
		return stashed.SaveDraft()
	}
	type file struct {
		Path string
		Name string
	}
	var references []draftReference
	if d.Client != nil && d.Client.Draft == &d.Text {
		d.Client.Sync()
		references = append([]draftReference{}, d.Client.references...)
	}
	value := struct {
		Text       string
		Files      []file
		Sends      []savedComposerSend
		References []draftReference
	}{Text: d.Text, Sends: d.recoverySends(), References: references}
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
		Text       *string
		Sends      []savedComposerSend
		References *[]draftReference
		Files      []struct {
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
	if value.References != nil {
		references, err := checkedDraftReferences(d.Text, *value.References)
		if err != nil {
			return nil, err
		}
		d.Client.references = references
		d.Client.shape()
	}
	for _, a := range value.Files {
		if strings.TrimSpace(a.Path) == "" {
			return nil, errors.New("saved attachment is missing its path")
		}
		d.Attachments = append(d.Attachments, session.ComposerAttachment{Path: a.Path, Name: a.Name})
	}
	d.restoreRecoverySends(value.Sends)
	return d, nil
}
