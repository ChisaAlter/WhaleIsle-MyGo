package previewprocess

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"time"

	"whaleisle/internal/desktop/openui"
)

// JSON escaping can expand a bounded 256 KiB program. State is deliberately
// absent: updates must not replace newer state held by the live viewer.
const maxOpenUIRenderBytes = 2 << 20
const openUIUpdateInterval = 100 * time.Millisecond

type openUIUpdate struct {
	InstanceID string `json:"instanceId"`
	Program    string `json:"program"`
	Streaming  bool   `json:"streaming"`
}

// Keep the original update shape for callers that use its three-field value.
// The revisioned wire embeds it so the JSON fields remain flat.
type revisionedOpenUIUpdate struct {
	openUIUpdate
	Revision  uint64            `json:"revision,omitempty"`
	Language  string            `json:"language,omitempty"`
	Locale    string            `json:"locale,omitempty"`
	Variables map[string]string `json:"variables,omitempty"`
	Panel     bool              `json:"panel,omitempty"`
}

func revisionedUpdate(r openui.Render) revisionedOpenUIUpdate {
	return revisionedOpenUIUpdate{openUIUpdate: openUIUpdate{r.InstanceID, r.Program, r.Streaming}, Revision: r.Revision, Language: r.Language, Locale: r.Locale, Variables: r.Variables, Panel: r.Panel}
}

type openUIUpdateQueue struct {
	mu              sync.Mutex
	instanceID      string
	capable, closed bool
	pending         []byte
	lastHash        [sha256.Size]byte
	lastSent        time.Time
	revision        uint64
}

func newOpenUIUpdateQueue(r openui.Render) *openUIUpdateQueue {
	raw, _ := json.Marshal(revisionedUpdate(r))
	return &openUIUpdateQueue{instanceID: r.InstanceID, lastHash: sha256.Sum256(raw), revision: r.Revision}
}

// UpdateOpenUI replaces only the pending program/streaming update for this
// helper instance. Admission is synchronous; delivery is rate-limited and a
// transport failure closes Done. State remains owned by the live parent shell.
func (s *Session) UpdateOpenUI(r openui.Render) error {
	if err := openui.ValidateRender(r); err != nil {
		return err
	}
	if s == nil || s.openUIUpdates == nil {
		return errors.New("OpenUI update channel unavailable")
	}
	q := s.openUIUpdates
	if r.InstanceID != q.instanceID {
		return errors.New("OpenUI update belongs to another instance")
	}
	raw, err := json.Marshal(revisionedUpdate(r))
	if err != nil {
		return err
	}
	if len(raw) > maxOpenUIRenderBytes {
		return errors.New("OpenUI update exceeds wire limit")
	}
	hash := sha256.Sum256(raw)
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return errors.New("OpenUI update channel closed")
	}
	if !q.capable {
		return errors.New("preview helper does not support OpenUI updates")
	}
	if hash == q.lastHash {
		return nil
	}
	if r.Revision < q.revision || (q.revision > 0 && r.Revision == q.revision) {
		return errors.New("OpenUI update does not advance its source revision")
	}
	q.pending = raw
	q.lastHash = hash
	q.revision = r.Revision
	return nil
}
func (s *Session) enableOpenUIUpdates(capable bool) {
	if q := s.openUIUpdates; q != nil {
		q.mu.Lock()
		q.capable = capable
		q.mu.Unlock()
	}
}
func (s *Session) takeOpenUIUpdate(now time.Time) []byte {
	q := s.openUIUpdates
	if q == nil {
		return nil
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed || !q.capable || len(q.pending) == 0 || (!q.lastSent.IsZero() && now.Sub(q.lastSent) < openUIUpdateInterval) {
		return nil
	}
	raw := q.pending
	q.pending = nil
	q.lastSent = now
	return raw
}
func (s *Session) closeOpenUIUpdates() {
	if q := s.openUIUpdates; q != nil {
		q.mu.Lock()
		q.closed = true
		q.pending = nil
		q.mu.Unlock()
	}
}
func startOpenUIUpdatePump(ctx context.Context, s *Session, send func([]byte) error) {
	if s.openUIUpdates == nil {
		return
	}
	go func() {
		tick := time.NewTicker(openUIUpdateInterval)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.Done():
				return
			case now := <-tick.C:
				raw := s.takeOpenUIUpdate(now)
				if len(raw) == 0 {
					continue
				}
				watchdog := time.AfterFunc(5*time.Second, func() { _ = s.Close() })
				err := send(raw)
				watchdog.Stop()
				if err != nil {
					_ = s.Close()
					return
				}
			}
		}
	}()
}

// validateOpenUIUpdate retains the original caller-facing value shape.
func validateOpenUIUpdate(raw []byte) (openUIUpdate, error) {
	update, err := validateRevisionedOpenUIUpdate(raw)
	return update.openUIUpdate, err
}

func validateRevisionedOpenUIUpdate(raw []byte) (revisionedOpenUIUpdate, error) {
	var update revisionedOpenUIUpdate
	if len(raw) < 1 || len(raw) > maxOpenUIRenderBytes {
		return update, errors.New("invalid OpenUI update length")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&update); err != nil {
		return update, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return update, errors.New("trailing OpenUI update JSON")
	}
	err := openui.ValidateRender(openui.Render{InstanceID: update.InstanceID, Revision: update.Revision, Program: update.Program, Streaming: update.Streaming, State: json.RawMessage(`{}`), Language: update.Language, Locale: update.Locale, Variables: update.Variables, Panel: update.Panel})
	return update, err
}
func sendOpenUIUpdate(w io.Writer, nonce string, raw []byte) error {
	if _, err := validateRevisionedOpenUIUpdate(raw); err != nil {
		return err
	}
	if err := writePacket(w, packet{Version: protocolVersion, Kind: "render", Nonce: nonce, Bytes: len(raw)}); err != nil {
		return err
	}
	return writeAll(w, raw)
}
func receiveOpenUIUpdate(r io.Reader, p packet, nonce string) ([]byte, error) {
	if err := expect(p, "render", nonce); err != nil {
		return nil, err
	}
	if p.Bytes < 1 || p.Bytes > maxOpenUIRenderBytes {
		return nil, errors.New("invalid OpenUI update byte count")
	}
	raw := make([]byte, p.Bytes)
	if _, err := io.ReadFull(r, raw); err != nil {
		return nil, err
	}
	update, err := validateRevisionedOpenUIUpdate(raw)
	if err != nil {
		return nil, err
	}
	// Marshal again: only fixed typed fields reach the fixed shell function.
	return json.Marshal(update)
}
func openUIUpdateScript(raw []byte) (string, error) {
	update, err := validateRevisionedOpenUIUpdate(raw)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(update)
	if err != nil {
		return "", err
	}
	return "window.__whaleOpenUIRender(" + string(encoded) + ");true", nil
}
