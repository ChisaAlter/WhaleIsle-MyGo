package projectflow

import (
	"context"
	"fmt"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The original Project store/read enforces the same UTF-8 byte limit.
const DocumentReadLimit = 512000

type DocumentEntry struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}
type DocumentContent struct {
	Path string `json:"path"`
	Text string `json:"text"`
}

// DocumentReadError is deliberately platform independent: these are Host
// material identifiers, never paths to open, stat or execute on this computer.
func DocumentReadError(e DocumentEntry) error {
	if !strings.HasPrefix(e.Path, "docs/") || path.Clean(e.Path) != e.Path || strings.ContainsAny(e.Path, "\\:%") || strings.HasSuffix(e.Path, "/") || !utf8.ValidString(e.Path) {
		return fmt.Errorf("Only a listed Project docs file can be read")
	}
	for _, r := range e.Path {
		if unicode.IsControl(r) {
			return fmt.Errorf("Invalid Project document path")
		}
	}
	for _, part := range strings.Split(e.Path, "/") {
		if part == "" || part == "." || part == ".." || strings.EqualFold(part, "internal") || strings.EqualFold(part, ".git") {
			return fmt.Errorf("This material is outside the readable Project docs scope")
		}
	}
	if e.Size < 0 || e.Size > DocumentReadLimit {
		return fmt.Errorf("This document exceeds the Host's 512000-byte text read limit")
	}
	return nil
}
func validateDocumentInventory(docs []DocumentEntry) error {
	if docs == nil {
		return fmt.Errorf("Host returned no Project document inventory")
	}
	seen := map[string]bool{}
	for _, e := range docs {
		if e.Path == "" || seen[e.Path] {
			return fmt.Errorf("Host returned an ambiguous Project document identity")
		}
		seen[e.Path] = true
	}
	return nil
}
func (c Client) Documents(ctx context.Context, projectID string) ([]DocumentEntry, error) {
	if projectID == "" {
		return nil, fmt.Errorf("Select a Project")
	}
	var out struct {
		Docs []DocumentEntry `json:"docs"`
	}
	if err := c.call(ctx, "store/list", map[string]any{"projectId": projectID}, &out); err != nil {
		return nil, err
	}
	if err := validateDocumentInventory(out.Docs); err != nil {
		return nil, err
	}
	return out.Docs, nil
}
func (c Client) ReadDocument(ctx context.Context, projectID string, entry DocumentEntry) (DocumentContent, error) {
	var out DocumentContent
	if projectID == "" {
		return out, fmt.Errorf("Select a Project")
	}
	if err := DocumentReadError(entry); err != nil {
		return out, err
	}
	var wire struct {
		Path string  `json:"path"`
		Text *string `json:"text"`
	}
	if err := c.call(ctx, "store/read", map[string]any{"projectId": projectID, "path": entry.Path}, &wire); err != nil {
		return DocumentContent{}, err
	}
	if wire.Text == nil {
		return DocumentContent{}, fmt.Errorf("Host returned no Project document body")
	}
	out = DocumentContent{Path: wire.Path, Text: *wire.Text}
	if err := validateDocumentContent(entry.Path, out); err != nil {
		return DocumentContent{}, err
	}
	return out, nil
}
func validateDocumentContent(requested string, content DocumentContent) error {
	if content.Path != requested {
		return fmt.Errorf("Host returned a different Project document")
	}
	if len(content.Text) > DocumentReadLimit {
		return fmt.Errorf("Document now exceeds the 512000-byte text read limit; it was not truncated")
	}
	if !utf8.ValidString(content.Text) || strings.ContainsRune(content.Text, 0) {
		return fmt.Errorf("This document cannot be displayed as plain UTF-8 text")
	}
	return nil
}

// Host is an opaque caller-supplied identity, not an address used for routing.
// Generation changes on replacement of the authenticated Host connection.
type DocumentScope struct {
	Host, ProjectID string
	Generation      uint64
}
type DocumentSnapshot struct {
	Scope                     DocumentScope
	Title                     string
	Docs                      []DocumentEntry
	Loaded, Busy              bool
	SelectedPath, Text, Error string
	TextLoaded                bool
}
type DocumentRequest struct {
	owner  *DocumentBrowser
	scope  DocumentScope
	serial uint64
	kind   string
	entry  DocumentEntry
}

