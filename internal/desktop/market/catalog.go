package market

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const CatalogURL = "https://awesome-dsh-plugin.com/plugins.json"
const catalogLimit = 8 * 1024 * 1024

//go:embed catalog_snapshot.json
var catalogSnapshot []byte

type CatalogItem struct {
	Screenshots                                                                                    []string
	ID, Owner, Repo, Description, PackageName, Homepage, InstallSpec, Category, Added, Replacement string
	Stars                                                                                          float64
	Deprecated                                                                                     bool
}
type Category struct {
	ID, Label string
	Count     int
}
type Catalog struct {
	Items                    []CatalogItem
	Categories               []Category
	Source, Warning, Updated string
	FetchedAt                time.Time
}
type registry struct {
	Updated    string                       `json:"updated"`
	Categories map[string]map[string]string `json:"categories"`
	Plugins    []json.RawMessage            `json:"plugins"`
}
type registryRow struct {
	Screenshots                                json.RawMessage
	Owner, Name, URL, Category, Added, Install string
	Description                                map[string]string
	NPM                                        *string
	Stars                                      *float64
	Deprecated                                 bool
	Replacement                                json.RawMessage
}
type diskCatalog struct {
	Version   int             `json:"version"`
	FetchedAt int64           `json:"fetchedAt"`
	Registry  json.RawMessage `json:"registry"`
}
type Service struct {
	mu        sync.Mutex
	fetchMu   sync.Mutex
	client    *http.Client
	cachePath string
	raw       []byte
	fetched   time.Time
	known     map[string]CatalogItem
	details   map[string]detailCache
	pending   map[string]*detailPending
}

