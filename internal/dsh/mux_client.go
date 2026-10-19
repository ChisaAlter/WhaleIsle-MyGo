package dsh

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// ErrStreamEnd means the Host half-closed the logical stream cleanly.
var ErrStreamEnd = errors.New("dsh mux: stream end")

// ErrCarrierLost means the physical mux socket died.
var ErrCarrierLost = errors.New("dsh mux: carrier lost")

// Mux is one physical WebSocket that carries many logical Remote streams.
// WebSocket Ping/Pong is handled by the underlying library (server pings every 2s).
type Mux struct {
	Endpoint Endpoint
	HTTP     *http.Client

	// DialTimeout caps the WebSocket handshake. Zero uses 15s.
	DialTimeout time.Duration
	// OnCarrierLost is invoked once when the physical socket dies (optional).
	OnCarrierLost    func(error)
	maintainMu       sync.Mutex
	maintainer       *muxMaintainer
	networkKnown     bool // guarded by maintainMu; unknown never means offline
	networkAvailable bool
	networkRevision  uint64

	mu      sync.Mutex
	conn    *websocket.Conn
	streams map[string]*logicalStream
	writeMu sync.Mutex
	closed  bool
	done    chan struct{}
	readErr error
}

type logicalStream struct {
	inbox chan ServerFrame
}

// Stream is one open logical Remote stream on a Mux.
type Stream struct {
	mux      *Mux
	conn     *websocket.Conn
	done     <-chan struct{}
	id       string
	inbox    <-chan ServerFrame
	cancel   context.CancelFunc
	opened   bool
	terminal bool
}

// Dial connects the physical mux socket. Cookie is sent; Origin is omitted.
func (m *Mux) Dial(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return fmt.Errorf("dsh mux: closed")
	}
	if m.conn != nil {
		return nil
	}
	timeout := m.DialTimeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	u, err := muxURL(m.Endpoint.Origin)
	if err != nil {
		return err
	}
	header := http.Header{}
	if m.Endpoint.Cookie != "" {
		header.Set("Cookie", m.Endpoint.Cookie)
	}
	opts := &websocket.DialOptions{
		HTTPClient: m.http(),
		HTTPHeader: header,
	}
	conn, _, err := websocket.Dial(dialCtx, u, opts)
	if err != nil {
		return fmt.Errorf("dsh mux: dial: %w", err)
	}
	conn.SetReadLimit(8 << 20)
	m.conn = conn
	m.streams = make(map[string]*logicalStream)
	m.done = make(chan struct{})
	m.readErr = nil
	go m.readLoop(conn, m.done)
	return nil
}

// Reset clears a Close() so Dial can be used again (reconnect loops).
func (m *Mux) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = false
	m.conn = nil
	m.readErr = nil
	m.streams = nil
	m.done = nil
}

// Close tears down the physical socket and fails every open stream.
func (m *Mux) Close() error {
	return m.closeCarrier(false)
}

func (m *Mux) closeCarrier(immediate bool) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil
	}
	m.closed = true
	conn := m.conn
	m.conn = nil
	m.streams = make(map[string]*logicalStream)
	done := m.done
	m.mu.Unlock()
	if conn != nil {
		if immediate {
			_ = conn.CloseNow()
		} else {
			_ = conn.Close(websocket.StatusNormalClosure, "closed")
		}
	}
	if done != nil {
		<-done
	}
	return nil
}

// Open starts one logical stream. Drain Recv until ErrStreamEnd or error;
// Cancel aborts early.
func (m *Mux) Open(ctx context.Context, endpoint string, payload any) (*Stream, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("dsh mux: empty endpoint")
	}
	raw, err := marshalPayload(payload)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	if m.closed || m.conn == nil {
		m.mu.Unlock()
		return nil, fmt.Errorf("dsh mux: not connected")
	}
	id := newID()
	ls := &logicalStream{inbox: make(chan ServerFrame, 64)}
	m.streams[id] = ls
	conn := m.conn
	done := m.done
	m.mu.Unlock()

	frame := ClientFrame{
		Type:     FrameOpen,
		StreamID: id,
		Endpoint: endpoint,
		Payload:  raw,
	}
	if err := m.send(ctx, conn, frame); err != nil {
		m.removeStream(id)
		return nil, err
	}
	streamCtx, cancel := context.WithCancel(ctx)
	s := &Stream{
		mux:    m,
		conn:   conn,
		done:   done,
		id:     id,
		inbox:  ls.inbox,
		cancel: cancel,
		opened: true,
	}
	go func() {
		select {
		case <-streamCtx.Done():
			s.sendCancel()
		case <-done:
		}
	}()
	return s, nil
}

// Recv returns the next downlink value, ErrStreamEnd, or a stream/carrier error.
func (s *Stream) Recv(ctx context.Context) (json.RawMessage, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		select {
		case <-s.done:
			return nil, s.carrierFailure()
		default:
		}
		select {
		case frame := <-s.inbox:
			return s.handleFrame(frame)
		default:
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case frame := <-s.inbox:
			return s.handleFrame(frame)
		case <-s.done:
			return nil, s.carrierFailure()
		}
	}
}

