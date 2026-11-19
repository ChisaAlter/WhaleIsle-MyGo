package i18n

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// copySurfaces are the packages that paint settings, sidebar, usage and market
// copy. The first sweep walked chat and the sidebar only, so a page in settings
// could grow a gateway string, a filesystem path or a raw environment variable
// and nothing failed.
var copySurfaces = []string{"../settings", "../sidebar", "../usage", "../market", "../../../cmd/whaleisle"}

// renderedCall matches the calls that put a string in front of a person: a body
// of text, a control label, a help string, or a catalog entry.
var renderedCall = regexp.MustCompile(`ui\.(?:Text|RichText|Button|ButtonBase)\(|\.(?:Label|Description|Placeholder|Tooltip)\(|(?:providerCopy|marketText|usageText|voiceText|settingsText|directoryPickerText|lifecycleText)\(|i18n\.Register\(|Register\(`)

// rawInternal is a string the transport, the environment or the file system
// wrote. None of it is copy: a gateway envelope, an internal variable, a status
// code, or a path on this machine. Each one has a sentence that says what
// happened, and the raw text belongs in the log or behind a details disclosure.
var rawInternal = regexp.MustCompile(`gateway/|\$DSH_HOME|\$cwd\b|\bHTTP_\d{3}\b|"?[A-Za-z]:\\\\|/(?:Users|home)/|\.jsonl\b|\b[A-Z][A-Z0-9]*(?:_[A-Z0-9]+){2,}\b`)

// allowance is the one way a raw string may sit on a painted line: the trailing
// comment says why, and a details-only string says so in its own word.
var allowance = regexp.MustCompile(`//\s*(raw-ok|detail-ok):\s*\S`)

func allowed(line string) bool { return allowance.MatchString(line) }

func copySourceFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		files = append(files, filepath.Join(dir, name))
	}
	return files
}

func TestRenderedCopyCarriesNoRawInternals(t *testing.T) {
	scanned := 0
	for _, dir := range copySurfaces {
		for _, path := range copySourceFiles(t, dir) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			scanned++
			for index, line := range strings.Split(string(source), "\n") {
				if !renderedCall.MatchString(line) || !rawInternal.MatchString(line) || allowed(line) {
					continue
				}
				t.Errorf("%s:%d paints an internal string: %s\n  write the sentence a person acts on (zh and en), keep the raw text for the log, or put it behind a details disclosure and mark the line // detail-ok: <where it shows>",
					path, index+1, strings.TrimSpace(line))
			}
		}
	}
	if scanned == 0 {
		t.Fatal("scanned no source files")
	}
}

// visibleText matches the calls whose literal is painted as the sentence itself,
// not a control's accessible name. The shell's accessible names are stable
// English handles and a separate sweep; the words a person reads are not.
var visibleText = []*regexp.Regexp{
	regexp.MustCompile(`ui\.(?:Text|Button)\(c,\s*"([^"]{4,})"`),
	regexp.MustCompile(`ui\.Span\{Text: "([^"]{4,})"`),
}

func englishSentence(text string) bool {
	if strings.ContainsFunc(text, func(r rune) bool { return r >= 0x2E80 }) {
		return false
	}
	if !regexp.MustCompile(`[a-z]{2,}`).MatchString(text) || !strings.Contains(text, " ") {
		return false
	}
	if regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)+`).MatchString(text) {
		return false
	}
	words := 0
	for _, word := range strings.Fields(text) {
		if regexp.MustCompile(`[A-Za-z]{2,}`).MatchString(strings.Trim(word, `.,;:!?()"'…·`)) {
			words++
		}
	}
	return words >= 3
}

func TestVisibleCopyIsLocalized(t *testing.T) {
	scanned := 0
	for _, dir := range copySurfaces {
		for _, path := range copySourceFiles(t, dir) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			scanned++
			for index, line := range strings.Split(string(source), "\n") {
				for _, call := range visibleText {
					match := call.FindStringSubmatch(line)
					if match == nil || !englishSentence(match[1]) || allowed(line) {
						continue
					}
					t.Errorf("%s:%d paints English in the Chinese interface: %q\n  wrap the words in the page's copy helper with a zh and an en sentence, or pass a localized key when the text is not this shell's own",
						path, index+1, match[1])
				}
			}
		}
	}
	if scanned == 0 {
		t.Fatal("scanned no source files")
	}
}
