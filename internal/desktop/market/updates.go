package market

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

var stableVersion = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// CompareStableVersions deliberately leaves prereleases, tags, and incomplete
// versions unknown rather than guessing that a different version is newer.
func CompareStableVersions(current, latest string) (int, bool) {
	if !stableVersion.MatchString(current) || !stableVersion.MatchString(latest) {
		return 0, false
	}
	a, b := strings.Split(current, "."), strings.Split(latest, ".")
	result := 0
	for i := range a {
		x, e := strconv.ParseUint(a[i], 10, 64)
		y, f := strconv.ParseUint(b[i], 10, 64)
		if e != nil || f != nil {
			return 0, false
		}
		if result == 0 {
			if x < y {
				result = -1
			}
			if x > y {
				result = 1
			}
		}
	}
	return result, true
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
		return "", errors.New("No comparable stable version")
	}
	return metadata.Version, nil
}
