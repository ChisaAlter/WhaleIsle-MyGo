package composer

import (
	"reflect"
	"sort"
	"strings"
	"whaleisle/internal/store/session"
)

// DraftFlight owns one detached editor snapshot. UI updates settle by its ID,
// never by comparing against a newer text or attachment list.
type DraftFlight struct {
	ID          uint64
	Text        string
	Attachments []session.ComposerAttachment
	Error       string
}

func (d *DraftState) BeginDetachedSend() (DraftFlight, bool) {
	if d.Sending || d.Preparing > 0 || d.FailedAttachmentError() != "" || d.EditSeq > 0 || strings.TrimSpace(d.Text) == "" && len(d.Attachments) == 0 {
		return DraftFlight{}, false
	}
	// Assigning a missing occurrence ID is normalization, not a user edit of
	// the recovered payload. Compare with that payload before allocating IDs.
	restoredUnchanged := d.failedRestoredMatches()
	// Restored or programmatically attached files may not have gone through
	// preparation yet. Give each occurrence an identity before detaching it.
	for i := range d.Attachments {
		if d.Attachments[i].PreparationID == 0 {
			d.Attachments[i].PreparationID = attachmentAttempt.Add(1)
		}
	}
	d.flightSeq++
	f := DraftFlight{ID: d.flightSeq, Text: d.Text, Attachments: append([]session.ComposerAttachment(nil), d.Attachments...)}
	if restoredUnchanged {
		d.failedSends = nil
	}
	if d.flights == nil {
		d.flights = make(map[uint64]DraftFlight)
	}
	d.flights[f.ID] = f
	d.Text, d.Attachments, d.Error = "", nil, ""
	d.failedRestored = nil
	d.Client.ClearHistory()
	return f, true
}
func (d *DraftState) CompleteDetachedSend(id uint64, attachments []session.ComposerAttachment, err error) {
	f, ok := d.flights[id]
	if !ok {
		return
	}
	delete(d.flights, id)
	if err == nil {
		return
	}
	restoredUnchanged := d.failedRestoredMatches()
	restoreText := restoredUnchanged || d.Text == "" && len(d.Attachments) == 0
	f.Error = err.Error()
	f.Attachments = append([]session.ComposerAttachment(nil), attachments...)
	if d.failedSends == nil {
		d.failedSends = make(map[uint64]DraftFlight)
	}
	// Like the original facade, return failed files to the rail even while a
	// newer text draft is protected. The rail then owns those occurrences;
	// retaining another copy in the text-recovery record would resend them twice.
	d.restoreFailedAttachments(f.Attachments)
	if restoredUnchanged {
		d.failedRestored.Attachments = append([]session.ComposerAttachment(nil), d.Attachments...)
	}
	f.Attachments = nil
	d.failedSends[id] = f
	d.Error = f.Error
	if restoreText {
		d.RestoreFailedSends()
	}
}

func (d *DraftState) restoreFailedAttachments(attachments []session.ComposerAttachment) {
	seen := make(map[uint64]bool, len(d.Attachments)+len(attachments))
	for _, a := range d.Attachments {
		if a.PreparationID != 0 {
			seen[a.PreparationID] = true
		}
	}
	var restored []session.ComposerAttachment
	for _, a := range attachments {
		if a.PreparationID != 0 && seen[a.PreparationID] {
			continue
		}
		if a.PreparationID == 0 {
			a.PreparationID = attachmentAttempt.Add(1)
		}
		seen[a.PreparationID] = true
		restored = append(restored, a)
	}
	d.Attachments = append(restored, d.Attachments...)
}
func (d *DraftState) failedRestoredMatches() bool {
	return d.failedRestored != nil && d.Text == d.failedRestored.Text && reflect.DeepEqual(d.Attachments, d.failedRestored.Attachments)
}
func (d *DraftState) FailedSendCount() int      { return len(d.failedSends) }
func (d *DraftState) FailedSendsRestored() bool { return d.failedRestoredMatches() }
func orderedDraftFlights(entries map[uint64]DraftFlight) []DraftFlight {
	out := make([]DraftFlight, 0, len(entries))
	for _, f := range entries {
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// RestoreFailedSends keeps any newly typed draft after the unsent messages.
func (d *DraftState) RestoreFailedSends() {
	if d.Sending || d.Preparing > 0 || d.FailedAttachmentError() != "" || d.EditSeq > 0 || len(d.failedSends) == 0 {
		return
	}
	text := d.Text
	if d.failedRestoredMatches() {
		text = ""
	}
	var texts []string
	var files []session.ComposerAttachment
	for _, f := range orderedDraftFlights(d.failedSends) {
		if f.Text != "" {
			texts = append(texts, f.Text)
		}
		files = append(files, f.Attachments...)
	}
	if text != "" {
		texts = append(texts, text)
	}
	d.restoreFailedAttachments(files)
	for id, f := range d.failedSends {
		f.Attachments = nil
		d.failedSends[id] = f
	}
	d.Text = strings.Join(texts, "\n\n")
	d.Client.ClearHistory()
	d.failedRestored = &DraftFlight{Text: d.Text, Attachments: append([]session.ComposerAttachment(nil), d.Attachments...)}
}

type savedComposerFile struct{ Path, Name string }
type savedComposerSend struct {
	ID          uint64
	Text, Error string
	Files       []savedComposerFile
}

func (d *DraftState) recoverySends() []savedComposerSend {
	entries := make(map[uint64]DraftFlight)
	for id, f := range d.flights {
		f.Error = "发送结果未确认，请先查看会话 / Send outcome unconfirmed; check the conversation before retrying"
		entries[id] = f
	}
	if !d.failedRestoredMatches() {
		for id, f := range d.failedSends {
			entries[id] = f
		}
	}
	var out []savedComposerSend
	for _, f := range orderedDraftFlights(entries) {
		saved := savedComposerSend{ID: f.ID, Text: f.Text, Error: f.Error}
		for _, a := range f.Attachments {
			saved.Files = append(saved.Files, savedComposerFile{Path: a.Path, Name: a.Name})
		}
		out = append(out, saved)
	}
	return out
}
func (d *DraftState) restoreRecoverySends(entries []savedComposerSend) {
	for _, entry := range entries {
		d.flightSeq++
		f := DraftFlight{ID: d.flightSeq, Text: entry.Text, Error: entry.Error}
		for _, a := range entry.Files {
			if a.Path != "" {
				f.Attachments = append(f.Attachments, session.ComposerAttachment{Path: a.Path, Name: a.Name})
			}
		}
		if d.failedSends == nil {
			d.failedSends = make(map[uint64]DraftFlight)
		}
		d.failedSends[f.ID] = f
		d.Error = f.Error
	}
}
