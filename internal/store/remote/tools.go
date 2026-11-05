package remote

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type ConnectionStatus struct {
	Connected    bool   `json:"connected"`
	HostKeyKnown bool   `json:"hostKeyKnown"`
	Workspace    string `json:"workspace"`
	LocalMirror  string `json:"localMirror"`
	Host         string `json:"host"`
	Username     string `json:"username"`
	Port         int    `json:"port"`
}

type Forward struct {
	ID         string `json:"id"`
	Direction  string `json:"direction"`
	ListenPort int    `json:"listenPort"`
	TargetHost string `json:"targetHost"`
	TargetPort int    `json:"targetPort"`
	MachineID  string `json:"machineId"`
	AutoStart  bool   `json:"autoStart"`
	Active     bool   `json:"active"`
}

type ForwardDraft struct {
	Direction, ListenPort, TargetHost, TargetPort string
	AutoStart                                     bool
}

type ToolsSnapshot struct {
	Loaded                    bool
	Error, Notice             string
	Forwards                  []Forward
	SSHEntries                []SSHConfigEntry
	AuditLoaded, AuditEnabled bool
	AuditLines                []string
}

func cloneTools(v ToolsSnapshot) ToolsSnapshot {
	v.Forwards = append([]Forward(nil), v.Forwards...)
	v.SSHEntries = append([]SSHConfigEntry(nil), v.SSHEntries...)
	v.AuditLines = append([]string(nil), v.AuditLines...)
	return v
}

func (s *Store) LoadTools()     { s.toolsRequest("load", nil, "") }
func (s *Store) LoadAudit()     { s.toolsRequest("audit", nil, "") }
func (s *Store) ForgetHostKey() { s.toolsRequest("forget-key", map[string]any{}, "") }
func (s *Store) TestMachine(id string) {
	s.mu.Lock()
	found := false
	for _, m := range s.view.Machines {
		if m.ID == id {
			found = true
			break
		}
	}
	s.mu.Unlock()
	if found {
		s.toolsRequest("test-connect", map[string]any{"machineId": id}, id)
	}
}
func (s *Store) TestDraft(d Draft) error {
	payload, err := s.payload(d)
	if err != nil {
		return err
	}
	delete(payload, "action")
	delete(payload, "id")
	if d.ID != "" {
		payload["machineId"] = d.ID
	}
	s.toolsRequest("test-connect", payload, d.ID)
	return nil
}
func (s *Store) DefineForward(d ForwardDraft) error {
	listen, err := parsePort(d.ListenPort, 0)
	if err != nil || listen == 0 {
		return fmt.Errorf("Listen port must be 1–65535")
	}
	target, err := parsePort(d.TargetPort, listen)
	if err != nil {
		return err
	}
	direction := d.Direction
	if direction == "" {
		direction = "local"
	}
	if direction != "local" && direction != "reverse" {
		return fmt.Errorf("Invalid forwarding direction")
	}
	host := strings.TrimSpace(d.TargetHost)
	if host == "" {
		host = "127.0.0.1"
	}
	s.toolsRequest("forwards", map[string]any{
		"action": "define", "direction": direction, "listenPort": listen,
		"targetHost": host, "targetPort": target, "autoStart": d.AutoStart,
	}, "")
	return nil
}
func (s *Store) ForwardAction(id, action string) {
	if action != "start" && action != "stop" && action != "remove" {
		return
	}
	s.mu.Lock()
	found := false
	for _, f := range s.view.Tools.Forwards {
		if f.ID == id && (action != "start" || f.MachineID == "" || f.MachineID == s.view.CurrentID) {
			found = true
			break
		}
	}
	s.mu.Unlock()
	if found {
		s.toolsRequest("forwards", map[string]any{"action": action, "id": id}, "")
	}
}

// All settings operations share the registry busy gate. A selection/reconnect
// cannot race a forward or host-key operation onto a different machine.
func (s *Store) toolsRequest(operation string, payload any, machineID string) {
	s.mu.Lock()
	if s.client == nil || s.view.Busy ||
		((operation == "forwards" || operation == "forget-key") && s.view.CurrentID == "") {
		s.mu.Unlock()
		return
	}
	s.view.Busy = true
	s.view.Tools.Error, s.view.Tools.Notice = "", ""
	c, n, g := s.client, s.notify, s.generation
	value := cloneTools(s.view.Tools)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	s.cancel = cancel
	s.mu.Unlock()
	if n != nil {
		n()
	}
	go func() {
		defer cancel()
		var err error
		switch operation {
		case "load":
			var entries struct {
				Entries []SSHConfigEntry `json:"entries"`
			}
			err = request(ctx, c, "GET", "ssh-config", nil, &entries)
			if err == nil {
				value.SSHEntries = entries.Entries
			}
		case "audit":
			var audit struct {
				AuditEnabled bool     `json:"auditEnabled"`
				Lines        []string `json:"lines"`
			}
			err = request(ctx, c, "GET", "audit?limit=100", nil, &audit)
			if err == nil {
				value.AuditLoaded, value.AuditEnabled = true, audit.AuditEnabled
				value.AuditLines = audit.Lines
			}
		case "test-connect":
			var probe struct {
				LatencyMS int    `json:"latencyMs"`
				Platform  string `json:"platform"`
			}
			err = request(ctx, c, "POST", operation, payload, &probe)
			if err == nil {
				value.Notice = fmt.Sprintf("Connected: %d ms · %s", probe.LatencyMS, probe.Platform)
			}
		case "forwards", "forget-key":
			err = request(ctx, c, "POST", operation, payload, nil)
			if err == nil && operation == "forget-key" {
				value.Notice = "Host key forgotten. Verify the fingerprint before reconnecting."
			}
		default:
			err = fmt.Errorf("Unknown remote settings operation")
		}
		if err == nil && (operation == "load" || operation == "forwards") {
			var list struct {
				Forwards []Forward `json:"forwards"`
			}
			err = request(ctx, c, "GET", "forwards", nil, &list)
			if err == nil {
				value.Forwards = list.Forwards
			}
		}
		var status ConnectionStatus
		if err == nil && operation == "forget-key" {
			err = request(ctx, c, "GET", "status", nil, &status)
		}
		s.mu.Lock()
		if g != s.generation {
			s.mu.Unlock()
			return
		}
		s.cancel = nil
		s.view.Busy = false
		if operation == "load" {
			value.Loaded = true
		}
		if err != nil {
			value.Error = err.Error()
			// A successful mutation followed by a failed refresh is not retried.
			if operation == "forwards" {
				value.Notice = "Refresh the forwarding list before retrying an uncertain operation."
			}
		} else if operation == "forget-key" {
			s.view.Status, s.view.Connected = status, status.Connected
		}
		s.view.Tools = value
		s.mu.Unlock()
		if n != nil {
			n()
		}
	}()
}
