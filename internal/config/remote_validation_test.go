package config

import "testing"

func TestExplicitRemoteAddressValidation(t *testing.T) {
	for _, address := range []string{"127.0.0.1", "0.0.0.0", "192.168.1.42", " 10.0.0.2 "} {
		if err := ValidateRemoteBindAddress(address); err != nil {
			t.Errorf("valid bind %q: %v", address, err)
		}
	}
	for _, address := range []string{"", "not-an-ip", "127.0.0.999", "::1", "127.0.0.1\x00", "https://127.0.0.1"} {
		if err := ValidateRemoteBindAddress(address); err == nil {
			t.Errorf("invalid bind %q accepted", address)
		}
	}
	for _, page := range []string{"", "https://example.com/pair", " http://public.example.test:8080/pair/ ", "https://ayase.cn/dshd"} {
		if err := ValidateRemoteAppBaseURL(page, "relay.example.test:443"); err != nil {
			t.Errorf("valid public page %q: %v", page, err)
		}
	}
	for _, page := range []string{"not-a-url", "ftp://example.com", "https://127.0.0.1/pair", "https://192.168.1.42/pair", "http://[fd00::1]/page", "http://LOCALHOST/page", "http://[::ffff:127.0.0.1]/page", "https://user:password@example.com/pair", "https://example.com/pair?token=x", "https://example.com/pair#offer=x", "https://example.com:0/pair", "https://example.com:99999/pair", "https://relay.example.test:443/pair"} {
		if err := ValidateRemoteAppBaseURL(page, "relay.example.test:443"); err == nil {
			t.Errorf("invalid public page %q accepted", page)
		}
	}
}
