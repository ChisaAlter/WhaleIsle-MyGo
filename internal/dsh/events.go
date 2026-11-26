package dsh

import (
	"context"
	"encoding/json"
	"fmt"
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

// eventEnvelope decodes any $events downlink frame in a single json.Unmarshal pass.
type eventEnvelope struct {
	Type     string `json:"type"`
	ClientID string `json:"clientId"`
	Host     struct {
		Home string `json:"home"`
	} `json:"host"`
	Event   string            `json:"event"`
	Args    []json.RawMessage `json:"args"`
	EventID string            `json:"eventId"`
	AgentID string            `json:"agentId"`
	Request json.RawMessage   `json:"request"`
}

func (e eventEnvelope) ready() EventsReady {
	out := EventsReady{Type: e.Type, ClientID: e.ClientID}
	out.Host.Home = e.Host.Home
	return out
}

func (e eventEnvelope) emit() EventsEmit {
	return EventsEmit{Type: e.Type, Event: e.Event, Args: e.Args}
}

func (e eventEnvelope) waterfall() EventsWaterfall {
	return EventsWaterfall{Type: e.Type, Event: e.Event, EventID: e.EventID, AgentID: e.AgentID, Request: e.Request}
}

func (e eventEnvelope) cancel() EventsCancel {
	return EventsCancel{Type: e.Type, EventID: e.EventID}
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
	var activeSeq uint64
	active := make(map[string]map[uint64]context.CancelFunc)
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
		var env eventEnvelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return fmt.Errorf("dsh events: item: %w", err)
		}
		if env.Type == EventWaterfall {
			wf := env.waterfall()
			requestCtx, requestCancel := context.WithCancel(ctx)
			activeMu.Lock()
			activeSeq++
			token := activeSeq
			attempts := active[wf.EventID]
			if attempts == nil {
				attempts = make(map[uint64]context.CancelFunc, 1)
				active[wf.EventID] = attempts
			}
			attempts[token] = requestCancel
			activeMu.Unlock()
			// A prompt must not block later requests or its own cancellation frame.
			tasks.Add(1)
			go func() {
				defer tasks.Done()
				defer func() {
					requestCancel()
					activeMu.Lock()
					if attempts := active[wf.EventID]; attempts != nil {
						delete(attempts, token)
						if len(attempts) == 0 {
							delete(active, wf.EventID)
						}
					}
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
		if env.Type == EventCancel {
			event := env.cancel()
			activeMu.Lock()
			attempts := active[event.EventID]
			cancels := make([]context.CancelFunc, 0, len(attempts))
			for _, cancelRequest := range attempts {
				cancels = append(cancels, cancelRequest)
			}
			activeMu.Unlock()
			for _, cancelRequest := range cancels {
				cancelRequest()
			}
		}
		if err := sess.dispatchEnvelope(ctx, handler, env); err != nil {
			return err
		}
	}
}

func (s *EventsSession) dispatch(ctx context.Context, handler EventsHandler, raw json.RawMessage) error {
	var env eventEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("dsh events: item: %w", err)
	}
	return s.dispatchEnvelope(ctx, handler, env)
}

func (s *EventsSession) dispatchEnvelope(ctx context.Context, handler EventsHandler, env eventEnvelope) error {
	switch env.Type {
	case EventReady:
		ready := env.ready()
		s.mu.Lock()
		s.ClientID = ready.ClientID
		s.Home = ready.Host.Home
		s.mu.Unlock()
		handler.OnReady(ready)
		return nil
	case EventEmit:
		handler.OnEmit(env.emit())
		return nil
	case EventWaterfall:
		return s.answerWaterfall(ctx, handler, env.waterfall())
	case EventCancel:
		handler.OnCancel(env.cancel())
		return nil
	default:
		return fmt.Errorf("dsh events: unknown item type %q", env.Type)
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
	// Forwarded event outcomes use the same Connection RPC envelope as other
	// methods: client-request -> payload.args -> the exact EventResult. Keep
	// delivery authority bound through that authenticated request and receipt.
	return c.Call(ctx, "$events/result", result, nil)
}
