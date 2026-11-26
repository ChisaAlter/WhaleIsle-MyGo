package imgcache

import (
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// DefaultMaxBytes is the maximum accepted image body size (8 MiB).
const DefaultMaxBytes int64 = 8 << 20

// DefaultDiskBudget is the default on-disk cache size cap (128 MiB).
const DefaultDiskBudget int64 = 128 << 20

// DefaultMaxEntries is the default in-memory entry map LRU cap.
const DefaultMaxEntries = 512

// Dir returns userData/cache/images.
func Dir(userData string) string {
	return filepath.Join(userData, "cache", "images")
}

// Status is the cache state for one URL.
type Status int

const (
	// StatusMissing means the URL is not cached and no fetch is in flight.
	StatusMissing Status = iota
	// StatusReady means a local file is available.
	StatusReady
	// StatusFetching means a download is in progress.
	StatusFetching
	// StatusError means the last fetch failed (placeholder until cleared).
	StatusError
)

// Cache stores remote images under Dir as sha256(URL) files.
type Cache struct {
	dir        string
	maxBytes   int64
	diskBudget int64
	maxEntries int
	client     *http.Client

	mu      sync.Mutex
	entries map[string]*entry
	lru     list.List
}

type entry struct {
	key     string
	status  Status
	path    string
	err     error
	lruElem *list.Element
}

// New creates a Cache rooted at dir (typically Dir(userData)).
func New(dir string) *Cache {
	return &Cache{
		dir:        dir,
		maxBytes:   DefaultMaxBytes,
		diskBudget: DefaultDiskBudget,
		maxEntries: DefaultMaxEntries,
		client:     newSafeHTTPClient(),
		entries:    make(map[string]*entry),
	}
}

func newSafeHTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil || len(ips) == 0 {
				if err != nil {
					return nil, err
				}
				return nil, fmt.Errorf("imgcache: no IP resolved for %q", host)
			}
			for _, ipAddr := range ips {
				if isBlockedIP(ipAddr.IP) {
					return nil, fmt.Errorf("imgcache: blocked non-public address %s", ipAddr.IP)
				}
			}
			for _, ipAddr := range ips {
				conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ipAddr.String(), port))
				if dialErr == nil {
					return conn, nil
				}
				err = dialErr
			}
			return nil, err
		},
	}
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return checkRedirectSafety(req, via)
		},
	}
}

func isBlockedIP(ip net.IP) bool {
	return ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified()
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func checkRedirectSafety(req *http.Request, via []*http.Request) error {
	if len(via) >= 5 {
		return errors.New("imgcache: too many redirects")
	}
	if req == nil || req.URL == nil || !IsHTTPURL(req.URL.String()) || req.URL.User != nil {
		return errors.New("imgcache: unsafe redirect target")
	}
	host := req.URL.Hostname()
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
			return errors.New("imgcache: redirect to restricted address blocked")
		}
		originLoopback := len(via) > 0 && via[0] != nil && via[0].URL != nil && isLoopbackHost(via[0].URL.Hostname())
		if !originLoopback && (ip.IsLoopback() || ip.IsPrivate()) {
			return errors.New("imgcache: redirect to private/loopback address blocked")
		}
	} else {
		originLoopback := len(via) > 0 && via[0] != nil && via[0].URL != nil && isLoopbackHost(via[0].URL.Hostname())
		if !originLoopback && isLoopbackHost(host) {
			return errors.New("imgcache: redirect to localhost blocked")
		}
	}
	return nil
}

// SetHTTPClient replaces the HTTP client (tests inject httptest clients).
func (c *Cache) SetHTTPClient(client *http.Client) {
	if client == nil {
		return
	}
	c.mu.Lock()
	c.client = client
	c.mu.Unlock()
}

// SetMaxBytes overrides DefaultMaxBytes when n > 0.
func (c *Cache) SetMaxBytes(n int64) {
	if n <= 0 {
		return
	}
	c.mu.Lock()
	c.maxBytes = n
	c.mu.Unlock()
}

// SetDiskBudget overrides DefaultDiskBudget when n > 0.
func (c *Cache) SetDiskBudget(n int64) {
	if n <= 0 {
		return
	}
	c.mu.Lock()
	c.diskBudget = n
	c.mu.Unlock()
}

