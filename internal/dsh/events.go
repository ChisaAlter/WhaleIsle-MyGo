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
	Result    func(EventResult, error) bool
	// Retire withdraws this logical delivery generation, preserving durable
	// request drafts. It is distinct from a real Host EventCancel.
	Retire func()
}

// OnRetire reports whether this handler owns generation cleanup. Legacy
// handlers without it keep per-event cancellation cleanup on stream exit.
func (f EventsFuncs) OnRetire() bool {
	if f.Retire == nil {
		return false
	}
	f.Retire()
	return true
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

// OnResult reports delivery acknowledgement. True requests a user-driven retry.
func (f EventsFuncs) OnResult(result EventResult, err error) bool {
	if f.Result != nil {
		return f.Result(result, err)
	}
	return false
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

	ctx, cancel := context.WithCancel(ctx)
	sess := &EventsSession{client: c, mux: mux, stream: stream}
	failures := make(chan error, 1)
	var activeMu sync.Mutex
	active := make(map[string]context.CancelFunc)
	var tasks sync.WaitGroup
	defer func() {
		cancel()
		stream.Cancel()
		retired := false
		if handler, ok := handler.(interface{ OnRetire() bool }); ok {
			retired = handler.OnRetire()
		}
		if !retired {
			activeMu.Lock()
			ids := make([]string, 0, len(active))
			for id := range active {
				ids = append(ids, id)
			}
			activeMu.Unlock()
			for _, id := range ids {
				handler.OnCancel(EventsCancel{Type: EventCancel, EventID: id})
			}
		}
		// An owned Retire releases its waiters before joining this generation.
		// Legacy handlers expose no context-aware join contract; preserve their
		// existing cancellation callback rather than waiting on arbitrary work.
		if retired {
			tasks.Wait()
		}
	}()
	for {
		raw, err := stream.Recv(ctx)
		if err != nil {
			select {
			case failure := <-failures:
				return failure
			default:
			}
			if err == ErrStreamEnd {
				return nil
			}
			return err
		}
		var head struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(raw, &head) == nil && head.Type == EventWaterfall {
			var wf EventsWaterfall
			if err := json.Unmarshal(raw, &wf); err != nil {
				return err
			}
			requestCtx, requestCancel := context.WithCancel(ctx)
			activeMu.Lock()
			active[wf.EventID] = requestCancel
			activeMu.Unlock()
			// A prompt must not block later requests or its own cancellation frame.
			tasks.Add(1)
			go func() {
				defer tasks.Done()
				defer func() {
					requestCancel()
					activeMu.Lock()
					delete(active, wf.EventID)
					activeMu.Unlock()
				}()
				if err := sess.answerWaterfall(requestCtx, handler, wf); err != nil && requestCtx.Err() == nil {
					select {
					case failures <- err:
					default:
					}
					stream.Cancel()
				}
			}()
			continue
		}
		if head.Type == EventCancel {
			var event EventsCancel
			if json.Unmarshal(raw, &event) == nil {
				activeMu.Lock()
				cancelRequest := active[event.EventID]
				activeMu.Unlock()
				if cancelRequest != nil {
					cancelRequest()
				}
			}
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
	for {
		outcome, err := handler.OnWaterfall(wf)
		result := EventResult{ClientID: clientID, EventID: wf.EventID}
		if err != nil {
			result.Outcome = EventResultOutcome{Kind: "rejected", Error: &EventRejection{Name: "Error", Message: err.Error()}}
		} else if outcome == nil {
			result = EventResultNext(clientID, wf.EventID)
		} else {
			result.Outcome = *outcome
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err = s.client.PostEventResult(ctx, result)
		retry := false
		if observer, ok := handler.(interface{ OnResult(EventResult, error) bool }); ok {
			retry = observer.OnResult(result, err)
		}
		if err == nil || !retry {
			return err
		}
	}

}

// PostEventResult answers one Host waterfall via HTTP RPC.
func (c *Client) PostEventResult(ctx context.Context, result EventResult) error {
	ctx, release, err := result.Outcome.Delivery.bind(ctx, c, result.EventID)
	if err != nil {
		return err
	}
	defer release()
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
