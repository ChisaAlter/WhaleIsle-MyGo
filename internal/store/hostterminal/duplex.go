package hostterminal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type inputCommand struct {
	Type     string `json:"type"`
	Sequence int64  `json:"sequence"`
	Data     string `json:"data,omitempty"`
	Cols     int    `json:"cols,omitempty"`
	Rows     int    `json:"rows,omitempty"`
}

type controlRefusal struct{ reason string }

func (e *controlRefusal) Error() string { return "Host terminal input refused: " + e.reason }

// Write succeeds after the Host provider acknowledges one ordered command on
// this physical stream. A completed WebSocket write alone is not acceptance.
// An uncertain outcome fails this attachment and is never replayed.
func (a *Attachment) Write(p []byte) (int, error) {
	a.ioMu.Lock()
	defer a.ioMu.Unlock()
	s := a.store
	s.mu.Lock()
	valid := s.writableLocked(a)
	maxBytes := s.sessionLocked(a.entry.key.session).environment.MaxInputBytes
	s.mu.Unlock()
	if !valid {
		return 0, errors.New("Host terminal attachment is not writable")
	}
	if len(p) > maxBytes {
		a.fail(errors.New("Host terminal input exceeds its byte limit"))
		return 0, errors.New("Host terminal input exceeds its byte limit")
	}
	if len(p) == 0 {
		return 0, nil
	}
	if err := a.sendCommand(inputCommand{Type: "write", Data: string(p)}); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Resize uses the same ordered, provider-acknowledged uplink as Write. The
// authoritative Host state, rather than the acknowledgement, updates its grid.
func (a *Attachment) Resize(cols, rows int) error {
	a.ioMu.Lock()
	defer a.ioMu.Unlock()
	s := a.store
	s.mu.Lock()
	if !s.writableLocked(a) {
		s.mu.Unlock()
		return errors.New("Host terminal attachment is not writable")
	}
	env := s.sessionLocked(a.entry.key.session).environment
	cols, rows = min(cols, env.MaxCols), min(rows, env.MaxRows)
	same := a.entry.info.Cols == cols && a.entry.info.Rows == rows
	s.mu.Unlock()
	if cols < 2 || rows < 1 || same {
		return nil
	}
	return a.sendCommand(inputCommand{Type: "resize", Cols: cols, Rows: rows})
}

// ioMu permits one in-flight command, so no queued input or input-window credit
// can escape this attachment's lifetime.
func (a *Attachment) sendCommand(command inputCommand) (err error) {
	s := a.store
	s.mu.Lock()
	if !s.writableLocked(a) || a.stream == nil {
		s.mu.Unlock()
		return errors.New("Host terminal attachment is not writable")
	}
	if a.inputSequence >= 9007199254740991 {
		s.mu.Unlock()
		err = errors.New("Host terminal input sequence exhausted")
		a.fail(err)
		return err
	}
	command.Sequence = a.inputSequence + 1
	encoded, err := json.Marshal(command)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	env := s.sessionLocked(a.entry.key.session).environment
	bytes := int64(len(encoded))
	tooLarge := env.MaxBufferedBytes > 0 && bytes > int64(env.MaxBufferedBytes)
	if window := a.inputWindow; window != nil {
		tooLarge = tooLarge || bytes > window.MaxBytes-window.FrameOverheadBytes
		if env.MaxBufferedBytes > 0 {
			tooLarge = tooLarge || window.FrameOverheadBytes > int64(env.MaxBufferedBytes)-bytes
		}
	}
	if tooLarge {
		s.mu.Unlock()
		err = errors.New("Host terminal command exceeds its encoded input window")
		a.fail(err)
		return err
	}
	a.inputSequence, a.pendingInput = command.Sequence, command.Sequence
	stream := a.stream
	s.mu.Unlock()
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	err = stream.SendItem(ctx, command)
	if err == nil {
		select {
		case <-ctx.Done():
			err = ctx.Err()
		case ack := <-a.inputAck:
			if ack.Sequence != command.Sequence {
				err = errors.New("Host terminal input acknowledgement is out of sequence")
			} else if ack.Refusal != "" {
				return &controlRefusal{reason: ack.Refusal}
			}
		}
	}
	if err != nil {
		err = fmt.Errorf("Host terminal %s was not confirmed; it will not be replayed: %w", command.Type, err)
		a.fail(err)
	}
	return err
}
