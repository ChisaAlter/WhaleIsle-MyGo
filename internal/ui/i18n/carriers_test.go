package i18n

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// carrierScan names a package and the fields, labels and text calls this shell
// paints. A line that matches both a carrier and a raw failure source shows a
// person the transport's own words, which is what internal/errcopy exists to
// prevent.
//
// The 5b surfaces are small and varied: several of them have no shared carrier
// to hang a local guard on, and ten copies of the same walk would be ten places
// to forget. One scan over all of them is the cheaper promise, and it fails for
// any of them that regrows a raw writer.
var carrierScan = []struct {
	dir     string
	carrier *regexp.Regexp
}{
	{"../remote", regexp.MustCompile(`\.Error = |ForwardError|CopyError|CreateError`)},
	{"../terminal", regexp.MustCompile(`renderError|selectionNotice|\.notice = |message = |ui\.Text\(c, shell\.`)},
	{"../diffsurface", regexp.MustCompile(`\.Status|ScanError|CompareError|GenerationError|RepositoryStatusError|errMsg|BranchStatus|SetDetails`)},
	{"../theme", regexp.MustCompile(`\.Status = `)},
	{"../frame", regexp.MustCompile(`NavigationError|closeError|JSONError|\.message = |fail\(`)},
	{"../projects", regexp.MustCompile(`NavigationError|Description:|ui\.Text\(c, `)},
	{"../im", regexp.MustCompile(`copyError|\.err = `)},
	{"../schedule", regexp.MustCompile(`\.status = `)},
	{"../assistant", regexp.MustCompile(`return err\.Error|ReviewText`)},
	{"../widgets/markdown", regexp.MustCompile(`label = `)},
	// The import wizard paints a session row's reason itself, so the launcher
	// owes every row a sentence rather than a string from below.
	{"../launcher", regexp.MustCompile(`label|ui\.Text\(c, |releaseText`)},
	// The importer is the row's model: it classifies a session it cannot copy
	// and hands over a category. Text put on an item here has no reader but
	// the screen, so it must not come from a failure.
	{"../../desktop/importer", regexp.MustCompile(`Label:|Group:|Reason:|Detail:|Message:`)},
	// boot is where a second classifier grew last time, and no CI job runs its
	// tests: the page's status line and its failure pane come from a failed
	// start, so this walk is the only thing watching them.
	{"../boot", regexp.MustCompile(`ui\.Text\(c, |StatusText = |Label\(`)},
	// The chat speaks a turn's failure and the sidebar speaks a Session's title,
	// and both take strings the Host wrote: the same walk keeps a new surface
	// from painting the transport's own text.
	{"../chat", regexp.MustCompile(`ui\.Text\(c, |ui\.Span\{Text:|Label\(`)},
	{"../sidebar", regexp.MustCompile(`ui\.Text\(c, |Label\(|Description\(`)},
	// hoststate is the copy every surface reads while the local service is not
	// ready: the sentence itself is the product, so nothing below it may leak a
	// failure's own words into it.
	{"../../hoststate", regexp.MustCompile(`Line:|Note:|Placeholder:|Action:|ui\.Text\(c, `)},
}

// rawFailure is every field or call that carries a failure's own words. A
// painted line may match it only when the words never reach the screen: .Error()
// is the transport's, and the two import-catalog fields are the shape this scan
// was added for, where a scanner's message became a checkbox's second half.
var rawFailure = regexp.MustCompile(`\.Error\(\)|UnsupportedReason|UnsupportedDetail|\bllm-[a-z0-9._-]+\s*:`)

// verbatimDiagnostics marks the one carrier whose job is to show the failure
// word for word: the progress panel that offers "copy full error". A person
// asking for the log text gets the log text there, and nowhere else.
const verbatimDiagnostics = "verbatim-diagnostics"

func TestPaintedCarriersNeverTakeARawError(t *testing.T) {
	scanned := 0
	for _, scan := range carrierScan {
		entries, err := os.ReadDir(scan.dir)
		if err != nil {
			t.Fatalf("read %s: %v", scan.dir, err)
		}
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			source, err := os.ReadFile(scan.dir + "/" + name)
			if err != nil {
				t.Fatalf("read %s/%s: %v", scan.dir, name, err)
			}
			scanned++
			for index, line := range strings.Split(string(source), "\n") {
				if !rawFailure.MatchString(line) || !scan.carrier.MatchString(line) {
					continue
				}
				if strings.Contains(line, verbatimDiagnostics) {
					continue
				}
				t.Errorf("%s/%s:%d paints a raw error: %s\n  use internal/errcopy.Sentence for the sentence, or translate a classified category, and keep the raw text for the log",
					scan.dir, name, index+1, strings.TrimSpace(line))
			}
		}
	}
	if scanned == 0 {
		t.Fatal("scanned no source files")
	}
}
