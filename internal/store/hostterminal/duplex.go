package hostterminal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type inputWindow struct {
	MaxBytes           int `json:"maxBytes"`
	FrameOverheadBytes int `json:"frameOverheadBytes"`
}
type inputAcceptance struct {
	sequence int64
	done     chan error
}

// command waits for provider acceptance on the same physical duplex stream.
// One pending command bounds the input queue and preserves its ordered sequence;
// an uncertain write is never replayed after reconnect or attachment takeover.
// The caller holds ioMu, while follow consumes acknowledgements independently.
func (a *Attachment) command(value map[string]any) error {
	s := a.store
	s.mu.Lock()
	if !s.writableLocked(a) || a.stream == nil {
		s.mu.Unlock()
		return errors.New("Host terminal attachment changed")
	}
	if a.inputSequence >= 9007199254740991 {
		s.mu.Unlock()
		return errors.New("Host terminal input sequence exhausted")
	}
	value["sequence"] = a.inputSequence + 1
	raw, err := json.Marshal(value)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	env := s.sessionLocked(a.entry.key.session).environment
	encoded := len(raw)
	if w := a.inputWindow; w != nil {
		encoded += w.FrameOverheadBytes + 1
		if encoded > w.MaxBytes {
			s.mu.Unlock()
			return errors.New("Host terminal command exceeds its carrier input window")
		}
	}
	if env.MaxBufferedBytes > 0 && encoded > env.MaxBufferedBytes {
		s.mu.Unlock()
		return errors.New("Host terminal command exceeds its encoded input limit")
	}
	a.inputSequence++
	pending := &inputAcceptance{sequence: a.inputSequence, done: make(chan error, 1)}
	a.pending = pending
	stream := a.stream
	s.mu.Unlock()
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	if err = stream.SendItem(ctx, value); err != nil {
		return err
	}
	select {
	case err = <-pending.done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// acceptLocked validates a strict per-attachment acknowledgement; output event
// sequence numbers belong to a different stream and must never advance input.
func (a *Attachment) acceptLocked(frame wireFrame) error {
	p := a.pending
	if p == nil || p.sequence != *frame.Sequence {
		return errors.New("unexpected Host terminal input acknowledgement")
	}
	a.pending = nil
	var err error
	if frame.Refusal != "" {
		a.writable = false
		err = fmt.Errorf("Host terminal input refused: %s", frame.Refusal)
	}
	p.done <- err
	return nil
}
