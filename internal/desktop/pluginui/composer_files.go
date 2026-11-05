package pluginui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
)

const composerFilesPath = "/__whale_plugin_ui/composer/files"
const composerFilePath = "/__whale_plugin_ui/composer/file"
const composerMetadataLimit = 16 * 1024

type composerTransferRequest struct {
	Page  uint64          `json:"page"`
	Owner ComposerBinding `json:"owner"`
	ID    uint64          `json:"id,omitempty"`
}

func (o *Owner) composerTransferCurrent(page *bridgePage, owner ComposerBinding) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return !o.closed && !o.incompatible && o.page == page && page != nil && o.selected.equal(owner.Subject) && owner.Subject.Kind == "session" && owner.Subject.SessionID != ""
}

func readComposerMetadata(r io.Reader, out any) error {
	b, err := io.ReadAll(io.LimitReader(r, composerMetadataLimit+1))
	if err != nil || len(b) > composerMetadataLimit {
		return errors.New("Plugin attachment metadata is too large")
	}
	if err := json.Unmarshal(b, out); err != nil {
		return errors.New("Invalid plugin attachment metadata")
	}
	return nil
}

func (o *Owner) composerTransferFailure(w http.ResponseWriter, page *bridgePage, owner ComposerBinding, err error, status int) {
	o.mu.Lock()
	if !o.closed && o.page == page && o.selected.equal(owner.Subject) {
		o.composerError = err.Error()
	}
	o.mu.Unlock()
	o.notify()
	http.Error(w, err.Error(), status)
}

// These routes run behind the existing exact-Host Gateway's ephemeral cookie.
// File bytes use HTTP streaming; the general socket remains capped at 512 KiB.
func (o *Owner) serveComposerFiles(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != composerFilesPath && r.URL.Path != composerFilePath {
		return false
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method != http.MethodPost || r.URL.RawQuery != "" || r.Header.Get("Origin") != "http://"+r.Host {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return true
	}
	var request composerTransferRequest
	var parts *multipart.Reader
	var err error
	if r.URL.Path == composerFilesPath {
		parts, err = r.MultipartReader()
		if err == nil {
			var first *multipart.Part
			first, err = parts.NextPart()
			if err == nil {
				if first.FormName() != "owner" {
					err = errors.New("Missing plugin attachment owner")
				} else {
					err = readComposerMetadata(first, &request)
				}
				_ = first.Close()
			}
		}
	} else {
		if strings.Split(r.Header.Get("Content-Type"), ";")[0] != "application/json" {
			err = errors.New("Invalid plugin attachment request")
		} else {
			err = readComposerMetadata(r.Body, &request)
		}
	}
	if err != nil {
		http.Error(w, "Invalid plugin attachment request", http.StatusBadRequest)
		return true
	}
	o.mu.Lock()
	page, bridge := o.page, o.composer
	o.mu.Unlock()
	transfer, supported := bridge.(ComposerFileTransfer)
	if !supported || !o.composerTransferCurrent(page, request.Owner) || page.id != request.Page {
		http.Error(w, "Plugin attachment owner expired", http.StatusConflict)
		return true
	}
	draft, err := bridge.Read(request.Owner.Subject)
	if err == nil && draft.Owner != request.Owner {
		err = errors.New("The composer changed. Read it again before retrying.")
	}
	if err != nil {
		o.composerTransferFailure(w, page, request.Owner, err, http.StatusConflict)
		return true
	}
	current := func() bool { return o.composerTransferCurrent(page, request.Owner) }
	if parts != nil {
		iterator := &composerMultipartFiles{ctx: r.Context(), parts: parts, current: current}
		err = transfer.ImportFiles(r.Context(), request.Owner, iterator)
		if err == nil && !iterator.finished {
			err = errors.New("Plugin attachment import did not consume its batch")
		}
		if err == nil && !current() {
			err = errors.New("Plugin attachment owner expired")
		}
		if err == nil {
			draft, err = bridge.Read(request.Owner.Subject)
		}
		if err == nil && !draft.Owner.Subject.equal(request.Owner.Subject) {
			err = errors.New("Plugin attachment owner expired")
		}
		if err != nil {
			o.composerTransferFailure(w, page, request.Owner, err, http.StatusConflict)
			return true
		}
		o.mu.Lock()
		if o.page == page && o.selected.equal(request.Owner.Subject) {
			o.composerError = ""
		}
		o.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"page": page.id, "draft": draft})
		o.notify()
		return true
	}
	if request.ID == 0 {
		http.Error(w, "Invalid attachment ID", http.StatusBadRequest)
		return true
	}
	found := false
	for _, attachment := range draft.Attachments {
		if attachment.ID == request.ID {
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "Attachment no longer belongs to this draft", http.StatusConflict)
		return true
	}
	file, err := transfer.ReadFile(r.Context(), request.Owner, request.ID)
	if err != nil {
		o.composerTransferFailure(w, page, request.Owner, err, http.StatusConflict)
		return true
	}
	if file.Content == nil {
		http.Error(w, "Attachment content unavailable", http.StatusNotFound)
		return true
	}
	defer file.Content.Close()
	// Recheck after opening, before any bytes or headers escape this owner.
	draft, err = bridge.Read(request.Owner.Subject)
	if err != nil || draft.Owner != request.Owner || !current() {
		http.Error(w, "Plugin attachment owner expired", http.StatusConflict)
		return true
	}
	mediaType, _, err := mime.ParseMediaType(file.MediaType)
	if err != nil {
		mediaType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", mediaType)
	// Always delivered to fetch as bytes, never as an active top-level document.
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": file.Name}))
	if file.Size >= 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(file.Size, 10))
	}
	if _, err := io.CopyBuffer(w, &composerCurrentReader{ctx: r.Context(), source: file.Content, current: current}, make([]byte, 64*1024)); err != nil {
		panic(http.ErrAbortHandler) // fetch must not accept a truncated preview as a complete File.
	}
	return true
}