func NewService(cachePath string) *Service { return NewServiceWithClient(cachePath, nil) }
func NewServiceWithClient(cachePath string, client *http.Client) *Service {
	c := http.Client{}
	if client != nil {
		c = *client
	}
	c.Jar = nil
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return errors.New("marketplace redirects are not allowed") }
	return &Service{client: &c, cachePath: cachePath, known: map[string]CatalogItem{}, details: map[string]detailCache{}, pending: map[string]*detailPending{}}
}
func publicBody(ctx context.Context, c *http.Client, address, accept string, limit int64) ([]byte, error) {
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if e != nil {
		return nil, e
	}
	req.Header.Set("User-Agent", "WhaleIsle-Native-Market")
	req.Header.Set("Accept", accept)
	res, e := c.Do(req)
	if e != nil {
		return nil, e
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d", res.StatusCode)
	}
	if res.ContentLength > limit {
		return nil, errors.New("marketplace response exceeds size limit")
	}
	b, e := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if e != nil {
		return nil, e
	}
	if int64(len(b)) > limit {
		return nil, errors.New("marketplace response exceeds size limit")
	}
	return b, nil
}
func localized(m map[string]string, locale string) string {
	if strings.HasPrefix(strings.ToLower(locale), "zh") {
		if m["zh"] != "" {
			return m["zh"]
		}
	}
	return m["en"]
}
func decodeCatalog(raw []byte, locale string) (Catalog, error) {
	var reg registry
	var out Catalog
	if len(raw) > catalogLimit {
		return out, errors.New("catalog exceeds size limit")
	}
	if e := json.Unmarshal(raw, &reg); e != nil {
		return out, e
	}
	if len(reg.Plugins) == 0 || len(reg.Plugins) > 10000 || len(reg.Categories) > 128 {
		return out, errors.New("invalid catalog structure")
	}
	out.Updated = reg.Updated
	seen := map[string]bool{}
	counts := map[string]int{}
	for _, b := range reg.Plugins {
		var r registryRow
		if e := json.Unmarshal(b, &r); e != nil {
			continue
		}
		id := r.Owner + "/" + r.Name
		if !ghOwner.MatchString(r.Owner) || r.Name == "" || len(id) > 300 || strings.ContainsAny(id, "\r\n\t\x00") {
			continue
		}
		if seen[id] {
			return Catalog{}, errors.New("duplicate catalog identity")
		}
		seen[id] = true
		i := CatalogItem{ID: id, Owner: r.Owner, Repo: r.Name, Homepage: r.URL, Category: r.Category, Added: r.Added, Description: localized(r.Description, locale), Deprecated: r.Deprecated}
		var screenshots []any
		if json.Unmarshal(r.Screenshots, &screenshots) == nil {
			for _, value := range screenshots {
				if image, ok := value.(string); ok {
					i.Screenshots = append(i.Screenshots, image)
					if len(i.Screenshots) == 12 {
						break
					}
				}
			}
		}
		if r.NPM != nil {
			i.PackageName = *r.NPM
		}
		if r.Stars != nil {
			i.Stars = *r.Stars
		}
		_ = json.Unmarshal(r.Replacement, &i.Replacement)
		i.InstallSpec = resolveSpec(i.PackageName, i.Homepage, r.Install)
		if dropped(i.PackageName) || dropped(i.Repo) {
			continue
		}
		if !ValidInstallItem(i) {
			i.InstallSpec = ""
		}
		out.Items = append(out.Items, i)
		counts[i.Category]++
	}
	if len(out.Items) == 0 {
		return out, errors.New("catalog has no readable entries")
	}
	all := "All"
	if strings.HasPrefix(locale, "zh") {
		all = "全部"
	}
	out.Categories = []Category{{ID: "all", Label: all, Count: len(out.Items)}}
	keys := make([]string, 0, len(reg.Categories))
	for k := range reg.Categories {
		keys = append(keys, k)
	}
	for k := range counts {
		if _, ok := reg.Categories[k]; !ok && k != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		label := localized(reg.Categories[k], locale)
		if label == "" {
			label = k
		}
		out.Categories = append(out.Categories, Category{k, label, counts[k]})
	}
	return out, nil
}
func (s *Service) remember(raw []byte, at time.Time, locale, source, warning string) (Catalog, error) {
	cat, e := decodeCatalog(raw, locale)
	if e != nil {
		return cat, e
	}
	cat.Source = source
	cat.FetchedAt = at
	cat.Warning = warning
	s.mu.Lock()
	s.known = map[string]CatalogItem{}
	for _, i := range cat.Items {
		s.known[i.ID] = i
	}
	s.mu.Unlock()
	return cat, nil
}
func (s *Service) disk() ([]byte, time.Time) {
	if s.cachePath == "" {
		return nil, time.Time{}
	}
	f, e := os.Open(s.cachePath)
	if e != nil {
		return nil, time.Time{}
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, catalogLimit+2049))
	if e != nil || len(b) > catalogLimit+2048 {
		return nil, time.Time{}
	}
	var d diskCatalog
	if json.Unmarshal(b, &d) != nil || d.Version != 3 {
		return nil, time.Time{}
	}
	if _, e = decodeCatalog(d.Registry, "en"); e != nil {
		return nil, time.Time{}
	}
	at := time.UnixMilli(d.FetchedAt)
	if at.After(time.Now().Add(time.Minute)) || d.FetchedAt <= 0 {
		return nil, time.Time{}
	}
	return d.Registry, at
}
func (s *Service) save(raw []byte, at time.Time) error {
	if s.cachePath == "" {
		return nil
	}
	if e := os.MkdirAll(filepath.Dir(s.cachePath), 0700); e != nil {
		return e
	}
	b, e := json.Marshal(diskCatalog{3, at.UnixMilli(), raw})
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(s.cachePath), ".market-cache-*")
	if e != nil {
		return e
	}
	name := f.Name()
	defer os.Remove(name)
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(name, s.cachePath)
}
func (s *Service) List(ctx context.Context, force bool, locale string) (Catalog, error) {
	s.fetchMu.Lock()
	defer s.fetchMu.Unlock()
	if e := ctx.Err(); e != nil {
		return Catalog{}, e
	}
	s.mu.Lock()
	raw, at := s.raw, s.fetched
	s.mu.Unlock()
	if len(raw) == 0 {
		raw, at = s.disk()
	}
	if !force && len(raw) > 0 && time.Since(at) < time.Hour {
		return s.remember(raw, at, locale, "cache", "Using a cached catalog (less than one hour old).")
	}
	fetchCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	b, e := publicBody(fetchCtx, s.client, CatalogURL, "application/json", catalogLimit)
	cancel()
	if e == nil {
		_, e = decodeCatalog(b, locale)
	}
	if e == nil {
		now := time.Now()
		s.mu.Lock()
		s.raw = append([]byte(nil), b...)
		s.fetched = now
		s.mu.Unlock()
		warning := ""
		if s.save(b, now) != nil {
			warning = "Catalog loaded; local cache could not be saved."
		}
		return s.remember(b, now, locale, "live", warning)
	}
	if ctx.Err() != nil {
		return Catalog{}, ctx.Err()
	}
	warning := "Catalog refresh failed: " + e.Error()
	if len(raw) > 0 {
		return s.remember(raw, at, locale, "cache", warning+". Using the last available catalog; it may be stale.")
	}
	return s.remember(catalogSnapshot, time.Time{}, locale, "snapshot", warning+". Using the limited offline snapshot dated 2026-08-27; it may be stale.")
}
