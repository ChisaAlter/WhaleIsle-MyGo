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
	if d.Sending || d.Preparing > 0 || d.EditSeq > 0 || strings.TrimSpace(d.Text) == "" && len(d.Attachments) == 0 {
		return DraftFlight{}, false
	}
	d.flightSeq++
	f := DraftFlight{ID: d.flightSeq, Text: d.Text, Attachments: append([]session.ComposerAttachment(nil), d.Attachments...)}
	if d.failedRestoredMatches() {
		d.failedSends = nil
	}
	if d.flights == nil {
		d.flights = make(map[uint64]DraftFlight)
	}
	d.flights[f.ID] = f
	d.Text, d.Attachments, d.Error = "", nil, ""
	d.failedRestored = nil
	d.Client.Sync()
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
	f.Error = err.Error()
	f.Attachments = append([]session.ComposerAttachment(nil), attachments...)
	if d.failedSends == nil {
		d.failedSends = make(map[uint64]DraftFlight)
	}
	d.failedSends[id] = f
	d.Error = f.Error
	if d.failedRestoredMatches() || d.Text == "" && len(d.Attachments) == 0 {
		d.RestoreFailedSends()
	}
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
	if d.Sending || d.Preparing > 0 || d.EditSeq > 0 || len(d.failedSends) == 0 {
		return
	}
	text, attachments := d.Text, append([]session.ComposerAttachment(nil), d.Attachments...)
	if d.failedRestoredMatches() {
		text, attachments = "", nil
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
	files = append(files, attachments...)
	d.Text, d.Attachments = strings.Join(texts, "\n\n"), files
	d.Client.Sync()
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
