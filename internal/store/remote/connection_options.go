package remote

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// ConnectionDraft is window-local form state. The host owns credential storage.
type ConnectionDraft struct {
	Password, Passphrase, HostKeyMode                                string
	UseAgent, KeyboardInteractive, EncryptPassword                   bool
	ProxyHost, ProxyPort, ProxyUsername, ProxyPassword, ProxyKeyPath string
}

func NewDraft() Draft {
	return Draft{Port: "22", Username: "root", Options: &ConnectionDraft{ProxyPort: "22"}}
}

func (m Machine) EditDraft() Draft {
	d := NewDraft()
	d.ID, d.Name, d.Host, d.Port = m.ID, m.Name, m.Host, strconv.Itoa(m.Port)
	d.Username, d.PrivateKeyPath, d.Workspace = m.Username, m.PrivateKeyPath, m.Workspace
	var saved struct {
		Passphrase          string `json:"passphrase"`
		HostKeyMode         string `json:"hostKeyMode"`
		UseAgent            bool   `json:"useAgent"`
		KeyboardInteractive bool   `json:"keyboardInteractive"`
		CredentialBackend   string `json:"credentialBackend"`
		Proxy               struct {
			Host           string `json:"host"`
			Port           int    `json:"port"`
			Username       string `json:"username"`
			PrivateKeyPath string `json:"privateKeyPath"`
		} `json:"proxy"`
	}
	raw, _ := json.Marshal(m.raw)
	_ = json.Unmarshal(raw, &saved)
	o := d.Options
	o.Passphrase, o.HostKeyMode = saved.Passphrase, saved.HostKeyMode
	o.UseAgent, o.KeyboardInteractive = saved.UseAgent, saved.KeyboardInteractive
	o.EncryptPassword = saved.CredentialBackend != "" && saved.CredentialBackend != "plain"
	o.ProxyHost, o.ProxyUsername, o.ProxyKeyPath = saved.Proxy.Host, saved.Proxy.Username, saved.Proxy.PrivateKeyPath
	if saved.Proxy.Port > 0 {
		o.ProxyPort = strconv.Itoa(saved.Proxy.Port)
	}
	return d
}

func parsePort(value string, fallback int) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 || n > 65535 {
		return 0, fmt.Errorf("Port must be 1–65535")
	}
	return n, nil
}

func (o ConnectionDraft) apply(body map[string]any) error {
	if o.HostKeyMode != "" && o.HostKeyMode != "verify" && o.HostKeyMode != "off" {
		return fmt.Errorf("Invalid host-key mode")
	}
	// Empty passwords are omitted. The existing host contract keeps saved secrets.
	delete(body, "password")
	if o.Password != "" {
		body["password"] = o.Password
	}
	body["passphrase"], body["hostKeyMode"] = o.Passphrase, o.HostKeyMode
	body["useAgent"], body["keyboardInteractive"] = o.UseAgent, o.KeyboardInteractive
	body["encryptPassword"] = o.EncryptPassword
	// The host chooses the OS credential backend when encryption is requested.
	delete(body, "credentialBackend")
	delete(body, "proxy")
	if host := strings.TrimSpace(o.ProxyHost); host != "" {
		port, err := parsePort(o.ProxyPort, 22)
		if err != nil {
			return fmt.Errorf("Proxy: %w", err)
		}
		proxy := map[string]any{
			"host": host, "port": port, "username": strings.TrimSpace(o.ProxyUsername),
			"privateKeyPath": strings.TrimSpace(o.ProxyKeyPath),
		}
		if o.ProxyPassword != "" {
			proxy["password"] = o.ProxyPassword
		}
		body["proxy"] = proxy
	}
	return nil
}

type SSHConfigEntry struct {
	Host         string `json:"host"`
	HostName     string `json:"hostName"`
	User         string `json:"user"`
	Port         int    `json:"port"`
	IdentityFile string `json:"identityFile"`
	ProxyJump    string `json:"proxyJump"`
}

func (e SSHConfigEntry) Draft() (Draft, error) {
	d := NewDraft()
	d.Name, d.Host, d.PrivateKeyPath = e.Host, e.HostName, e.IdentityFile
	if d.Host == "" {
		d.Host = e.Host
	}
	if e.Port > 0 {
		d.Port = strconv.Itoa(e.Port)
	}
	if e.User != "" {
		d.Username = e.User
	}
	if err := importProxyJump(d.Options, e.ProxyJump); err != nil {
		return Draft{}, err
	}
	return d, nil
}

// The SSH config parser returns raw ProxyJump text. The host takes one
// structured proxy, so never pass user@host:port or a jump chain as a hostname.
func importProxyJump(o *ConnectionDraft, value string) error {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "none") {
		return nil
	}
	if strings.ContainsAny(value, ", \t\r\n") {
		return fmt.Errorf("Multi-hop ProxyJump is not supported; configure one jump host manually")
	}
	if at := strings.LastIndex(value, "@"); at >= 0 {
		o.ProxyUsername, value = value[:at], value[at+1:]
		if o.ProxyUsername == "" || strings.Contains(o.ProxyUsername, "@") {
			return fmt.Errorf("Invalid ProxyJump username")
		}
	}
	host, port, err := net.SplitHostPort(value)
	if err == nil {
		n, err := parsePort(port, 0)
		if err != nil || n == 0 {
			return fmt.Errorf("Invalid ProxyJump port")
		}
		o.ProxyHost, o.ProxyPort = host, strconv.Itoa(n)
	} else if strings.Count(value, ":") == 1 {
		return fmt.Errorf("Invalid ProxyJump host or port")
	} else {
		host = strings.Trim(value, "[]")
		if strings.Contains(host, ":") && net.ParseIP(host) == nil {
			return fmt.Errorf("Invalid ProxyJump address")
		}
		o.ProxyHost = host
	}
	if o.ProxyHost == "" {
		return fmt.Errorf("ProxyJump host is required")
	}
	return nil
}
