// Package remote manages the existing dsh-remote machine registry, without a
// second local registry or any new credential storage.
package remote

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"whaleisle/internal/dsh"
	"whaleisle/internal/state"
)

type Machine struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Host            string  `json:"host"`
	Port            int     `json:"port"`
	Username        string  `json:"username"`
	PrivateKeyPath  string  `json:"privateKeyPath"`
	Workspace       string  `json:"workspace"`
	PasswordSet     bool    `json:"passwordSet"`
	LatencyMS       float64 `json:"latencyMs"`
	LastConnectedAt string  `json:"lastConnectedAt"`
	raw             map[string]json.RawMessage
}

func (m *Machine) UnmarshalJSON(b []byte) error {
	type plain Machine
	if err := json.Unmarshal(b, (*plain)(m)); err != nil {
		return err
	}
	return json.Unmarshal(b, &m.raw)
}

type Draft struct {
	ID, Name, Host, Port, Username, PrivateKeyPath, Workspace string
	Options                                                   *ConnectionDraft
}
type Snapshot struct {
	Generation      uint64
	RemovedID       string
	Tools           ToolsSnapshot
	Notice          string
	Status          ConnectionStatus
	Directory       DirectorySnapshot
	Machines        []Machine
	CurrentID       string
	Connected       bool
	Busy            bool
	Loaded          bool
	Error           *state.Fault
	MutationApplied bool
}

// Store keeps the mutable working view under mu and hands readers a
// published, immutable Snapshot version. The attach lease's generation is
// the public Generation; every async RPC body holds an op lease whose
// Renew cancels the previous request, and each directory picker request
// mints a request id from its own lease owner.
type Store struct {
	mu      sync.Mutex
	client  *dsh.Client
	notify  func()
	attach  state.LeaseOwner
	attachL state.Lease
	op      state.LeaseOwner
	opL     state.Lease
	dir     state.LeaseOwner
	dirL    state.Lease
	// directoryRequestID is the public picker token minted from the dir lease;
	// tests may advance it to simulate a later request.
	directoryRequestID uint64
	view               Snapshot
	pub                state.Published[Snapshot]
}

func New() *Store { return &Store{} }
func (s *Store) Attach(c *dsh.Client, notify func()) {
	s.mu.Lock()
	s.op.Retire()
	s.dir.Retire()
	s.client = c
	s.notify = notify
	s.attachL = s.attach.Renew(context.Background())
	s.view = Snapshot{Generation: s.attachL.Generation()}
	s.publishLocked()
	s.mu.Unlock()
}

// Snapshot returns the current published version, shared by all readers
// until the next write; Value is immutable.
func (s *Store) Snapshot() *state.Version[Snapshot] {
	if v := s.pub.Load(); v != nil {
		return v
	}
	s.mu.Lock()
	if s.pub.Load() == nil {
		s.publishLocked()
	}
	s.mu.Unlock()
	return s.pub.Load()
}

// Changed closes when the next version is published.
func (s *Store) Changed() <-chan struct{} { return s.pub.Changed() }