func (r DocumentRequest) ProjectID() string    { return r.scope.ProjectID }
func (r DocumentRequest) Entry() DocumentEntry { return r.entry }

type DocumentBrowser struct {
	state  DocumentSnapshot
	serial uint64
	active bool
}

func NewDocumentBrowser(scope DocumentScope, title string) *DocumentBrowser {
	return &DocumentBrowser{state: DocumentSnapshot{Scope: scope, Title: title}, active: true}
}
func (b *DocumentBrowser) Snapshot() DocumentSnapshot {
	s := b.state
	s.Docs = append([]DocumentEntry{}, s.Docs...)
	return s
}
func (b *DocumentBrowser) begin(kind string, entry DocumentEntry) (DocumentRequest, error) {
	if !b.active || b.state.Scope.Host == "" || b.state.Scope.ProjectID == "" {
		return DocumentRequest{}, fmt.Errorf("Project document reader is closed")
	}
	if b.state.Busy {
		return DocumentRequest{}, fmt.Errorf("A Project document request is already pending")
	}
	b.serial++
	b.state.Busy = true
	b.state.Error = ""
	return DocumentRequest{owner: b, scope: b.state.Scope, serial: b.serial, kind: kind, entry: entry}, nil
}
func (b *DocumentBrowser) BeginList() (DocumentRequest, error) {
	return b.begin("list", DocumentEntry{})
}
func (b *DocumentBrowser) BeginRead(documentPath string) (DocumentRequest, error) {
	for _, entry := range b.state.Docs {
		if entry.Path != documentPath {
			continue
		}
		if err := DocumentReadError(entry); err != nil {
			b.state.Error = err.Error()
			return DocumentRequest{}, err
		}
		r, err := b.begin("read", entry)
		if err != nil {
			return r, err
		}
		if b.state.SelectedPath != documentPath {
			b.state.Text = ""
			b.state.TextLoaded = false
		}
		b.state.SelectedPath = documentPath
		return r, nil
	}
	return DocumentRequest{}, fmt.Errorf("Document is not in this Project's current inventory")
}
func (b *DocumentBrowser) owns(r DocumentRequest, kind string) bool {
	return r.owner == b && b.active && b.state.Busy && b.state.Scope == r.scope && b.serial == r.serial && r.kind == kind
}

// Pending rejects completions after Back, close, a newer read, or a new reader.
func (b *DocumentBrowser) Pending(r DocumentRequest) bool { return b.owns(r, r.kind) }
func (b *DocumentBrowser) CompleteList(r DocumentRequest, docs []DocumentEntry, err error) bool {
	if !b.owns(r, "list") {
		return false
	}
	b.state.Busy = false
	if err == nil {
		err = validateDocumentInventory(docs)
	}
	if err != nil {
		b.state.Error = err.Error()
		return true
	}
	b.state.Docs = append([]DocumentEntry{}, docs...)
	b.state.Loaded = true
	b.state.SelectedPath = ""
	b.state.Text = ""
	b.state.TextLoaded = false
	b.state.Error = ""
	return true
}
func (b *DocumentBrowser) CompleteRead(r DocumentRequest, content DocumentContent, err error) bool {
	if !b.owns(r, "read") || b.state.SelectedPath != r.entry.Path {
		return false
	}
	b.state.Busy = false
	if err == nil {
		err = validateDocumentContent(r.entry.Path, content)
	}
	if err != nil {
		b.state.Error = err.Error()
		return true
	}
	b.state.Text = content.Text
	b.state.TextLoaded = true
	b.state.Error = ""
	return true
}

// Back invalidates outstanding work without erasing the readable inventory.
func (b *DocumentBrowser) Back() {
	b.serial++
	b.state.Busy = false
	b.state.SelectedPath = ""
	b.state.Text = ""
	b.state.TextLoaded = false
	b.state.Error = ""
}
func (b *DocumentBrowser) Close() { b.Back(); b.active = false }
