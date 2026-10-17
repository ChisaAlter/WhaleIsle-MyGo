package openui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"whaleisle/internal/desktop/imagepreview"
)

var errNonPublicImageAddress = errors.New("image source must use a public network address")

// IsGlobalUnicast alone includes private, documentation and special-use space.
// IPv6 is additionally restricted to current global-unicast allocation, so
// translated/transition addresses cannot encode otherwise-blocked IPv4 targets.
var publicImageExcluded = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("3fff::/20"),
}

var publicImageIPv6 = netip.MustParsePrefix("2000::/3")

func publicImageAddress(ip netip.Addr) bool {
	if ip.Zone() != "" {
		return false
	}
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || (ip.Is6() && !publicImageIPv6.Contains(ip)) {
		return false
	}
	for _, prefix := range publicImageExcluded {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}

func publicImageURL(raw string) (*url.URL, error) {
	if len(raw) == 0 || len(raw) > MaxEventBytes || strings.TrimSpace(raw) != raw {
		return nil, errors.New("src must be a bounded absolute HTTP(S) image URL")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Opaque != "" || u.User != nil || u.Hostname() == "" {
		return nil, errors.New("src must be an absolute HTTP(S) image URL without credentials")
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if strings.Contains(host, "%") || host == "localhost" || host == "local" || host == "internal" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") || host == "home.arpa" || strings.HasSuffix(host, ".home.arpa") {
		return nil, errNonPublicImageAddress
	}
	if ip, parseErr := netip.ParseAddr(host); parseErr == nil && !publicImageAddress(ip) {
		return nil, errNonPublicImageAddress
	}
	if port := u.Port(); port != "" {
		n, parseErr := strconv.Atoi(port)
		if parseErr != nil || n < 1 || n > 65535 {
			return nil, errors.New("invalid public image URL port")
		}
	} else if strings.HasSuffix(u.Host, ":") {
		return nil, errors.New("invalid public image URL port")
	}
	u.Fragment, u.RawFragment = "", ""
	return u, nil
}

// Resolve and dial in the same operation. The socket receives a validated IP,
// never the hostname for a second lookup; HTTP Host and TLS verification still
// use the original URL. A mixed public/private DNS answer is rejected outright.
func dialPublicImage(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, errors.New("image hostname has no public address")
	}
	for _, ip := range ips {
		if !publicImageAddress(ip) {
			return nil, errNonPublicImageAddress
		}
	}
	var last error
	dialer := net.Dialer{Timeout: 8 * time.Second}
	for _, ip := range ips {
		conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.Unmap().String(), port))
		if dialErr == nil {
			return conn, nil
		}
		last = dialErr
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	return nil, last
}

func (a *ResourceAPI) publicImage(ctx context.Context, e Event) APIResponse {
	u, err := publicImageURL(e.Query["src"])
	if err != nil {
		status := 400
		if errors.Is(err, errNonPublicImageAddress) {
			status = 403
		}
		return apiError(e, status, err.Error())
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	// A fresh direct transport does not inherit proxy settings, a cookie jar,
	// Host sessions, authentication headers, cached connections or TLS overrides.
	transport := &http.Transport{
		Proxy: nil, DialContext: dialPublicImage, DisableKeepAlives: true,
		DisableCompression: true, TLSHandshakeTimeout: 8 * time.Second,
		ResponseHeaderTimeout: 8 * time.Second, MaxResponseHeaderBytes: MaxEventBytes,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	for redirects := 0; ; redirects++ {
		req, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if requestErr != nil {
			return apiError(e, 400, requestErr.Error())
		}
		res, requestErr := client.Do(req)
		if requestErr != nil {
			status := 502
			if errors.Is(requestErr, errNonPublicImageAddress) {
				status = 403
			} else if ctx.Err() != nil {
				status = 504
			}
			return apiError(e, status, "Public image request failed: "+requestErr.Error())
		}
		switch res.StatusCode {
		case 301, 302, 303, 307, 308:
			next, locationErr := res.Location()
			res.Body.Close()
			if redirects >= 4 {
				return apiError(e, 502, "Public image exceeded the redirect limit")
			}
			if locationErr != nil {
				return apiError(e, 502, "Public image returned an invalid redirect")
			}
			u, err = publicImageURL(next.String())
			if err != nil {
				return apiError(e, 403, "Public image redirect rejected: "+err.Error())
			}
			// A new request deliberately carries no cookies, Referer, auth or
			// upstream headers from the previous response, even on the same host.
			continue
		}
		if res.StatusCode != http.StatusOK {
			res.Body.Close()
			return apiError(e, 502, fmt.Sprintf("Public image returned HTTP %d", res.StatusCode))
		}
		if res.ContentLength > imagepreview.MaxBytes {
			res.Body.Close()
			return apiError(e, 413, "Image exceeds the 16 MiB byte budget")
		}
		data, readErr := io.ReadAll(io.LimitReader(res.Body, imagepreview.MaxBytes+1))
		res.Body.Close()
		if readErr != nil {
			status := 502
			if ctx.Err() != nil {
				status = 504
			}
			return apiError(e, status, "Public image body could not be read: "+readErr.Error())
		}
		if int64(len(data)) > imagepreview.MaxBytes {
			return apiError(e, 413, "Image exceeds the 16 MiB byte budget")
		}
		source, imageErr := imagepreview.ResourceDataURL(data)
		if imageErr != nil {
			return apiError(e, 415, "Unsupported or invalid public image: "+imageErr.Error())
		}
		body := map[string]any{"dataUrl": source, "source": u.String()}
		if strings.HasPrefix(source, "data:image/svg+xml;base64,") {
			// Only this Host-validated response authorizes a static SVG image;
			// unvalidated program-supplied SVG data URLs remain unsupported.
			body["validatedSvg"] = true
		}
		return apiResult(e, 200, body)
	}
}
