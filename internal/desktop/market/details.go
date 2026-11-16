package market

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"whaleisle/internal/pluginmanager"
)

type Details struct {
	Readme, Version          string
	Requirements             []string
	Partial                  bool
	CapabilityKind           CapabilityKind
	NativeCompatible         bool
	NativeCompatibilityLabel string
	Capabilities             PackageCapabilities
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
	value := Details{
		CapabilityKind:           item.CapabilityKind,
		NativeCompatible:         item.NativeCompatible,
		NativeCompatibilityLabel: item.NativeCompatibilityLabel,
		Capabilities:             item.Capabilities,
	}
	var npmManifest []byte
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
			npmManifest = b
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
	probed := probeDetailsCapabilities(ctx, s.client, o, r, npmManifest)
	if probed.Kind != "" && probed.Kind != CapabilityUnknown {
		value.CapabilityKind = probed.Kind
		value.NativeCompatible = probed.NativeCompatible
		value.NativeCompatibilityLabel = probed.CompatibilityLabel
		value.Capabilities = probed
	} else if value.CapabilityKind == "" {
		inferred := pluginmanager.InferCapabilityFromCatalogMetadata(item.Category, item.ID, item.PackageName, item.Description)
		value.CapabilityKind = inferred.Kind
		value.NativeCompatible = inferred.NativeCompatible
		value.NativeCompatibilityLabel = inferred.CompatibilityLabel
		value.Capabilities = inferred
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

func probeDetailsCapabilities(ctx context.Context, client *http.Client, owner, repo string, npmManifest []byte) PackageCapabilities {
	if len(npmManifest) > 0 {
		if caps := pluginmanager.ClassifyPackageManifestJSON(npmManifest, false, false); caps.Kind != CapabilityUnknown {
			return caps
		}
	}
	if owner == "" || repo == "" || client == nil {
		return PackageCapabilities{}
	}
	var ghManifest []byte
	c1, cancel1 := context.WithTimeout(ctx, 4*time.Second)
	if b, err := publicBody(c1, client, "https://api.github.com/repos/"+owner+"/"+repo+"/contents/package.json", "application/vnd.github.raw+json", 256*1024); err == nil && isJSONObjectBytes(b) {
		ghManifest = b
		if caps := pluginmanager.ClassifyPackageManifestJSON(ghManifest, false, false); caps.Kind != CapabilityUnknown {
			cancel1()
			return caps
		}
	}
	cancel1()

	hasMCP := false
	c2, cancel2 := context.WithTimeout(ctx, 4*time.Second)
	if b, err := publicBody(c2, client, "https://api.github.com/repos/"+owner+"/"+repo+"/contents/.mcp.json", "application/vnd.github.raw+json", 256*1024); err == nil && isJSONObjectBytes(b) {
		hasMCP = true
	}
	cancel2()

	hasSkill := false
	c3, cancel3 := context.WithTimeout(ctx, 4*time.Second)
	if b, err := publicBody(c3, client, "https://api.github.com/repos/"+owner+"/"+repo+"/contents/SKILL.md", "application/vnd.github.raw+json", 256*1024); err == nil {
		trimmed := strings.TrimSpace(string(b))
		if strings.HasPrefix(trimmed, "---") || strings.HasPrefix(trimmed, "#") {
			hasSkill = true
		}
	}
	cancel3()

	if len(ghManifest) == 0 && !hasMCP && !hasSkill {
		return PackageCapabilities{}
	}
	return pluginmanager.ClassifyPackageManifestJSON(ghManifest, hasSkill, hasMCP)
}

func isJSONObjectBytes(raw []byte) bool {
	trimmed := strings.TrimSpace(string(raw))
	if !strings.HasPrefix(trimmed, "{") {
		return false
	}
	var obj map[string]json.RawMessage
	return json.Unmarshal(raw, &obj) == nil
}
