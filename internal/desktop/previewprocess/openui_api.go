package previewprocess

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"time"
	"whaleisle/internal/desktop/openui"
)

// Reuses the event transport budget. Replies are FIFO, never coalesced. This is
// bounded backpressure, not a limit on the number of resource requests.
type openUIAPIQueue struct {
	mu              sync.Mutex
	instanceID      string
	capable, closed bool
	values          chan []byte
	space           chan struct{}
	done            chan struct{}
}

func newOpenUIAPIQueue(instance string) *openUIAPIQueue {
	return &openUIAPIQueue{instanceID: instance, values: make(chan []byte, maxOpenUIEvents), space: make(chan struct{}), done: make(chan struct{})}
}

// ReplyOpenUIAPI admits one bounded response for this exact helper instance.
// The trusted parent shell additionally requires its pending requestId. Call on
// the resource worker, since a full transport queue waits for space or Close.
func (s *Session) ReplyOpenUIAPI(response openui.APIResponse) error {
	if s == nil || s.openUIAPI == nil {
		return errors.New("OpenUI API response channel unavailable")
	}
	q := s.openUIAPI
	if err := openui.ValidateAPIResponse(response, q.instanceID); err != nil {
		return err
	}
	raw, err := json.Marshal(response)
	if err != nil {
		return err
	}
	for {
		q.mu.Lock()
		if q.closed {
			q.mu.Unlock()
			return errors.New("OpenUI API response channel closed")
		}
		if !q.capable {
			q.mu.Unlock()
			return errors.New("preview helper does not support OpenUI API responses")
		}
		select {
		case q.values <- raw:
			q.mu.Unlock()
			return nil
		default:
		}
		space := q.space
		q.mu.Unlock()
		select {
		case <-q.done:
			return errors.New("OpenUI API response channel closed")
		case <-space:
		}
	}
}
func (s *Session) enableOpenUIAPI(capable bool) {
	if q := s.openUIAPI; q != nil {
		q.mu.Lock()
		q.capable = capable
		q.mu.Unlock()
	}
}
func (s *Session) openUIAPIResponses() <-chan []byte {
	if s.openUIAPI == nil {
		return nil
	}
	return s.openUIAPI.values
}
func (s *Session) receivedOpenUIAPIResponse() bool {
	q := s.openUIAPI
	if q == nil {
		return false
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	close(q.space)
	q.space = make(chan struct{})
	return !q.closed
}
func (s *Session) closeOpenUIAPI() {
	if q := s.openUIAPI; q != nil {
		q.mu.Lock()
		defer q.mu.Unlock()
		if !q.closed {
			q.closed = true
			close(q.done)
			for {
				select {
				case <-q.values:
				default:
					return
				}
			}
		}
	}
}
func startOpenUIAPIPump(ctx context.Context, s *Session, send func([]byte) error) {
	if s.openUIAPI == nil {
		return
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.Done():
				return
			case <-s.openUIAPI.done:
				return
			case raw := <-s.openUIAPIResponses():
				if !s.receivedOpenUIAPIResponse() {
					return
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
func validateOpenUIAPIResponse(raw []byte) (openui.APIResponse, error) {
	var response openui.APIResponse
	if len(raw) < 1 || len(raw) > maxOpenUIEventBytes {
		return response, errors.New("invalid OpenUI API response length")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&response); err != nil {
		return response, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return response, errors.New("trailing OpenUI API response")
	}
	if err := openui.ValidateRender(openui.Render{InstanceID: response.InstanceID, Program: "response", State: json.RawMessage(`{}`)}); err != nil {
		return response, err
	}
	return response, openui.ValidateAPIResponse(response, response.InstanceID)
}
func sendOpenUIAPIResponse(w io.Writer, nonce string, raw []byte) error {
	if _, err := validateOpenUIAPIResponse(raw); err != nil {
		return err
	}
	if err := writePacket(w, packet{Version: protocolVersion, Kind: "api-response", Nonce: nonce, Bytes: len(raw)}); err != nil {
		return err
	}
	return writeAll(w, raw)
}
func receiveOpenUIAPIResponse(r io.Reader, p packet, nonce string) ([]byte, error) {
	if err := expect(p, "api-response", nonce); err != nil {
		return nil, err
	}
	if p.Bytes < 1 || p.Bytes > maxOpenUIEventBytes {
		return nil, errors.New("invalid OpenUI API response byte count")
	}
	raw := make([]byte, p.Bytes)
	if _, err := io.ReadFull(r, raw); err != nil {
		return nil, err
	}
	response, err := validateOpenUIAPIResponse(raw)
	if err != nil {
		return nil, err
	}
	return json.Marshal(response)
}
func openUIAPIResponseScript(raw []byte) (string, error) {
	response, err := validateOpenUIAPIResponse(raw)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		return "", err
	}
	return "(()=>{if(window.__whaleOpenUIAPIResponse(" + string(encoded) + ")!==true)throw new Error('OpenUI API response rejected');return true;})()", nil
}
