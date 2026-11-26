// Package wallpapers implements the original e9 wallpaper catalog contracts.
package wallpapers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"whaleisle/internal/store/storelocale"
)

type Source struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}
type Item struct {
	ID        string `json:"id"`
	SourceID  string `json:"sourceId"`
	Title     string `json:"title"`
	Copyright string `json:"copyright,omitempty"`
	ThumbURL  string `json:"thumbUrl"`
	ImageURL  string `json:"imageUrl"`
}
type Query struct {
	Source     Source
	Search     string
	Categories string
	Year       int
	Page       int
}
type Result struct {
	Items    []Item
	NextPage int
	Warning  string
}

func Sources() []Source {
	return []Source{{ID: "bing", Kind: "bing", Name: storelocale.T(storelocale.KeyStoreWallpapersBing)}, {ID: "wallhaven", Kind: "wallhaven", Name: "Wallhaven"}}
}

func allowTestHTTP() bool {
	return os.Getenv("DSHD_WALLPAPER_ALLOW_HTTP") == "1"
}

var (
	blockedIPv4Prefixes = mustPrefixes(
		"0.0.0.0/8",
		"10.0.0.0/8",
		"100.64.0.0/10",
		"169.254.0.0/16",
		"172.16.0.0/12",
		"192.0.0.0/24",
		"192.0.2.0/24",
		"192.88.99.0/24",
		"192.168.0.0/16",
		"198.18.0.0/15",
		"198.51.100.0/24",
		"203.0.113.0/24",
		"224.0.0.0/4",
		"240.0.0.0/4",
	)
	blockedIPv6Prefixes = mustPrefixes(
		"::/128",
		"64:ff9b:1::/48",
		"100::/64",
		"2001::/23",
		"2001:db8::/32",
		"2002::/16",
		"3fff::/20",
		"fc00::/7",
		"fe80::/10",
		"fec0::/10",
		"ff00::/8",
	)
	loopbackIPv4Prefix = netip.MustParsePrefix("127.0.0.0/8")
	loopbackIPv6Prefix = netip.MustParsePrefix("::1/128")
)

func mustPrefixes(cidrs ...string) []netip.Prefix {
	out := make([]netip.Prefix, 0, len(cidrs))
	for _, cidr := range cidrs {
		out = append(out, netip.MustParsePrefix(cidr))
	}
	return out
}

