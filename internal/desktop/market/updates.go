package market

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
)

var semanticVersion = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-([0-9A-Za-z.-]+))?(?:\+[0-9A-Za-z.-]+)?$`)

// CompareStableVersions retains its public entry point and now compares complete
// semver, including prereleases. Build metadata does not affect precedence.
func CompareStableVersions(current, latest string) (int, bool) {
	a, b := semanticVersion.FindStringSubmatch(current), semanticVersion.FindStringSubmatch(latest)
	if a == nil || b == nil || !validPrerelease(a[4]) || !validPrerelease(b[4]) {
		return 0, false
	}
	numeric := func(x, y string) int {
		x, y = strings.TrimLeft(x, "0"), strings.TrimLeft(y, "0")
		if len(x) < len(y) {
			return -1
		}
		if len(x) > len(y) {
			return 1
		}
		return strings.Compare(x, y)
	}
	for i := 1; i <= 3; i++ {
		if n := numeric(a[i], b[i]); n != 0 {
			return n, true
		}
	}
	if a[4] == "" || b[4] == "" {
		if a[4] == b[4] {
			return 0, true
		}
		if a[4] == "" {
			return 1, true
		}
		return -1, true
	}
	x, y := strings.Split(a[4], "."), strings.Split(b[4], ".")
	for i := 0; i < max(len(x), len(y)); i++ {
		if i >= len(x) {
			return -1, true
		}
		if i >= len(y) {
			return 1, true
		}
		if x[i] == "" || y[i] == "" {
			return 0, false
		}
		xn, yn := strings.Trim(x[i], "0123456789") == "", strings.Trim(y[i], "0123456789") == ""
		var n int
		if xn && yn {
			n = numeric(x[i], y[i])
		} else if xn != yn {
			if xn {
				n = -1
			} else {
				n = 1
			}
		} else {
			n = strings.Compare(x[i], y[i])
		}
		if n != 0 {
			return n, true
		}
	}
	return 0, true
}

// Match the reviewed catalog value by content, including copied screenshots.
// CatalogItem now contains a slice and cannot use Go's struct comparison.
func sameCatalogItem(a, b CatalogItem) bool {
	return a.ID == b.ID && a.Owner == b.Owner && a.Repo == b.Repo &&
		a.Description == b.Description && a.PackageName == b.PackageName &&
		a.Homepage == b.Homepage && a.InstallSpec == b.InstallSpec &&
		a.Category == b.Category && a.Added == b.Added &&
		a.Replacement == b.Replacement && a.Stars == b.Stars &&
		a.Deprecated == b.Deprecated && slices.Equal(a.Screenshots, b.Screenshots)
}

// LatestVersion is a read-only registry metadata request. README contents and
// Git repository tags are never treated as executable or authoritative versions.
func (s *Service) LatestVersion(ctx context.Context, item CatalogItem) (string, error) {
	s.mu.Lock()
	known, ok := s.known[item.ID]
	s.mu.Unlock()
	if !ok || !sameCatalogItem(known, item) || !ValidInstallItem(known) || known.InstallSpec != known.PackageName || !validNPM(known.PackageName) {
		return "", errors.New("No verified npm update source")
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	b, err := publicBody(ctx, s.client, "https://registry.npmjs.org/"+url.PathEscape(known.PackageName)+"/latest", "application/json", 256*1024)
	if err != nil {
		return "", err
	}
	var metadata struct{ Name, Version string }
	if json.Unmarshal(b, &metadata) != nil || metadata.Name != known.PackageName || len(metadata.Version) > 100 {
		return "", errors.New("Invalid package version metadata")
	}
	if _, ok := CompareStableVersions(metadata.Version, metadata.Version); !ok {
		return "", errors.New("No comparable semantic version")
	}
	return metadata.Version, nil
}

func validPrerelease(value string) bool {
	if value == "" {
		return true
	}
	for _, part := range strings.Split(value, ".") {
		if part == "" || len(part) > 1 && part[0] == '0' && strings.Trim(part, "0123456789") == "" {
			return false
		}
	}
	return true
}
