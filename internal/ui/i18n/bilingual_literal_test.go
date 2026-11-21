package i18n

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// bilingualLiteral is the "中文 / English" form this tree's copy used before it
// moved to catalogs and locale pairs: a Han span, " / ", then an ASCII word
// inside one string literal. One joined string cannot switch with the locale,
// so the form itself is what this walk forbids.
var bilingualLiteral = regexp.MustCompile(`"[^"\n]*\p{Han}[^"\n]* */ +[A-Za-z]`)

// literalTrees are the roots whose copy must come from a catalog key or a
// ZH/EN pair, never from a joined bilingual string.
var literalTrees = []string{"..", filepath.Join("..", "..", "..", "cmd", "whaleisle")}

// literalCatalog reports whether a file is itself the catalog: locales tables
// legitimately hold "edit 修改 / pause 暂停" style text in their ZH fields, and
// guarding them would ban the mechanism instead of the bypass.
func literalCatalog(name string) bool {
	switch {
	case strings.HasSuffix(name, "_test.go"):
		return true
	case name == "locales.go", strings.HasPrefix(name, "catalog_"):
		return true
	case strings.HasPrefix(name, "locales_"), strings.HasSuffix(name, "_locales.go"):
		return true
	}
	return false
}

func TestNoBilingualLiteralStrings(t *testing.T) {
	scanned := 0
	for _, root := range literalTrees {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "i18n" || strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			name := d.Name()
			if !strings.HasSuffix(name, ".go") || literalCatalog(name) {
				return nil
			}
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			scanned++
			for index, line := range strings.Split(string(source), "\n") {
				if !bilingualLiteral.MatchString(line) {
					continue
				}
				t.Errorf("%s:%d keeps a bilingual 'zh / en' literal: %s\n  move the copy into a catalog entry or a ZH/EN pair so it follows the locale",
					path, index+1, strings.TrimSpace(line))
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
