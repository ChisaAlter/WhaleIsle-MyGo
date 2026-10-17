package config

import (
	"errors"
	"net"
	"strconv"
	"strings"
)

// RemoteGatewayPreferences contains no credentials or device identities.
type RemoteGatewayPreferences struct {
	Mode, Port, BindAddress, RelayEndpoint string
	RelayTLS                               bool
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
	if p.Mode != "lan" && p.Mode != "relay" && p.Mode != "ssh" {
		return errors.New("Choose LAN, relay or SSH / 请选择连接模式")
	}
	port, err := strconv.Atoi(p.Port)
	if err != nil || port < 1024 || port > 65535 {
		return errors.New("Port must be 1024–65535 / 端口必须为 1024–65535")
	}
	if net.ParseIP(p.BindAddress) == nil || strings.Contains(p.BindAddress, ":") || normalizeBindAddress(p.BindAddress) != p.BindAddress {
		return errors.New("Bind address must be an IPv4 address / 绑定地址必须是 IPv4 地址")
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
