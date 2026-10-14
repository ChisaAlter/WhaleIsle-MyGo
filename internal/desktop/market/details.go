package market

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strings"
	"time"
)

type Details struct {
	Readme, Version string
	Requirements    []string
	Partial         bool
}
type detailCache struct {
	value Details
	at    time.Time
}
type detailPending struct {
	done  chan struct{}
	value Details
	err   error
}

func trim(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}
func (s *Service) Details(ctx context.Context, item CatalogItem, force bool) (Details, error) {
	s.mu.Lock()
	known, ok := s.known[item.ID]
	if !ok || known.InstallSpec != item.InstallSpec {
		s.mu.Unlock()
		return Details{}, errors.New("catalog target has changed; reopen details")
	}
	item = known
	key := item.ID + ":" + item.InstallSpec
	if prior, ok := s.details[key]; ok && !force {
		ttl := 30 * time.Minute
		if prior.value.Partial {
			ttl = 30 * time.Second
		}
		if time.Since(prior.at) < ttl {
			s.mu.Unlock()
			return prior.value, nil
		}
	}
	if p := s.pending[key]; p != nil {
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			return Details{}, ctx.Err()
		case <-p.done:
			return p.value, p.err
		}
	}
	if len(s.pending) >= 4 {
		s.mu.Unlock()
		return Details{}, errors.New("too many detail requests; retry shortly")
	}
	p := &detailPending{done: make(chan struct{})}
	s.pending[key] = p
	s.mu.Unlock()
	value := Details{}
	o, r, _ := sourceRepo(item.Homepage)
	if o != "" {
		c, cancel := context.WithTimeout(ctx, 8*time.Second)
		b, e := publicBody(c, s.client, "https://api.github.com/repos/"+o+"/"+r+"/readme", "application/vnd.github.raw+json", 256*1024)
		cancel()
		if e != nil {
			value.Partial = true
		} else {
			value.Readme = string(b)
		}
	} else {
		value.Partial = true
	}
	if item.InstallSpec == item.PackageName && validNPM(item.PackageName) {
		c, cancel := context.WithTimeout(ctx, 8*time.Second)
		b, e := publicBody(c, s.client, "https://registry.npmjs.org/"+url.PathEscape(item.PackageName)+"/latest", "application/json", 256*1024)
		cancel()
		var m struct {
			Name, Version    string
			Engines          map[string]string
			PeerDependencies map[string]string
		}
		if e == nil {
			e = json.Unmarshal(b, &m)
		}
		if e != nil || m.Name != item.PackageName {
			value.Partial = true
		} else {
			value.Version = trim(m.Version, 100)
			if m.Engines["dsh"] != "" {
				value.Requirements = append(value.Requirements, "dsh: "+trim(m.Engines["dsh"], 256))
			}
			keys := make([]string, 0, len(m.PeerDependencies))
			for k := range m.PeerDependencies {
				if strings.HasPrefix(k, "@deepseek-ai/") {
					keys = append(keys, k)
				}
			}
			sort.Strings(keys)
			if len(keys) > 100 {
				keys = keys[:100]
			}
			for _, k := range keys {
				value.Requirements = append(value.Requirements, trim(k, 214)+": "+trim(m.PeerDependencies[k], 256))
			}
		}
	}
	s.mu.Lock()
	delete(s.pending, key)
	p.value = value
	p.err = ctx.Err()
	if p.err == nil {
		if len(s.details) >= 30 {
			var oldest string
			var at time.Time
			for k, v := range s.details {
				if oldest == "" || v.at.Before(at) {
					oldest, at = k, v.at
				}
			}
			delete(s.details, oldest)
		}
		s.details[key] = detailCache{value, time.Now()}
	}
	close(p.done)
	s.mu.Unlock()
	return value, p.err
}
