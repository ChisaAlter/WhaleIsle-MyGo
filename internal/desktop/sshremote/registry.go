package sshremote

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ProxyConfig describes an optional SSH jump/proxy connection. Passwords are
// kept in the local registry only when a caller explicitly supplies them; UI
// responses should use SanitizedConnection to avoid exposing them.
type ProxyConfig struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	User         string `json:"username,omitempty"`
	Password     string `json:"password,omitempty"`
	IdentityFile string `json:"privateKeyPath,omitempty"`
}

// SavedConnection mirrors the dsh-remote machine registry. A saved entry is a
// standby profile; saving or updating it never changes CurrentID.
type SavedConnection struct {
	ID                  string       `json:"id"`
	Name                string       `json:"name"`
	Host                string       `json:"host"`
	Port                int          `json:"port"`
	User                string       `json:"username"`
	Password            string       `json:"password,omitempty"`
	IdentityFile        string       `json:"privateKeyPath,omitempty"`
	Passphrase          string       `json:"passphrase,omitempty"`
	Workspace           string       `json:"workspace,omitempty"`
	HostKeyMode         string       `json:"hostKeyMode,omitempty"`
	UseAgent            bool         `json:"useAgent,omitempty"`
	KeyboardInteractive bool         `json:"keyboardInteractive,omitempty"`
	LocalForward        string       `json:"localForward,omitempty"`
	Proxy               *ProxyConfig `json:"proxy,omitempty"`
	RecentWorkspaces    []string     `json:"recentWorkspaces,omitempty"`
	LastConnectedAt     string       `json:"lastConnectedAt,omitempty"`
	LatencyMs           int64        `json:"latencyMs,omitempty"`
}

// SanitizedConnection strips credentials before a profile is sent to a UI.
type SanitizedConnection struct {
	SavedConnection
	PasswordSet      bool `json:"passwordSet"`
	PassphraseSet    bool `json:"passphraseSet"`
	ProxyPasswordSet bool `json:"proxyPasswordSet,omitempty"`
}

// ConnectionRegistry is the persisted machine list and explicit active
// selection. CurrentID == "" means no current machine; ExplicitNone records
// that the user deliberately cleared it so a restart cannot auto-promote one.
type ConnectionRegistry struct {
	List         []SavedConnection `json:"list"`
	CurrentID    string            `json:"currentId,omitempty"`
	ExplicitNone bool              `json:"-"`
}

type registryDocument struct {
	List      []SavedConnection `json:"list"`
	CurrentID json.RawMessage   `json:"currentId,omitempty"`
}

// RegistryPath is the profile file used by the native SSH surface.
func RegistryPath(userData string) string {
	return filepath.Join(userData, "ssh", "machines.json")
}

// LoadConnectionRegistry loads a registry document. Missing files are a new
// registry and do not create directories or files.
func LoadConnectionRegistry(path string) (ConnectionRegistry, error) {
	if strings.TrimSpace(path) == "" {
		return ConnectionRegistry{}, errors.New("registry path required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ConnectionRegistry{}, nil
		}
		return ConnectionRegistry{}, err
	}
	var doc registryDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return ConnectionRegistry{}, fmt.Errorf("ssh registry: %w", err)
	}
	reg := ConnectionRegistry{List: cloneConnections(doc.List)}
	if len(doc.CurrentID) > 0 {
		if string(doc.CurrentID) == "null" {
			reg.ExplicitNone = true
		} else {
			var current string
			if err := json.Unmarshal(doc.CurrentID, &current); err != nil {
				return ConnectionRegistry{}, fmt.Errorf("ssh registry: currentId: %w", err)
			}
			reg.CurrentID = strings.TrimSpace(current)
			reg.ExplicitNone = reg.CurrentID == ""
		}
	}
	if reg.CurrentID != "" && !connectionExists(reg.List, reg.CurrentID) {
		reg.CurrentID = ""
	}
	return reg, nil
}