func (s *Stream) carrierFailure() error {
	s.terminal = true
	s.mux.mu.Lock()
	var err error
	if s.mux.done == s.done {
		err = s.mux.readErr
	}
	s.mux.mu.Unlock()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCarrierLost, err)
	}
	return ErrCarrierLost
}

func (s *Stream) handleFrame(frame ServerFrame) (json.RawMessage, error) {
	switch frame.Type {
	case FrameItem:
		return frame.Value, nil
	case FrameEnd:
		s.terminal = true
		s.mux.removeStream(s.id)
		return nil, ErrStreamEnd
	case FrameError:
		s.terminal = true
		s.mux.removeStream(s.id)
		if frame.Error != nil {
			return nil, frame.Error
		}
		return nil, fmt.Errorf("dsh mux: stream error")
	default:
		return nil, fmt.Errorf("dsh mux: unexpected frame %q", frame.Type)
	}
}

// SendItem pushes one uplink item.
func (s *Stream) SendItem(ctx context.Context, value any) error {
	raw, err := marshalPayload(value)
	if err != nil {
		return err
	}
	return s.send(ctx, ClientFrame{Type: FrameItem, StreamID: s.id, Value: raw})
}

// End half-closes the uplink.
func (s *Stream) End(ctx context.Context) error {
	return s.send(ctx, ClientFrame{Type: FrameEnd, StreamID: s.id})
}

// Cancel aborts the logical stream on the Host.
func (s *Stream) Cancel() {
	s.cancel()
}

func (s *Stream) sendCancel() {
	if !s.opened || s.terminal {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.send(ctx, ClientFrame{Type: FrameCancel, StreamID: s.id})
	s.mux.removeStream(s.id)
}

func (s *Stream) send(ctx context.Context, frame ClientFrame) error {
	s.mux.mu.Lock()
	current := !s.mux.closed && s.mux.conn == s.conn && s.mux.done == s.done
	s.mux.mu.Unlock()
	if !current {
		return ErrCarrierLost
	}
	return s.mux.send(ctx, s.conn, frame)
}

func (m *Mux) http() *http.Client {
	if m.HTTP != nil {
		return m.HTTP
	}
	return http.DefaultClient
}

func (m *Mux) readLoop(conn *websocket.Conn, done chan struct{}) {
	if done != nil {
		defer close(done)
	}
	if conn == nil {
		return
	}
	for {
		var frame ServerFrame
		err := wsjson.Read(context.Background(), conn, &frame)
		if err != nil {
			m.carrierLost(conn, done, err)
			return
		}
		if err := frame.Validate(); err != nil {
			m.carrierLost(conn, done, err)
			_ = conn.CloseNow()
			return
		}
		m.mu.Lock()
		if m.closed || m.conn != conn || m.done != done {
			m.mu.Unlock()
			return
		}
		ls := m.streams[frame.StreamID]
		m.mu.Unlock()
		if ls == nil {
			continue
		}
		select {
		case ls.inbox <- frame:
		default:
			// Inbox full: treat as carrier failure for this connection.
			m.carrierLost(conn, done, fmt.Errorf("dsh mux: stream %s inbox full", frame.StreamID))
			_ = conn.CloseNow()
			return
		}
	}
}

// Only the socket owning this read loop may retire its carrier or notify the
// maintainer. A late failure from an old loop cannot change a newer Dial.
func (m *Mux) carrierLost(conn *websocket.Conn, done chan struct{}, err error) {
	m.mu.Lock()
	if m.closed || m.conn != conn || m.done != done {
		m.mu.Unlock()
		return
	}
	m.readErr = err
	m.conn = nil
	m.streams = make(map[string]*logicalStream)
	cb := m.OnCarrierLost
	m.mu.Unlock()
	if cb != nil {
		cb(err)
	}
}

func (m *Mux) removeStream(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.streams, id)
}

func (m *Mux) sendTo(ctx context.Context, frame ClientFrame) error {
	m.mu.Lock()
	conn := m.conn
	closed := m.closed
	m.mu.Unlock()
	if closed || conn == nil {
		return fmt.Errorf("dsh mux: not connected")
	}
	return m.send(ctx, conn, frame)
}

func (m *Mux) send(ctx context.Context, conn *websocket.Conn, frame ClientFrame) error {
	if err := frame.Validate(); err != nil {
		return err
	}
	m.writeMu.Lock()
	defer m.writeMu.Unlock()
	return wsjson.Write(ctx, conn, frame)
}

func muxURL(origin string) (string, error) {
	u, err := url.Parse(origin)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
		// ok
	default:
		return "", fmt.Errorf("dsh mux: unsupported origin scheme %q", u.Scheme)
	}
	u.Path = MuxPath
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func marshalPayload(payload any) (json.RawMessage, error) {
	if payload == nil {
		return json.RawMessage(`{}`), nil
	}
	switch v := payload.(type) {
	case json.RawMessage:
		if len(v) == 0 {
			return json.RawMessage(`{}`), nil
		}
		return v, nil
	case []byte:
		if len(v) == 0 {
			return json.RawMessage(`{}`), nil
		}
		return json.RawMessage(v), nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		return b, nil
	}
}
