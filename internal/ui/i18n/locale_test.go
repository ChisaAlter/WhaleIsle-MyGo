package i18n

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// localeComparison finds a copy helper deciding which of its two strings to
// show by comparing the active catalog id to a literal.
var localeComparison = regexp.MustCompile(`i18n\.Locale\(\)\s*[!=]=\s*"([^"]*)"`)

// TestCopyHelpersAskTheCatalogWhichLocaleIsActive pins how a helper asks which
// locale is on: the catalog normalizes to these two ids, so any other literal
// never matches and its branch is dead copy — the English string a Chinese
// interface shows beside its Chinese neighbours.
func TestCopyHelpersAskTheCatalogWhichLocaleIsActive(t *testing.T) {
	canonical := map[string]bool{LocaleZH: true, LocaleEN: true}
	scanned := 0
	for _, root := range []string{"../..", "../../../cmd"} {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if name := entry.Name(); name == "testdata" || name == "third_party" || name == "reference" {
					return fs.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			source, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			scanned++
			for index, line := range strings.Split(string(source), "\n") {
				match := localeComparison.FindStringSubmatch(line)
				if match == nil || canonical[match[1]] {
					continue
				}
				t.Errorf("%s:%d compares the locale to %q, which the catalog never returns: use i18n.LocaleZH or i18n.LocaleEN, or the zh branch is dead copy\n  %s",
					path, index+1, match[1], strings.TrimSpace(line))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
	if scanned == 0 {
		t.Fatal("scanned no source files")
	}
}

// TestCanonicalLocaleIDsAreTheOnesTheCatalogNormalizesTo keeps the guard honest:
// if a catalog id ever changes, this test names the new pair instead of leaving
// every helper comparing to an id that no longer exists.
func TestCanonicalLocaleIDsAreTheOnesTheCatalogNormalizesTo(t *testing.T) {
	for _, locale := range []string{LocaleZH, LocaleEN} {
		if Normalize(locale) != locale {
			t.Fatalf("Normalize(%q) = %q", locale, Normalize(locale))
		}
	}
	for _, alias := range []string{"zh", "zh-CN", "zh_CN", "en-US"} {
		if alias == LocaleZH || alias == LocaleEN {
			continue
		}
		if Normalize(alias) == alias {
			t.Fatalf("Normalize(%q) kept an id no helper may compare to", alias)
		}
	}
}