// SaveConnectionRegistry writes a registry atomically and preserves the
// source shell's no-auto-promotion semantics.
func SaveConnectionRegistry(path string, reg ConnectionRegistry) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("registry path required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	list := cloneConnections(reg.List)
	current := strings.TrimSpace(reg.CurrentID)
	if current != "" && !connectionExists(list, current) {
		return fmt.Errorf("ssh registry: current connection %q not found", current)
	}
	doc := registryDocument{List: list}
	if reg.ExplicitNone || current != "" {
		if reg.ExplicitNone {
			doc.CurrentID = json.RawMessage("null")
		} else {
			encoded, err := json.Marshal(current)
			if err != nil {
				return err
			}
			doc.CurrentID = encoded
		}
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".machines-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

// SaveConnectionList updates the list while preserving the currentId key
// already on disk. This is used by Add/Update forms.
func SaveConnectionList(path string, list []SavedConnection) error {
	reg, err := LoadConnectionRegistry(path)
	if err != nil {
		return err
	}
	reg.List = cloneConnections(list)
	return SaveConnectionRegistry(path, reg)
}

// UpsertConnection adds or replaces a profile without activating it.
func UpsertConnection(reg *ConnectionRegistry, connection SavedConnection) error {
	if reg == nil {
		return errors.New("registry required")
	}
	if strings.TrimSpace(connection.Host) == "" {
		return errors.New("ssh connection host required")
	}
	if connection.Port <= 0 {
		connection.Port = 22
	}
	if connection.Port > 65535 {
		return errors.New("ssh connection port out of range")
	}
	if strings.TrimSpace(connection.ID) == "" {
		connection.ID = NewConnectionID()
	}
	if strings.TrimSpace(connection.Name) == "" {
		connection.Name = connection.Host
	}
	connection.HostKeyMode = string(NormalizeHostKeyMode(connection.HostKeyMode))
	connection.RecentWorkspaces = append([]string(nil), connection.RecentWorkspaces...)
	for i := range reg.List {
		if reg.List[i].ID == connection.ID {
			reg.List[i] = connection
			return nil
		}
	}
	reg.List = append(reg.List, connection)
	return nil
}

// DeleteConnection removes a saved profile. Deleting the current profile
// leaves an explicit active-none state and never promotes another entry.
func DeleteConnection(reg *ConnectionRegistry, id string) bool {
	if reg == nil {
		return false
	}
	for i, connection := range reg.List {
		if connection.ID != id {
			continue
		}
		reg.List = append(reg.List[:i], reg.List[i+1:]...)
		if reg.CurrentID == id {
			reg.CurrentID = ""
			reg.ExplicitNone = true
		}
		return true
	}
	return false
}

// SetCurrent selects an existing profile or explicitly clears the selection.
func SetCurrentConnection(reg *ConnectionRegistry, id string) error {
	if reg == nil {
		return errors.New("registry required")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		reg.CurrentID = ""
		reg.ExplicitNone = true
		return nil
	}
	if !connectionExists(reg.List, id) {
		return fmt.Errorf("ssh registry: connection %q not found", id)
	}
	reg.CurrentID = id
	reg.ExplicitNone = false
	return nil
}

// Sanitized returns a connection safe for a settings response.
func (c SavedConnection) Sanitized() SanitizedConnection {
	out := SanitizedConnection{SavedConnection: c, PasswordSet: c.Password != "", PassphraseSet: c.Passphrase != ""}
	out.SavedConnection.Password = ""
	out.SavedConnection.Passphrase = ""
	if c.Proxy != nil {
		proxy := *c.Proxy
		out.ProxyPasswordSet = proxy.Password != ""
		proxy.Password = ""
		out.SavedConnection.Proxy = &proxy
	}
	return out
}

// NewConnectionID creates a stable-looking random profile id. Timestamp keeps
// ids sortable while the random suffix prevents collisions in one process.
func NewConnectionID() string {
	var suffix [4]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return fmt.Sprintf("m-%x", time.Now().UnixNano())
	}
	return fmt.Sprintf("m-%x-%s", time.Now().UnixMilli(), hex.EncodeToString(suffix[:]))
}

func connectionExists(list []SavedConnection, id string) bool {
	for _, connection := range list {
		if connection.ID == id {
			return true
		}
	}
	return false
}

func cloneConnections(list []SavedConnection) []SavedConnection {
	if list == nil {
		return nil
	}
	out := make([]SavedConnection, len(list))
	copy(out, list)
	for i := range out {
		out[i].RecentWorkspaces = append([]string(nil), out[i].RecentWorkspaces...)
		if out[i].Proxy != nil {
			proxy := *out[i].Proxy
			out[i].Proxy = &proxy
		}
	}
	return out
}