func isBlockedIPv4Addr(addr netip.Addr) bool {
	if loopbackIPv4Prefix.Contains(addr) || addr == netip.IPv4Unspecified() {
		return !allowTestHTTP()
	}
	for _, p := range blockedIPv4Prefixes {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

// embeddedIPv4 extracts an IPv4 address from ::/96 (IPv4-compatible),
// ::ffff:0/96 (IPv4-mapped) or 64:ff9b::/96 (NAT64 well-known prefix).
func embeddedIPv4(addr netip.Addr) (netip.Addr, bool) {
	if !addr.Is6() {
		return netip.Addr{}, false
	}
	b := addr.As16()
	isCompat := b[0] == 0 && b[1] == 0 && b[2] == 0 && b[3] == 0 && b[4] == 0 && b[5] == 0 && b[6] == 0 && b[7] == 0 && b[8] == 0 && b[9] == 0 && b[10] == 0 && b[11] == 0
	isMapped := b[0] == 0 && b[1] == 0 && b[2] == 0 && b[3] == 0 && b[4] == 0 && b[5] == 0 && b[6] == 0 && b[7] == 0 && b[8] == 0 && b[9] == 0 && b[10] == 0xff && b[11] == 0xff
	isNAT64 := b[0] == 0x00 && b[1] == 0x64 && b[2] == 0xff && b[3] == 0x9b && b[4] == 0 && b[5] == 0 && b[6] == 0 && b[7] == 0 && b[8] == 0 && b[9] == 0 && b[10] == 0 && b[11] == 0
	if !isCompat && !isMapped && !isNAT64 {
		return netip.Addr{}, false
	}
	return netip.AddrFrom4([4]byte{b[12], b[13], b[14], b[15]}), true
}

// Retain the original catalog's HTTPS/public-host, redirect, byte and timeout boundaries.
func blocked(host string) bool {
	host = strings.TrimRight(strings.Trim(strings.TrimSpace(host), "[]"), ".")
	host = strings.ToLower(host)
	if host == "" {
		return true
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		if addr.Zone() != "" {
			return true
		}
		if addr.Is4() {
			return isBlockedIPv4Addr(addr)
		}
		if addr.Is4In6() {
			return isBlockedIPv4Addr(addr.Unmap())
		}
		if v4, ok := embeddedIPv4(addr); ok {
			return isBlockedIPv4Addr(v4)
		}
		if loopbackIPv6Prefix.Contains(addr) {
			return !allowTestHTTP()
		}
		for _, p := range blockedIPv6Prefixes {
			if p.Contains(addr) {
				return true
			}
		}
		return false
	}
	if strings.Contains(host, ":") {
		return true
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") {
		return !allowTestHTTP()
	}
	return false
}
func URL(raw string) (*url.URL, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || len(trimmed) > 2000 {
		return nil, errors.New(storelocale.T(storelocale.KeyStoreWallpapersTheWallpaperAddressMustBeAPublicHttps))
	}
	u, e := url.Parse(trimmed)
	schemeOK := e == nil && (u.Scheme == "https" || (u.Scheme == "http" && allowTestHTTP()))
	if !schemeOK || u.User != nil || blocked(u.Hostname()) {
		return nil, errors.New(storelocale.T(storelocale.KeyStoreWallpapersTheWallpaperAddressMustBeAPublicHttps))
	}
	return u, nil
}
func resolveAllowedAddrs(ctx context.Context, hostname string) ([]netip.Addr, error) {
	if blocked(hostname) {
		return nil, errors.New(storelocale.T(storelocale.KeyStoreWallpapersTheWallpaperAddressResolvesToADisallow))
	}
	clean := strings.TrimRight(strings.Trim(strings.TrimSpace(hostname), "[]"), ".")
	if addr, err := netip.ParseAddr(clean); err == nil {
		return []netip.Addr{addr}, nil
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, clean)
	if err != nil {
		return nil, err
	}
	addrs := make([]netip.Addr, 0, len(ips))
	for _, ip := range ips {
		if blocked(ip.IP.String()) {
			return nil, errors.New(storelocale.T(storelocale.KeyStoreWallpapersTheWallpaperAddressResolvesToADisallow))
		}
		if addr, ok := netip.AddrFromSlice(ip.IP); ok {
			addrs = append(addrs, addr)
		}
	}
	if len(addrs) == 0 {
		return nil, errors.New(storelocale.T(storelocale.KeyStoreWallpapersTheWallpaperAddressResolvesToADisallow))
	}
	return addrs, nil
}
func allowed(ctx context.Context, u *url.URL) error {
	if _, err := URL(u.String()); err != nil {
		return err
	}
	_, err := resolveAllowedAddrs(ctx, u.Hostname())
	return err
}
func Fetch(ctx context.Context, raw string, image bool) ([]byte, error) {
	u, err := URL(raw)
	if err != nil {
		return nil, err
	}
	if err = allowed(ctx, u); err != nil {
		return nil, err
	}
	limit := int64(4000000)
	timeout := 30 * time.Second
	if image {
		limit = 12 * 1024 * 1024
		timeout = 20 * time.Second
	}
	dialer := &net.Dialer{Timeout: timeout}
	transport := &http.Transport{
		DialContext: func(dialCtx context.Context, network, addr string) (net.Conn, error) {
			host, port, splitErr := net.SplitHostPort(addr)
			if splitErr != nil {
				return nil, splitErr
			}
			addrs, resolveErr := resolveAllowedAddrs(dialCtx, host)
			if resolveErr != nil {
				return nil, resolveErr
			}
			var lastErr error
			for _, resolved := range addrs {
				conn, dialErr := dialer.DialContext(dialCtx, network, net.JoinHostPort(resolved.String(), port))
				if dialErr == nil {
					return conn, nil
				}
				lastErr = dialErr
			}
			return nil, lastErr
		},
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Timeout: timeout, Transport: transport, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 4 {
			return errors.New(storelocale.T(storelocale.KeyStoreWallpapersTooManyWallpaperRedirects))
		}
		return allowed(req.Context(), req.URL)
	}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Deepseek-Harness-Desktop")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New(fmt.Sprintf(storelocale.T(storelocale.KeyStoreWallpapersWallpaperRequestFailedD), resp.StatusCode))
	}
	if image {
		mime := strings.ToLower(strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0]))
		switch mime {
		case "image/png", "image/jpeg", "image/jpg", "image/webp", "image/gif":
		default:
			return nil, errors.New(storelocale.T(storelocale.KeyStoreWallpapersTheWallpaperIsNotAUsableImage))
		}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err == nil && int64(len(data)) > limit {
		err = errors.New(storelocale.T(storelocale.KeyStoreWallpapersTheWallpaperFileIsTooLarge))
	}
	return data, err
}
func resolve(base, relative string) string {
	b, e := url.Parse(base)
	if e != nil {
		return ""
	}
	u, e := url.Parse(relative)
	if e != nil || relative == "" {
		return ""
	}
	v := b.ResolveReference(u).String()
	if _, e = URL(v); e != nil {
		return ""
	}
	return v
}
func List(ctx context.Context, q Query) Result {
	page := max(1, q.Page)
	var urls []string
	archive := false
	switch q.Source.Kind {
	case "bing":
		if q.Year > 0 {
			archive = true
			urls = []string{fmt.Sprintf("https://bing.npanuhin.me/CN-zh.%d.json", q.Year)}
		} else {
			urls = []string{"https://cn.bing.com/HPImageArchive.aspx?format=js&idx=0&n=8&mkt=zh-CN", "https://cn.bing.com/HPImageArchive.aspx?format=js&idx=8&n=8&mkt=zh-CN"}
		}
	case "wallhaven":
		categories := q.Categories
		if categories != "010" && categories != "001" {
			categories = "100"
		}
		v := url.Values{"purity": {"100"}, "categories": {categories}, "sorting": {"toplist"}, "atleast": {"1920x1080"}, "page": {strconv.Itoa(page)}}
		if q.Search != "" {
			v.Set("q", q.Search)
		}
		urls = []string{"https://wallhaven.cc/api/v1/search?" + v.Encode()}
	case "catalog":
		urls = []string{q.Source.URL}
	default:
		return Result{Warning: storelocale.T(storelocale.KeyStoreWallpapersChooseAGallerySource)}
	}
	result := Result{}
	seen := map[string]bool{}
	for _, u := range urls {
		data, err := Fetch(ctx, u, false)
		if err != nil {
			result.Warning += err.Error() + "；"
			continue
		}
		var rows []map[string]json.RawMessage
		var body struct {
			Images []map[string]json.RawMessage `json:"images"`
			Items  []map[string]json.RawMessage `json:"items"`
			Data   []map[string]json.RawMessage `json:"data"`
			Meta   struct {
				Current int `json:"current_page"`
				Last    int `json:"last_page"`
			} `json:"meta"`
		}
		kind := q.Source.Kind
		if archive {
			err = json.Unmarshal(data, &rows)
		} else {
			err = json.Unmarshal(data, &body)
			if body.Images != nil {
				rows = body.Images
				kind = "bing"
			} else if body.Items != nil {
				rows = body.Items
				kind = "catalog"
			} else if body.Data != nil && kind == "wallhaven" {
				rows = body.Data
				if body.Meta.Current < body.Meta.Last {
					result.NextPage = body.Meta.Current + 1
				}
			} else if err == nil {
				err = errors.New(storelocale.T(storelocale.KeyStoreWallpapersNotAWallpaperDirectory))
			}
		}
		if err != nil {
			result.Warning += err.Error() + "；"
			continue
		}
		for _, r := range rows[:min(500, len(rows))] {
			s := func(key string) string { var v string; _ = json.Unmarshal(r[key], &v); return v }
			item := Item{SourceID: q.Source.ID, Title: s("title"), Copyright: s("copyright")}
			switch {
			case archive:
				item.ID = "bing-" + s("date")
				item.ImageURL = resolve(u, s("bing_url"))
				if item.ImageURL == "" {
					item.ImageURL = resolve(u, s("url"))
				}
				item.ThumbURL = item.ImageURL
			case kind == "bing":
				if string(r["wp"]) == "false" {
					continue
				}
				base := s("urlbase")
				if base != "" {
					item.ImageURL = resolve(u, base+"_1920x1080.jpg")
					item.ThumbURL = resolve(u, base+"_400x240.jpg")
				} else {
					item.ImageURL = resolve(u, s("url"))
					item.ThumbURL = item.ImageURL
				}
				item.ID = s("hsh")
				if item.ID == "" {
					item.ID = item.ImageURL
				}
				if item.Title == "" {
					item.Title = item.Copyright
				}
			case kind == "wallhaven":
				id := s("id")
				item.ID = "wallhaven-" + id
				item.Title = id
				item.ImageURL = resolve(u, s("path"))
				var thumbs map[string]string
				_ = json.Unmarshal(r["thumbs"], &thumbs)
				item.ThumbURL = resolve(u, thumbs["large"])
				if item.ThumbURL == "" {
					item.ThumbURL = resolve(u, thumbs["small"])
				}
			default:
				item.ID = s("id")
				item.ImageURL = resolve(u, s("imageUrl"))
				item.ThumbURL = resolve(u, s("thumbUrl"))
			}
			if item.ID == "" || item.ImageURL == "" || item.ThumbURL == "" || seen[item.ID] {
				continue
			}
			seen[item.ID] = true
			if q.Source.Kind != "wallhaven" && q.Search != "" && !strings.Contains(strings.ToLower(item.Title+" "+item.Copyright+" "+item.ID), strings.ToLower(q.Search)) {
				continue
			}
			result.Items = append(result.Items, item)
		}
	}
	return result
}