// SetMaxEntries overrides DefaultMaxEntries when n > 0.
func (c *Cache) SetMaxEntries(n int) {
	if n <= 0 {
		return
	}
	c.mu.Lock()
	c.maxEntries = n
	c.evictEntriesLocked()
	c.mu.Unlock()
}

func (c *Cache) putEntryLocked(rawURL string, e *entry) {
	e.key = rawURL
	if old := c.entries[rawURL]; old != nil && old.lruElem != nil {
		c.lru.Remove(old.lruElem)
		old.lruElem = nil
	}
	e.lruElem = c.lru.PushFront(e)
	c.entries[rawURL] = e
	c.evictEntriesLocked()
}

func (c *Cache) deleteEntryLocked(rawURL string) {
	if old := c.entries[rawURL]; old != nil {
		if old.lruElem != nil {
			c.lru.Remove(old.lruElem)
			old.lruElem = nil
		}
		delete(c.entries, rawURL)
	}
}

func (c *Cache) evictEntriesLocked() {
	limit := c.maxEntries
	if limit <= 0 {
		limit = DefaultMaxEntries
	}
	for len(c.entries) > limit && c.lru.Len() > 0 {
		evicted := false
		for el := c.lru.Back(); el != nil; el = el.Prev() {
			e := el.Value.(*entry)
			if e.status == StatusFetching {
				continue
			}
			c.lru.Remove(el)
			e.lruElem = nil
			delete(c.entries, e.key)
			evicted = true
			break
		}
		if !evicted {
			break
		}
	}
}

func (c *Cache) pruneDisk(keepPath string) {
	c.mu.Lock()
	budget := c.diskBudget
	dir := c.dir
	c.mu.Unlock()
	if budget <= 0 {
		budget = DefaultDiskBudget
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type diskItem struct {
		path    string
		size    int64
		modTime time.Time
	}
	var items []diskItem
	var total int64
	for _, de := range entries {
		if de.IsDir() || strings.HasSuffix(de.Name(), ".tmp") {
			continue
		}
		info, err := de.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		p := filepath.Join(dir, de.Name())
		sz := info.Size()
		total += sz
		items = append(items, diskItem{path: p, size: sz, modTime: info.ModTime()})
	}
	if total <= budget {
		return
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].modTime.Before(items[j].modTime)
	})
	var removed map[string]bool
	for _, it := range items {
		if total <= budget {
			break
		}
		if keepPath != "" && it.path == keepPath {
			continue
		}
		if os.Remove(it.path) == nil {
			total -= it.size
			if removed == nil {
				removed = make(map[string]bool)
			}
			removed[it.path] = true
		}
	}
	if len(removed) > 0 {
		c.mu.Lock()
		for k, e := range c.entries {
			if e.status == StatusReady && removed[e.path] {
				c.deleteEntryLocked(k)
			}
		}
		c.mu.Unlock()
	}
}

// FilePath is the on-disk path for rawURL (sha256 hex of the URL string).
func (c *Cache) FilePath(rawURL string) string {
	sum := sha256.Sum256([]byte(rawURL))
	return filepath.Join(c.dir, hex.EncodeToString(sum[:]))
}

// Lookup returns the local path when the URL is already cached on disk or in memory.
func (c *Cache) Lookup(rawURL string) (string, bool) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || c == nil {
		return "", false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.entries[rawURL]; ok && e.status == StatusReady && e.path != "" {
		if e.lruElem != nil {
			c.lru.MoveToFront(e.lruElem)
		}
		return e.path, true
	}
	path := c.FilePath(rawURL)
	if st, err := os.Stat(path); err == nil && !st.IsDir() && st.Size() > 0 {
		c.putEntryLocked(rawURL, &entry{status: StatusReady, path: path})
		return path, true
	}
	return "", false
}