func (s *Store) publishLocked() { s.pub.Publish(s.view) }
func request(ctx context.Context, c *dsh.Client, method, route string, body any, out any) error {
	if c == nil {
		return fmt.Errorf("Harness is not ready")
	}
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.Endpoint.Origin, "/")+"/api/dsh-remote/"+route, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Endpoint.Cookie != "" {
		req.Header.Set("Cookie", c.Endpoint.Cookie)
	}
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return err
	}
	var result struct {
		OK      *bool  `json:"ok"`
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err = json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("dsh-remote %s: invalid JSON (HTTP %d)", route, res.StatusCode)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 || result.OK != nil && !*result.OK {
		detail := result.Error
		if detail == "" {
			detail = result.Message
		}
		if detail == "" {
			detail = http.StatusText(res.StatusCode)
		}
		return fmt.Errorf("dsh-remote %s: %s", route, detail)
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}
func (s *Store) run(action func(context.Context, *dsh.Client) error) {
	s.runMutation(action, nil, nil)
}

// The completion runs under the generation guard after the mutation and refresh.
func (s *Store) runMutation(action func(context.Context, *dsh.Client) error, generation *uint64, applied func(*Snapshot, error, []Machine, bool)) bool {
	s.mu.Lock()
	if s.view.Busy || (generation != nil && *generation != s.view.Generation) {
		s.mu.Unlock()
		return false
	}
	s.view.Busy = true
	s.view.Error = nil
	s.view.Notice = ""
	s.view.MutationApplied = false
	s.view.RemovedID = ""
	c, n, attachL := s.client, s.notify, s.attachL
	parent, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	s.opL = s.op.Renew(parent)
	op := s.opL
	s.publishLocked()
	s.mu.Unlock()
	if n != nil {
		n()
	}
	go func() {
		defer cancel()
		ctx := op.Context()
		var err error
		if action != nil {
			err = action(ctx, c)
		}
		mutationApplied := action != nil && err == nil
		var registry struct {
			Machines  []Machine `json:"machines"`
			CurrentID string    `json:"currentId"`
		}
		var status ConnectionStatus
		registryLoaded := false
		if err == nil {
			err = request(ctx, c, "GET", "machines", nil, &registry)
			registryLoaded = err == nil
		}
		if err == nil {
			err = request(ctx, c, "GET", "status", nil, &status)
		}
		s.mu.Lock()
		if s.attachL != attachL {
			s.mu.Unlock()
			return
		}
		s.view.Busy = false
		s.view.Loaded = true
		s.view.MutationApplied = mutationApplied
		if mutationApplied && err != nil {
			s.view.Directory = DirectorySnapshot{}
		}
		// Registry and connection status are separate reads. Keep the committed
		// selection and saved fields even when status is temporarily unavailable.
		if registryLoaded {
			for _, old := range s.view.Machines {
				if old.ID != s.view.CurrentID {
					continue
				}
				for _, next := range registry.Machines {
					if next.ID == old.ID && (next.Host != old.Host || next.Port != old.Port || next.Username != old.Username) {
						s.view.Directory = DirectorySnapshot{}
					}
				}
			}
			s.view.Machines = registry.Machines
			if s.view.CurrentID != registry.CurrentID {
				s.view.Directory = DirectorySnapshot{}
				s.view.Tools = ToolsSnapshot{}
			}
			s.view.CurrentID = registry.CurrentID
		}
		if err != nil {
			s.view.Error = state.FaultOf(err)
			s.view.Connected = false
			s.view.Status = ConnectionStatus{}
		} else {
			s.view.Status = status
			s.view.Connected = status.Connected
		}
		if mutationApplied && applied != nil {
			applied(&s.view, err, registry.Machines, registryLoaded)
		}
		s.publishLocked()
		s.mu.Unlock()
		if n != nil {
			n()
		}
	}()
	return true
}
func (s *Store) Refresh() { s.run(nil) }
func (s *Store) Select(id string) {
	s.run(func(ctx context.Context, c *dsh.Client) error {
		return request(ctx, c, "POST", "current", map[string]string{"id": id}, nil)
	})
}
func (s *Store) Connect() {
	s.run(func(ctx context.Context, c *dsh.Client) error {
		return request(ctx, c, "POST", "connect", map[string]any{}, nil)
	})
}
func (s *Store) Remove(id string) {
	s.RemoveAtGeneration(id, s.Snapshot().Value.Generation)
}

// Save preserves existing authentication fields unless the editor supplies options.
// Blank passwords keep the existing host-managed secret; no credentials are stored locally.
func (s *Store) Save(d Draft) error {
	body, err := s.payload(d)
	if err != nil {
		return err
	}
	s.run(func(ctx context.Context, c *dsh.Client) error {
		var result struct {
			Warning       string `json:"warning"`
			WarningDetail string `json:"warningDetail"`
		}
		err := request(ctx, c, "POST", "machines", body, &result)
		if err == nil && result.Warning != "" {
			s.mu.Lock()
			if c == s.client {
				s.view.Notice = strings.TrimSpace(result.Warning + " " + result.WarningDetail)
			}
			s.mu.Unlock()
		}
		return err
	})
	return nil
}
func (s *Store) payload(d Draft) (map[string]any, error) {
	var port int
	if d.Port == "" {
		port = 22
	} else {
		if _, err := fmt.Sscan(d.Port, &port); err != nil {
			return nil, fmt.Errorf("Port must be 1–65535")
		}
	}
	// Reject partially parsed integers as well.
	if d.Port != "" && fmt.Sprint(port) != strings.TrimSpace(d.Port) {
		return nil, fmt.Errorf("Port must be 1–65535")
	}
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("Port must be 1–65535")
	}
	if strings.TrimSpace(d.Host) == "" {
		return nil, fmt.Errorf("Host is required")
	}
	username := strings.TrimSpace(d.Username)
	if username == "" {
		username = "root"
	}
	body := map[string]any{"action": "add"}
	if d.ID != "" {
		s.mu.Lock()
		defer s.mu.Unlock()
		found := false
		for _, m := range s.view.Machines {
			if m.ID == d.ID {
				for k, v := range m.raw {
					body[k] = v
				}
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("Machine no longer exists; refresh the list")
		}
		body["action"] = "update"
		body["id"] = d.ID
	}
	body["name"] = strings.TrimSpace(d.Name)
	body["host"] = strings.TrimSpace(d.Host)
	body["port"] = port
	body["username"] = username
	body["privateKeyPath"] = strings.TrimSpace(d.PrivateKeyPath)
	body["workspace"] = strings.TrimSpace(d.Workspace)
	if d.Options != nil {
		if err := d.Options.apply(body); err != nil {
			return nil, err
		}
	}
	return body, nil
}
