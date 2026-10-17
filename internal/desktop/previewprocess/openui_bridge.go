package previewprocess

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"sync"

	"whaleisle/internal/desktop/htmlpreview"
	"whaleisle/internal/desktop/openui"
)

const maxOpenUIEventBytes = 64 << 10
const maxOpenUIEvents = 16
const maxOpenUIBatchBytes = maxOpenUIEvents*(maxOpenUIEventBytes+1) + 2

// This expression runs in the trusted top-level shell, never the opaque iframe.
// It has no interpolated script, URL, credentials or native operation name.
const openUIDrainScript = "(()=>{if(window.__previewState&&window.__previewState.error)throw new Error('OpenUI shell failed');return JSON.stringify(window.__whaleOpenUIDrain());})()"

type openUIEventQueue struct {
	mu         sync.Mutex
	values     chan json.RawMessage
	instanceID string
	closed     bool
}

// Events carries only validated bounded renderer state/user-intent reports.
// It does not authorize a Host call. Consumers must recheck their owner ticket
// and apply their own action policy. Ordinary HTML sessions return nil.
func (s *Session) Events() <-chan json.RawMessage {
	if s == nil || s.openUIEvents == nil {
		return nil
	}
	return s.openUIEvents.values
}
func (s *Session) initOpenUIEvents(doc Document) {
	if doc.OpenUI != nil {
		s.openUIUpdates = newOpenUIUpdateQueue(*doc.OpenUI)
		s.openUIEvents = &openUIEventQueue{values: make(chan json.RawMessage, maxOpenUIEvents), instanceID: doc.OpenUI.InstanceID}
	}
}
func (s *Session) closeOpenUIEvents() {
	if q := s.openUIEvents; q != nil {
		q.mu.Lock()
		defer q.mu.Unlock()
		if !q.closed {
			q.closed = true
			for len(q.values) > 0 {
				<-q.values
			}
			close(q.values)
		}
	}
}
func (s *Session) acceptOpenUIEvent(raw json.RawMessage) error {
	q := s.openUIEvents
	if q == nil || len(raw) == 0 || len(raw) > maxOpenUIEventBytes {
		return errors.New("unexpected or oversized OpenUI event")
	}
	if _, err := openui.ValidateEvent(raw, q.instanceID); err != nil {
		return err
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return errors.New("OpenUI event queue closed")
	}
	owned := append(json.RawMessage(nil), raw...)
	select {
	case q.values <- owned:
		return nil
	default:
		return errors.New("OpenUI event queue overflow")
	}
}

// Only trusted application code supplies OpenUI. Render data stays in the
// bounded HTML body; the small protocol header carries just the mode bit.
func prepareOpenUIDocument(d Document) (Document, error) {
	if d.OpenUI == nil {
		return d, nil
	}
	if err := validateDocument(d); err != nil {
		return Document{}, err
	}
	shell, err := openui.Shell(d.HTML, *d.OpenUI, d.Width, d.Appearance)
	if err != nil {
		return Document{}, err
	}
	d.HTML = shell
	d.OpenUI = nil
	d.openUIShell = true
	return d, validateDocument(d)
}
func documentShell(d Document) (string, error) {
	if d.OpenUI != nil {
		prepared, err := prepareOpenUIDocument(d)
		return prepared.HTML, err
	}
	if d.openUIShell {
		return d.HTML, validateDocument(d)
	}
	return htmlpreview.ProcessDocumentShell(d.HTML, d.Width, d.Appearance)
}

func decodeOpenUIBatch(value any) ([]json.RawMessage, error) {
	text, ok := value.(string)
	if !ok || len(text) < 2 || len(text) > maxOpenUIBatchBytes {
		return nil, errors.New("invalid OpenUI event batch")
	}
	var batch []json.RawMessage
	decoder := json.NewDecoder(bytes.NewBufferString(text))
	if err := decoder.Decode(&batch); err != nil {
		return nil, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, errors.New("trailing OpenUI event batch")
	}
	if batch == nil || len(batch) > maxOpenUIEvents {
		return nil, errors.New("invalid OpenUI event count")
	}
	for _, raw := range batch {
		trimmed := bytes.TrimSpace(raw)
		if len(raw) == 0 || len(raw) > maxOpenUIEventBytes || len(trimmed) < 2 || trimmed[0] != '{' {
			return nil, errors.New("invalid OpenUI event payload")
		}
	}
	return batch, nil
}
func sendOpenUIEvent(w io.Writer, nonce string, raw json.RawMessage) error {
	if len(raw) < 1 || len(raw) > maxOpenUIEventBytes || !json.Valid(raw) {
		return errors.New("invalid OpenUI event size or JSON")
	}
	if err := writePacket(w, packet{Version: protocolVersion, Kind: "openui-event", Nonce: nonce, Bytes: len(raw)}); err != nil {
		return err
	}
	return writeAll(w, raw)
}
func (s *Session) receiveOpenUIEvent(r io.Reader, p packet, nonce string) error {
	if err := expect(p, "openui-event", nonce); err != nil {
		return err
	}
	if s.openUIEvents == nil || p.Bytes < 1 || p.Bytes > maxOpenUIEventBytes {
		return errors.New("invalid OpenUI event metadata")
	}
	raw := make(json.RawMessage, p.Bytes)
	if _, err := io.ReadFull(r, raw); err != nil {
		return err
	}
	return s.acceptOpenUIEvent(raw)
}

// Inline macOS responds with up to one bounded event batch followed by a frame.
// Header/body framing remains strict; an event can never masquerade as input.
func (s *Session) receiveOpenUIFrame(r io.Reader, nonce string, width int) ([]byte, error) {
	for count := 0; count <= maxOpenUIEvents; count++ {
		p, err := readPacket(r)
		if err != nil {
			return nil, err
		}
		if p.Kind == "frame" {
			if err := expect(p, "frame", nonce); err != nil {
				return nil, err
			}
			if p.Bytes < 1 || p.Bytes > maxFrameBytes || p.Width != width || p.Height != interactiveHeight {
				return nil, errors.New("invalid preview frame metadata")
			}
			data := make([]byte, p.Bytes)
			if _, err := io.ReadFull(r, data); err != nil {
				return nil, err
			}
			return data, validateFrame(data, width)
		}
		if count == maxOpenUIEvents {
			return nil, errors.New("too many OpenUI events before frame")
		}
		if err := s.receiveOpenUIEvent(r, p, nonce); err != nil {
			return nil, err
		}
	}
	return nil, errors.New("missing OpenUI frame")
}

// Older helpers must fail visibly instead of rendering a viewer whose actions
// can never leave the sandbox. Ordinary HTML remains backward compatible.
func requireOpenUIBridge(doc Document, ready packet) error {
	if doc.OpenUI != nil && !ready.OpenUI {
		return errors.New("preview helper does not support OpenUI events")
	}
	return nil
}
