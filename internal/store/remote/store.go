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
)

type Machine struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Username       string `json:"username"`
	PrivateKeyPath string `json:"privateKeyPath"`
	Workspace      string `json:"workspace"`
	raw            map[string]json.RawMessage
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
	Tools           ToolsSnapshot
	Notice          string
	Status          ConnectionStatus
	Directory       DirectorySnapshot
	Machines        []Machine
	CurrentID       string
	Connected       bool
	Busy            bool
	Loaded          bool
	Error           string
	MutationApplied bool
}
type Store struct {
	mu         sync.Mutex
	client     *dsh.Client
	notify     func()
	generation uint64
	view       Snapshot
	cancel     context.CancelFunc
}

func New() *Store { return &Store{} }
func (s *Store) Attach(c *dsh.Client, notify func()) {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.client = c
	s.notify = notify
	s.generation++
	s.view = Snapshot{}
	s.mu.Unlock()
}
func (s *Store) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.view
	v.Machines = append([]Machine(nil), v.Machines...)
	v.Directory.Items = append([]DirectoryEntry(nil), v.Directory.Items...)
	v.Tools = cloneTools(v.Tools)
	return v
}
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
	s.mu.Lock()
	if s.view.Busy {
		s.mu.Unlock()
		return
	}
	s.view.Busy = true
	s.view.Error = ""
	s.view.Notice = ""
	s.view.MutationApplied = false
	c, n, g := s.client, s.notify, s.generation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	s.cancel = cancel
	s.mu.Unlock()
	if n != nil {
		n()
	}
	go func() {
		defer cancel()
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
		if err == nil {
			err = request(ctx, c, "GET", "machines", nil, &registry)
		}
		if err == nil {
			err = request(ctx, c, "GET", "status", nil, &status)
		}
		s.mu.Lock()
		if g != s.generation {
			s.mu.Unlock()
			return
		}
		s.view.Busy = false
		s.view.Loaded = true
		s.view.MutationApplied = mutationApplied
		if mutationApplied && err != nil {
			s.view.Directory = DirectorySnapshot{}
		}
		if err != nil {
			s.view.Error = err.Error()
			s.view.Connected = false
		} else {
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
			s.view.Status = status
			s.view.CurrentID = registry.CurrentID
			s.view.Connected = status.Connected

		}
		s.mu.Unlock()
		if n != nil {
			n()
		}
	}()
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
	s.run(func(ctx context.Context, c *dsh.Client) error {
		return request(ctx, c, "POST", "machines", map[string]string{"action": "delete", "id": id}, nil)
	})
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
	if strings.TrimSpace(d.Host) == "" || strings.TrimSpace(d.Username) == "" {
		return nil, fmt.Errorf("Host and username are required")
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
	body["username"] = strings.TrimSpace(d.Username)
	body["privateKeyPath"] = strings.TrimSpace(d.PrivateKeyPath)
	body["workspace"] = strings.TrimSpace(d.Workspace)
	if d.Options != nil {
		if err := d.Options.apply(body); err != nil {
			return nil, err
		}
	}
	return body, nil
}
