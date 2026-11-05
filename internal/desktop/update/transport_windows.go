//go:build windows

package update

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var winHTTP = windows.NewLazySystemDLL("winhttp.dll")
var winHttpIEConfig = winHTTP.NewProc("WinHttpGetIEProxyConfigForCurrentUser")
var winHttpOpen = winHTTP.NewProc("WinHttpOpen")
var winHttpProxyForURL = winHTTP.NewProc("WinHttpGetProxyForUrl")
var winHttpTimeouts = winHTTP.NewProc("WinHttpSetTimeouts")
var winHttpClose = winHTTP.NewProc("WinHttpCloseHandle")
var globalFreeProxy = windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalFree")

type ieProxyConfig struct {
	AutoDetect                   uint32
	AutoConfigURL, Proxy, Bypass *uint16
}
type autoProxyOptions struct {
	Flags, AutoDetectFlags           uint32
	AutoConfigURL                    *uint16
	Reserved                         uintptr
	Reserved1, AutoLogonIfChallenged uint32
}
type proxyInfo struct {
	AccessType    uint32
	Proxy, Bypass *uint16
}

func freeProxyString(p *uint16) {
	if p != nil {
		globalFreeProxy.Call(uintptr(unsafe.Pointer(p)))
	}
}
func proxyString(p *uint16) string {
	if p == nil {
		return ""
	}
	return windows.UTF16PtrToString(p)
}

func updateProxy(req *http.Request) (*url.URL, error) {
	host := req.URL.Hostname()
	if strings.EqualFold(host, "localhost") {
		return nil, nil
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return nil, nil
	}
	for _, name := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
		if os.Getenv(name) != "" {
			return http.ProxyFromEnvironment(req)
		}
	}
	var config ieProxyConfig
	ok, _, configErr := winHttpIEConfig.Call(uintptr(unsafe.Pointer(&config)))
	if ok == 0 {
		if configErr == windows.ERROR_FILE_NOT_FOUND {
			return nil, nil
		}
		return nil, fmt.Errorf("read system update proxy: %w", configErr)
	}
	defer freeProxyString(config.AutoConfigURL)
	defer freeProxyString(config.Proxy)
	defer freeProxyString(config.Bypass)
	proxy, bypass := proxyString(config.Proxy), proxyString(config.Bypass)
	if config.AutoDetect != 0 || config.AutoConfigURL != nil {
		name, _ := windows.UTF16PtrFromString("WhaleIsle-MyGo updater")
		h, _, err := winHttpOpen.Call(uintptr(unsafe.Pointer(name)), 1, 0, 0, 0)
		if h == 0 {
			return nil, fmt.Errorf("open system proxy resolver: %w", err)
		}
		defer winHttpClose.Call(h)
		winHttpTimeouts.Call(h, 10000, 10000, 10000, 10000)
		options := autoProxyOptions{AutoLogonIfChallenged: 1}
		if config.AutoConfigURL != nil {
			options.Flags = 2
			options.AutoConfigURL = config.AutoConfigURL
		} else {
			options.Flags = 1
			options.AutoDetectFlags = 3
		}
		address, _ := windows.UTF16PtrFromString(req.URL.String())
		var info proxyInfo
		ok, _, resolveErr := winHttpProxyForURL.Call(h, uintptr(unsafe.Pointer(address)), uintptr(unsafe.Pointer(&options)), uintptr(unsafe.Pointer(&info)))
		if ok == 0 {
			// WPAD discovery is optional. No discovered PAC falls back to the
			// configured static proxy/direct route; an explicit/broken PAC does not.
			if config.AutoConfigURL == nil && resolveErr == syscall.Errno(12180) {
				return proxyAddress(req.URL, proxy, bypass)
			}
			return nil, fmt.Errorf("resolve system proxy/PAC: %w", resolveErr)
		}
		defer freeProxyString(info.Proxy)
		defer freeProxyString(info.Bypass)
		proxy, bypass = proxyString(info.Proxy), proxyString(info.Bypass)
	}
	return proxyAddress(req.URL, proxy, bypass)
}

func proxyAddress(target *url.URL, proxies, bypass string) (*url.URL, error) {
	host := strings.ToLower(target.Hostname())
	for _, pattern := range strings.FieldsFunc(strings.ToLower(bypass), func(r rune) bool { return r == ';' || r == ',' || r == ' ' }) {
		if pattern == "<local>" && !strings.Contains(host, ".") {
			return nil, nil
		}
		if matched, _ := filepath.Match(pattern, host); matched {
			return nil, nil
		}
	}
	selected := ""
	for _, entry := range strings.FieldsFunc(proxies, func(r rune) bool { return r == ';' || r == ' ' }) {
		if pair := strings.SplitN(entry, "=", 2); len(pair) == 2 {
			if strings.EqualFold(pair[0], target.Scheme) {
				selected = pair[1]
				break
			}
		} else if selected == "" {
			selected = entry
		}
	}
	if selected == "" || strings.EqualFold(selected, "DIRECT") {
		return nil, nil
	}
	if !strings.Contains(selected, "://") {
		selected = "http://" + selected
	}
	u, err := url.Parse(selected)
	if err != nil || u.Hostname() == "" {
		return nil, fmt.Errorf("invalid system update proxy")
	}
	return u, nil
}
