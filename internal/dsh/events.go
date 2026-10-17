package dsh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
)

// EventsHandler receives $events downlink frames for one Client generation.
// Waterfall handlers that return (nil, nil) cause an automatic "next" reply.
type EventsHandler interface {
	OnReady(ready EventsReady)
	OnEmit(emit EventsEmit)
	OnWaterfall(wf EventsWaterfall) (outcome *EventResultOutcome, err error)
	OnCancel(cancel EventsCancel)
}

// EventsFuncs adapts functions to EventsHandler; nil funcs are no-ops / next.
type EventsFuncs struct {
	Ready     func(EventsReady)
	Emit      func(EventsEmit)
	Waterfall func(EventsWaterfall) (*EventResultOutcome, error)
	Cancel    func(EventsCancel)
}

func (f EventsFuncs) OnReady(ready EventsReady) {
	if f.Ready != nil {
		f.Ready(ready)
	}
}

func (f EventsFuncs) OnEmit(emit EventsEmit) {
	if f.Emit != nil {
		f.Emit(emit)
	}
}

func (f EventsFuncs) OnWaterfall(wf EventsWaterfall) (*EventResultOutcome, error) {
	if f.Waterfall != nil {
		return f.Waterfall(wf)
	}
	return nil, nil
}

func (f EventsFuncs) OnCancel(cancel EventsCancel) {
	if f.Cancel != nil {
		f.Cancel(cancel)
	}
}

// EventsSession owns one $events logical stream generation.
type EventsSession struct {
	ClientID string
	Home     string

	client *Client
	mux    *Mux
	stream *Stream
	mu     sync.Mutex
}

// FollowEvents dials the mux if needed, opens $events, and dispatches until ctx
// ends or the stream ends. Unhandled waterfalls are answered with outcome next.
func (c *Client) FollowEvents(ctx context.Context, mux *Mux, handler EventsHandler) error {
	if mux == nil {
		return fmt.Errorf("dsh events: nil mux")
	}
	if handler == nil {
		handler = EventsFuncs{}
	}
	if err := mux.Dial(ctx); err != nil {
		return err
	}
	stream, err := mux.Open(ctx, EventsEndpoint, EventsStreamPayload)
	if err != nil {
		return err
	}
	defer stream.Cancel()

	sess := &EventsSession{client: c, mux: mux, stream: stream}
	for {
		raw, err := stream.Recv(ctx)
		if err != nil {
			if err == ErrStreamEnd {
				return nil
			}
			return err
		}
		if err := sess.dispatch(ctx, handler, raw); err != nil {
			return err
		}
	}
}

func (s *EventsSession) dispatch(ctx context.Context, handler EventsHandler, raw json.RawMessage) error {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &head); err != nil {
		return fmt.Errorf("dsh events: item: %w", err)
	}
	switch head.Type {
	case EventReady:
		var ready EventsReady
		if err := json.Unmarshal(raw, &ready); err != nil {
			return err
		}
		s.mu.Lock()
		s.ClientID = ready.ClientID
		s.Home = ready.Host.Home
		s.mu.Unlock()
		handler.OnReady(ready)
		return nil
	case EventEmit:
		var emit EventsEmit
		if err := json.Unmarshal(raw, &emit); err != nil {
			return err
		}
		handler.OnEmit(emit)
		return nil
	case EventWaterfall:
		var wf EventsWaterfall
		if err := json.Unmarshal(raw, &wf); err != nil {
			return err
		}
		return s.answerWaterfall(ctx, handler, wf)
	case EventCancel:
		var cancel EventsCancel
		if err := json.Unmarshal(raw, &cancel); err != nil {
			return err
		}
		handler.OnCancel(cancel)
		return nil
	default:
		return fmt.Errorf("dsh events: unknown item type %q", head.Type)
	}
}

func (s *EventsSession) answerWaterfall(ctx context.Context, handler EventsHandler, wf EventsWaterfall) error {
	s.mu.Lock()
	clientID := s.ClientID
	s.mu.Unlock()
	if clientID == "" {
		return fmt.Errorf("dsh events: waterfall before ready")
	}
	outcome, err := handler.OnWaterfall(wf)
	if err != nil {
		rej := &EventRejection{Name: "Error", Message: err.Error()}
		return s.client.PostEventResult(ctx, EventResult{
			ClientID: clientID,
			EventID:  wf.EventID,
			Outcome:  EventResultOutcome{Kind: "rejected", Error: rej},
		})
	}
	if outcome == nil {
		return s.client.PostEventResult(ctx, EventResultNext(clientID, wf.EventID))
	}
	return s.client.PostEventResult(ctx, EventResult{
		ClientID: clientID,
		EventID:  wf.EventID,
		Outcome:  *outcome,
	})
}

// PostEventResult answers one Host waterfall via HTTP RPC.
func (c *Client) PostEventResult(ctx context.Context, result EventResult) error {
	if result.ClientID == "" || result.EventID == "" || result.Outcome.Kind == "" {
		return fmt.Errorf("dsh events: incomplete result")
	}
	body, err := json.Marshal(result)
	if err != nil {
		return err
	}
	endpoint, err := url.JoinPath(c.Endpoint.Origin, EventsResultPath)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Endpoint.Cookie != "" {
		req.Header.Set("Cookie", c.Endpoint.Cookie)
	}
	res, err := c.http().Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("dsh events/result: HTTP %d: %s", res.StatusCode, clip(raw, 200))
	}
	// Gateway may return an empty body or a Typert envelope; accept both.
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	var parsed struct {
		Type   string `json:"type"`
		Result struct {
			OK    bool `json:"ok"`
			Error *struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil
	}
	if parsed.Type == "server-response" && !parsed.Result.OK {
		msg := "events/result failed"
		code := ""
		if parsed.Result.Error != nil {
			msg = parsed.Result.Error.Message
			code = parsed.Result.Error.Code
		}
		return &Error{Code: code, Message: msg}
	}
	return nil
}
