package imgcache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DefaultMaxBytes is the maximum accepted image body size (8 MiB).
const DefaultMaxBytes int64 = 8 << 20

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
	dir      string
	maxBytes int64
	client   *http.Client

	mu      sync.Mutex
	entries map[string]*entry
}

type entry struct {
	status Status
	path   string
	err    error
}

// New creates a Cache rooted at dir (typically Dir(userData)).
func New(dir string) *Cache {
	return &Cache{
		dir:      dir,
		maxBytes: DefaultMaxBytes,
		client:   &http.Client{Timeout: 30 * time.Second},
		entries:  make(map[string]*entry),
	}
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
		return e.path, true
	}
	path := c.FilePath(rawURL)
	if st, err := os.Stat(path); err == nil && !st.IsDir() && st.Size() > 0 {
		c.entries[rawURL] = &entry{status: StatusReady, path: path}
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
	c.entries[rawURL] = &entry{status: StatusFetching}
	c.mu.Unlock()

	go func() {
		_, err := c.Fetch(context.Background(), rawURL)
		c.mu.Lock()
		if err != nil {
			c.entries[rawURL] = &entry{status: StatusError, err: err}
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
		delete(c.entries, rawURL)
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
		client = &http.Client{Timeout: 30 * time.Second}
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
	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	n, err := io.Copy(f, io.LimitReader(resp.Body, maxBytes+1))
	closeErr := f.Close()
	if err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return "", closeErr
	}
	if n > maxBytes {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("imgcache: body exceeds max %d bytes", maxBytes)
	}
	if n == 0 {
		_ = os.Remove(tmp)
		return "", errors.New("imgcache: empty body")
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}

	c.mu.Lock()
	c.entries[rawURL] = &entry{status: StatusReady, path: dest}
	c.mu.Unlock()
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
		return u.Host != ""
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
