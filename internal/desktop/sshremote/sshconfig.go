package sshremote

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// SSHConfigEntry is the non-secret portion of one OpenSSH Host stanza.
// IdentityFile is intentionally kept as a path; importing a config must not
// read private key material.
type SSHConfigEntry struct {
	Host         string `json:"host"`
	HostName     string `json:"hostName"`
	User         string `json:"user"`
	Port         int    `json:"port"`
	IdentityFile string `json:"identityFile"`
	ProxyJump    string `json:"proxyJump"`
}

// SSHConfigPath returns the user's OpenSSH config path. An explicit home
// argument is useful for tests and for a desktop profile with a redirected
// home directory.
func SSHConfigPath(home ...string) string {
	base := ""
	if len(home) > 0 {
		base = strings.TrimSpace(home[0])
	}
	if base == "" {
		base, _ = os.UserHomeDir()
	}
	return filepath.Join(base, ".ssh", "config")
}

// ReadSSHConfig reads path and returns an empty list when the file is absent.
func ReadSSHConfig(path string) ([]SSHConfigEntry, error) {
	if strings.TrimSpace(path) == "" {
		path = SSHConfigPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return ParseSSHConfig(string(data)), nil
}

// ParseSSHConfig parses the Host/HostName/User/Port/IdentityFile/ProxyJump
// subset used by the native remote settings. Unknown directives and Include
// lines are ignored, matching the reference dsh-remote parser.
func ParseSSHConfig(text string) []SSHConfigEntry {
	var out []SSHConfigEntry
	current := -1
	for _, raw := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// OpenSSH allows comments after a directive. Preserve '#' in a path
		// only when it is escaped; the subset has no other quoting semantics.
		line = stripSSHConfigComment(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		key := strings.ToLower(parts[0])
		value := strings.TrimSpace(line[len(parts[0]):])
		switch key {
		case "host":
			out = append(out, SSHConfigEntry{Host: value, Port: 22})
			current = len(out) - 1
		case "hostname":
			if current >= 0 {
				out[current].HostName = value
			}
		case "user":
			if current >= 0 {
				out[current].User = value
			}
		case "port":
			if current >= 0 {
				if port, err := strconv.Atoi(value); err == nil && port > 0 && port <= 65535 {
					out[current].Port = port
				}
			}
		case "identityfile":
			if current >= 0 {
				out[current].IdentityFile = value
			}
		case "proxyjump":
			if current >= 0 {
				out[current].ProxyJump = value
			}
		case "include":
			// Includes are intentionally not expanded: doing so would turn a
			// read-only import into filesystem traversal with surprising
			// precedence rules.
		}
	}
	return out
}

// ImportableSSHConfigEntries filters wildcard, negated and comma-list Host
// patterns. The remaining entries are concrete aliases a user can choose in
// the remote settings panel.
func ImportableSSHConfigEntries(text string) []SSHConfigEntry {
	entries := ParseSSHConfig(text)
	out := make([]SSHConfigEntry, 0, len(entries))
	for _, entry := range entries {
		host := strings.TrimSpace(entry.Host)
		if host == "" || strings.ContainsAny(host, "*!?,") {
			continue
		}
		if entry.HostName == "" && entry.User == "" && entry.Port == 22 && entry.IdentityFile == "" && entry.ProxyJump == "" {
			continue
		}
		entry.Host = host
		out = append(out, entry)
	}
	return out
}

func stripSSHConfigComment(line string) string {
	escaped := false
	for i, r := range line {
		switch {
		case r == '\\' && !escaped:
			escaped = true
		case r == '#' && !escaped:
			return strings.TrimSpace(line[:i])
		default:
			escaped = false
		}
	}
	return strings.TrimSpace(line)
}

// ValidateSSHConfigEntry performs the same concrete-entry checks used before
// importing a profile into a saved connection.
func ValidateSSHConfigEntry(entry SSHConfigEntry) error {
	if strings.TrimSpace(entry.Host) == "" {
		return errors.New("ssh config host required")
	}
	if strings.ContainsAny(entry.Host, "*!?,") {
		return errors.New("ssh config host must be a concrete alias")
	}
	if entry.Port < 1 || entry.Port > 65535 {
		return errors.New("ssh config port out of range")
	}
	return nil
}