type composerCurrentReader struct {
	ctx     context.Context
	source  io.Reader
	current func() bool
}

func (r *composerCurrentReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	if !r.current() {
		return 0, errors.New("Plugin attachment owner expired")
	}
	return r.source.Read(p)
}

type composerMultipartFiles struct {
	ctx      context.Context
	parts    *multipart.Reader
	current  func() bool
	previous *composerPartFile
	finished bool
}

func (f *composerMultipartFiles) Next() (ComposerFileContent, error) {
	if f.previous != nil && !f.previous.done {
		return ComposerFileContent{}, errors.New("Plugin attachment stream was not consumed")
	}
	part, err := f.parts.NextPart()
	if err == io.EOF {
		f.finished = true
		return ComposerFileContent{}, io.EOF
	}
	if err != nil {
		return ComposerFileContent{}, err
	}
	if part.FormName() != "metadata" {
		return ComposerFileContent{}, errors.New("Missing plugin file metadata")
	}
	var metadata struct {
		Name      string `json:"name"`
		MediaType string `json:"mediaType"`
		Size      int64  `json:"size"`
	}
	err = readComposerMetadata(part, &metadata)
	_ = part.Close()
	if err != nil {
		return ComposerFileContent{}, err
	}
	if metadata.Name == "" || strings.ContainsAny(metadata.Name, "\x00\r\n") || metadata.Size < 0 {
		return ComposerFileContent{}, errors.New("Invalid plugin file metadata")
	}
	part, err = f.parts.NextPart()
	if err != nil {
		return ComposerFileContent{}, err
	}
	if part.FormName() != "file" {
		return ComposerFileContent{}, errors.New("Missing plugin file bytes")
	}
	stream := &composerPartFile{reader: composerCurrentReader{ctx: f.ctx, source: part, current: f.current}, part: part, remaining: metadata.Size}
	f.previous = stream
	return ComposerFileContent{Name: metadata.Name, MediaType: metadata.MediaType, Size: metadata.Size, Content: stream}, nil
}

type composerPartFile struct {
	reader    composerCurrentReader
	part      *multipart.Part
	remaining int64
	done      bool
}

func (f *composerPartFile) Read(p []byte) (int, error) {
	n, err := f.reader.Read(p)
	f.remaining -= int64(n)
	if f.remaining < 0 {
		return 0, errors.New("Plugin attachment exceeds its declared size")
	}
	if err == io.EOF {
		f.done = true
		if f.remaining != 0 {
			return n, fmt.Errorf("Plugin attachment is missing %d bytes", f.remaining)
		}
	}
	return n, err
}
func (f *composerPartFile) Close() error { return f.part.Close() }