// State reports cache status for rawURL. path is set when StatusReady.
func (c *Cache) State(rawURL string) (st Status, path string, err error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || c == nil {
		return StatusMissing, "", nil
	}
	if p, ok := c.Lookup(rawURL); ok {
		return StatusReady, p, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.entries[rawURL]; ok {
		return e.status, e.path, e.err
	}
	return StatusMissing, "", nil
}

// Ensure starts an async fetch when the URL is not ready and not already in flight.
// done runs after the attempt finishes (success or error); may be nil.
func (c *Cache) Ensure(rawURL string, done func()) {
	rawURL = strings.TrimSpace(rawURL)
	if c == nil || rawURL == "" || !IsHTTPURL(rawURL) {
		return
	}
	if _, ok := c.Lookup(rawURL); ok {
		return
	}
	c.mu.Lock()
	if e, ok := c.entries[rawURL]; ok {
		if e.status == StatusFetching || e.status == StatusError || e.status == StatusReady {
			c.mu.Unlock()
			return
		}
	}
	c.putEntryLocked(rawURL, &entry{status: StatusFetching})
	c.mu.Unlock()

	go func() {
		_, err := c.Fetch(context.Background(), rawURL)
		c.mu.Lock()
		if err != nil {
			c.putEntryLocked(rawURL, &entry{status: StatusError, err: err})
		}
		c.mu.Unlock()
		if done != nil {
			done()
		}
	}()
}

// Retry is an explicit user refresh, not a repaint retry loop. Ready images and
// an in-flight download stay shared; only a failed entry is released.
func (c *Cache) Retry(rawURL string, done func()) {
	if c == nil {
		return
	}
	c.mu.Lock()
	if e := c.entries[rawURL]; e != nil && e.status == StatusError {
		c.deleteEntryLocked(rawURL)
	}
	c.mu.Unlock()
	c.Ensure(rawURL, done)
}

// Fetch downloads rawURL into the cache directory and returns the local path.
func (c *Cache) Fetch(ctx context.Context, rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if c == nil {
		return "", errors.New("imgcache: nil cache")
	}
	if !IsHTTPURL(rawURL) {
		return "", fmt.Errorf("imgcache: not an http(s) URL")
	}
	if path, ok := c.Lookup(rawURL); ok {
		return path, nil
	}
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return "", err
	}

	c.mu.Lock()
	client := c.client
	maxBytes := c.maxBytes
	c.mu.Unlock()
	if client == nil {
		client = newSafeHTTPClient()
	} else if client.CheckRedirect == nil {
		cloned := *client
		cloned.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return checkRedirectSafety(req, via)
		}
		client = &cloned
	}
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "image/*,*/*;q=0.8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("imgcache: HTTP %d", resp.StatusCode)
	}
	if err := checkImageContentType(resp.Header.Get("Content-Type")); err != nil {
		return "", err
	}
	if resp.ContentLength > maxBytes {
		return "", fmt.Errorf("imgcache: content-length %d exceeds max %d", resp.ContentLength, maxBytes)
	}

	dest := c.FilePath(rawURL)
	f, err := os.CreateTemp(c.dir, filepath.Base(dest)+".*.tmp")
	if err != nil {
		return "", err
	}
	tmp := f.Name()
	cleanupTmp := true
	defer func() {
		if cleanupTmp {
			_ = os.Remove(tmp)
		}
	}()
	n, err := io.Copy(f, io.LimitReader(resp.Body, maxBytes+1))
	closeErr := f.Close()
	if err != nil {
		return "", err
	}
	if closeErr != nil {
		return "", closeErr
	}
	if n > maxBytes {
		return "", fmt.Errorf("imgcache: body exceeds max %d bytes", maxBytes)
	}
	if n == 0 {
		return "", errors.New("imgcache: empty body")
	}
	if err := os.Rename(tmp, dest); err != nil {
		return "", err
	}
	cleanupTmp = false

	c.mu.Lock()
	c.putEntryLocked(rawURL, &entry{status: StatusReady, path: dest})
	c.mu.Unlock()
	c.pruneDisk(dest)
	return dest, nil
}

// IsHTTPURL reports whether raw is an http or https URL.
func IsHTTPURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return u.Host != "" && u.User == nil
	default:
		return false
	}
}

func checkImageContentType(ct string) error {
	ct = strings.TrimSpace(ct)
	if ct == "" {
		return errors.New("imgcache: missing Content-Type")
	}
	media, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return fmt.Errorf("imgcache: Content-Type: %w", err)
	}
	if m := strings.ToLower(media); !strings.HasPrefix(m, "image/") && m != "application/octet-stream" && m != "binary/octet-stream" {
		return fmt.Errorf("imgcache: Content-Type %q is not image/*", media)
	}
	return nil
}
