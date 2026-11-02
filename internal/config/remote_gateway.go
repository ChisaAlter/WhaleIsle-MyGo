package config

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// RemoteGatewayPreferences contains no credentials or device identities.
type RemoteGatewayPreferences struct {
	Mode, Port, BindAddress, RelayEndpoint string
	RelayTLS                               bool
}

// ValidateRemoteBindAddress validates an explicit write before normalization.
// Invalid input must not silently become the loopback default in a mixed patch.
func ValidateRemoteBindAddress(value string) error {
	value = strings.TrimSpace(value)
	if net.ParseIP(value) == nil || strings.Contains(value, ":") || normalizeBindAddress(value) != value {
		return errors.New("Bind address must be an IPv4 address / 绑定地址必须是 IPv4 地址")
	}
	return nil
}

// ValidateRemoteAppBaseURL accepts an empty reset or a public HTTP(S) pairing
// page. It uses the stored config's normalization contract but rejects discarded
// user information, query/fragment text and malformed or private destinations.
func ValidateRemoteAppBaseURL(value, relay string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	u, err := url.Parse(value)
	if err != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.ForceQuery || strings.ContainsAny(value, "\\\r\n\t") || normalizePublicAppBaseURL(value, relay) == "" {
		return errors.New("Pairing page must be a public HTTP(S) URL without credentials, query or fragment / 配对页必须是公开 HTTP(S) 地址，不能包含凭据、查询或片段")
	}
	hostname := strings.ToLower(u.Hostname())
	address := net.ParseIP(hostname)
	if strings.EqualFold(hostname, "localhost") || (address != nil && (!address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() || address.IsLinkLocalUnicast() || address.IsUnspecified())) {
		return errors.New("Pairing page must use a public destination / 配对页必须使用公开地址")
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return errors.New("Invalid pairing page port / 配对页端口无效")
		}
	}
	return nil
}

func RemoteGatewayFrom(c Config) RemoteGatewayPreferences {
	return RemoteGatewayPreferences{Mode: c.RemoteMode, Port: strconv.Itoa(c.RemotePort), BindAddress: c.RemoteBindAddress, RelayEndpoint: c.RemoteRelayEndpoint, RelayTLS: c.RemoteRelayUseTls}
}

// SaveRemoteGateway rejects invalid input rather than silently replacing it with
// defaults. Existing normalization remains authoritative on the persisted copy.
func (s *Store) SaveRemoteGateway(expected, p RemoteGatewayPreferences) error {
	p.Mode = strings.TrimSpace(p.Mode)
	p.Port = strings.TrimSpace(p.Port)
	p.BindAddress = strings.TrimSpace(p.BindAddress)
	p.RelayEndpoint = strings.TrimSpace(p.RelayEndpoint)
	// The gateway has exactly two transports (native config.js:235); SSH is a
	// remote-workspace transport and is rejected here instead of being stored.
	if p.Mode != "lan" && p.Mode != "relay" {
		return errors.New("Choose LAN or relay / 请选择连接模式")
	}
	port, err := strconv.Atoi(p.Port)
	if err != nil || port < 1024 || port > 65535 {
		return errors.New("Port must be 1024–65535 / 端口必须为 1024–65535")
	}
	if err := ValidateRemoteBindAddress(p.BindAddress); err != nil {
		return err
	}
	if p.RelayEndpoint != "" {
		if strings.ContainsAny(p.RelayEndpoint, "/@?# \t\r\n\\") || normalizeRelayEndpoint(p.RelayEndpoint) == "" {
			return errors.New("Relay must be host[:port], without scheme, path or credentials / 中继填写主机[:端口]")
		}
		if i := strings.LastIndex(p.RelayEndpoint, ":"); i >= 0 {
			n, e := strconv.Atoi(p.RelayEndpoint[i+1:])
			if e != nil || n < 1 || n > 65535 {
				return errors.New("Invalid relay port / 中继端口无效")
			}
		}
	}
	stale := false
	err = s.Update(func(c *Config) {
		if RemoteGatewayFrom(*c) != expected {
			stale = true
			return
		}
		c.RemoteMode = p.Mode
		c.RemotePort = port
		c.RemoteBindAddress = p.BindAddress
		c.RemoteRelayUrl = p.RelayEndpoint
		c.RemoteRelayEndpoint = p.RelayEndpoint
		c.RemoteRelayUseTls = p.RelayTLS
	})
	if err != nil {
		return err
	}
	if stale {
		return errors.New("Gateway changed; reload before saving / 网关已变更，请重新载入")
	}
	return nil
}
