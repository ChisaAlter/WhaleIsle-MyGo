// Package failurepolicy holds the check that keeps a settings failure's raw text
// out of the sentence a person reads.
package failurepolicy

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// A settings failure is reported in the product's words: what did not happen and
// what to do next. The store's text — a path, a code, an err.Error() in English —
// goes behind the alert's 详情 disclosure or into the log.
//
// This list is empty on purpose. A site that must show the raw text as its
// sentence is a design decision: raise it with the Captain and change the
// surface instead of adding an exception below.
var allowlist = map[string]string{}

// scannedTrees are the trees the rules cover, each with the rule that reads it.
// A settings surface answers both questions: is the sentence the store's own
// text, and did the cause it classified survive as the detail. The theme editor
// carries statuses a person reads, so its copy is checked for translation.
var scannedTrees = []struct {
	tree string
	rule func(rel, src string) []string
	fix  string
}{
	{
		tree: "internal/ui/settings",
		rule: Violations,
		fix: "Fix: write what failed and what to do next, and pass the raw text to settingsAlert's fifth argument " +
			"or AlertOptions.Detail (internal/ui/widgets/alert.go), or to the boot log.",
	},
	{
		tree: "internal/ui/settings",
		rule: DroppedCause,
		fix: "Fix: keep the error beside the sentence and pass its own text as the last argument, " +
			"so 详情 opens the cause instead of nothing.",
	},
	{
		tree: "internal/ui/theme",
		rule: UntranslatedStatus,
		fix: "Fix: register the sentence with i18n and set it with i18n.T, or refuse the preference through invalid() " +
			"so the words are translated.",
	},
	{
		tree: "internal/ui/custompanel",
		rule: Violations,
		fix: "Fix: draw the sentence the view wrote and pass the cause to AlertOptions.Detail, " +
			"so a refused panel shows what happened and opens its cause.",
	},
	{
		tree: "internal/store/custompanel",
		rule: StoreCopy,
		fix: "Fix: name the problem with a Problem constant and pass the store's own text to refuse(); " +
			"the sentence belongs to internal/ui/custompanel, where it is translated.",
	},
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod above the test's directory")
		}
		dir = parent
	}
}

func TestNoRawFailureSentence(t *testing.T) {
	root := repoRoot(t)
	for _, scan := range scannedTrees {
		var found []string
		walkErr := filepath.Walk(filepath.Join(root, scan.tree), func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			if _, allowed := allowlist[rel]; allowed {
				return nil
			}
			src, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			found = append(found, scan.rule(rel, string(src))...)
			return nil
		})
		if walkErr != nil {
			t.Fatal(walkErr)
		}
		if len(found) == 0 {
			continue
		}
		sort.Strings(found)
		t.Fatalf("%s: %d site(s) show a failure without saying what happened, keeping the cause, or speaking the interface language.\n%s\n%s",
			scan.tree, len(found), scan.fix, strings.Join(found, "\n"))
	}
}

// The rule above is only worth its absence of findings if a violation is caught,
// so each shape is driven into the checker directly.
func TestGuardCatchesEachRawSentenceShape(t *testing.T) {
	const header = "package probe\n"
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "alert sentence is a store field",
			body: "func draw(c *ui.Context, s store.Snapshot) {\n\tsettingsAlert(c, \"probe\", s.Err != \"\", s.Err, \"\")\n}\n",
			want: "settingsAlert sentence is the store's own failure text",
		},
		{
			name: "alert sentence is err.Error()",
			body: "func draw(c *ui.Context, err error) {\n\tsettingsAlert(c, \"probe\", err != nil, err.Error(), \"\")\n}\n",
			want: "settingsAlert sentence is the store's own failure text",
		},
		{
			name: "alert still takes the old three-argument form",
			body: "func draw(c *ui.Context, s store.Snapshot) {\n\tsettingsAlert(c, \"probe\", s.Err != \"\", func() string { return s.Err })\n}\n",
			want: "settingsAlert takes the sentence and the store's text as separate arguments",
		},
		{
			name: "alert message is a store field",
			body: "func draw(c *ui.Context, s store.Snapshot) {\n\twidgets.Alert(c, widgets.AlertOptions{ID: \"probe\", Message: s.Err})\n}\n",
			want: "Alert Message is the store's own failure text",
		},
		{
			name: "text draws a store field",
			body: "func draw(c *ui.Context, s store.Snapshot) {\n\tui.Text(c, s.Err)\n}\n",
			want: "Text draws the store's own failure text as the sentence",
		},
		{
			name: "text draws err.Error()",
			body: "func draw(c *ui.Context, err error) {\n\tui.Text(c, err.Error())\n}\n",
			want: "Text draws the store's own failure text as the sentence",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			found := Violations("probe.go", header+tc.body)
			if len(found) != 1 || !strings.Contains(found[0], tc.want) {
				t.Fatalf("Violations = %q, want one naming %q", found, tc.want)
			}
		})
	}
	for _, clean := range []string{
		"func draw(c *ui.Context, s store.Snapshot) {\n\tsettingsAlert(c, \"probe\", s.Err != \"\", providerCopy(\"读取失败，请重试。\", \"Could not read; try again.\"), s.Err)\n}\n",
		"func draw(c *ui.Context, s store.Snapshot) {\n\twidgets.Alert(c, widgets.AlertOptions{ID: \"probe\", Message: providerCopy(\"读取失败，请重试。\", \"Could not read; try again.\"), Detail: s.Err})\n}\n",
		"func draw(c *ui.Context, sentence, detail string) {\n\tui.Text(c, sentence)\n}\n",
	} {
		if found := Violations("probe.go", header+clean); len(found) != 0 {
			t.Fatalf("clean site was refused: %q\n%s", found, clean)
		}
	}
}

// A classified sentence with nowhere to open is the same defect one step later:
// the person reads what happened but cannot show anyone the cause.
func TestGuardCatchesADroppedCause(t *testing.T) {
	const header = "package probe\n"
	cases := []struct {
		name string
		body string
	}{
		{
			name: "alert sentence classified in place, empty detail",
			body: "func draw(c *ui.Context, err error) {\n\tsentence := errcopy.Sentence(err)\n\tsettingsAlert(c, \"probe\", err != nil, sentence, \"\")\n}\n",
		},
		{
			name: "failure line seat classified into a pointer, empty detail",
			body: "func draw(c *ui.Context, err error) {\n\tfailure := ui.Local(c.Root(), \"probe\", func() string { return \"\" })\n\tif err != nil {\n\t\t*failure = errcopy.Sentence(err)\n\t}\n\tsettingsFailureLine(c, \"probe\", *failure, \"\")\n}\n",
		},
		{
			name: "theme status drawn as the sentence, empty detail",
			body: "func draw(c *ui.Context, e *theme.Editor) {\n\tsettingsAlert(c, \"probe\", e.Status != \"\", e.Status, \"\")\n}\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			found := DroppedCause("probe.go", header+tc.body)
			if len(found) != 1 || !strings.Contains(found[0], "passes nothing to 详情") {
				t.Fatalf("DroppedCause = %q, want one finding naming the empty detail", found)
			}
		})
	}
	for _, clean := range []string{
		"func draw(c *ui.Context, err error) {\n\tsentence := errcopy.Sentence(err)\n\tsettingsAlert(c, \"probe\", err != nil, sentence, err.Error())\n}\n",
		"func draw(c *ui.Context, failure settingsRefusal) {\n\tsettingsAlert(c, \"probe\", failure.failed(), failure.sentence, failure.detail)\n}\n",
		"func draw(c *ui.Context, e *theme.Editor) {\n\tsettingsAlert(c, \"probe\", e.Status != \"\", e.Status, e.StatusDetail)\n}\n",
	} {
		if found := DroppedCause("probe.go", header+clean); len(found) != 0 {
			t.Fatalf("clean site was refused: %q\n%s", found, clean)
		}
	}
}

// A validator's status is the message a person reads, so it takes the same
// route as every other sentence: registered copy, not a literal.
func TestGuardCatchesStatusCopyWrittenInCode(t *testing.T) {
	const header = "package theme\n"
	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "english literal", body: "func (e *Editor) Save() {\n\te.Status = \"Font size must be an integer\"\n}\n"},
		{name: "bilingual literal", body: "func (e *Editor) Save() {\n\te.Status = \"已保存 / Saved\"\n}\n"},
		{name: "chinese literal", body: "func (e *Editor) Save() {\n\te.Status = \"字号需在 8–48 之间\"\n}\n"},
		{name: "formatted literal", body: "func (e *Editor) Save(lo, hi float32) {\n\te.Status = fmt.Sprintf(\"Font size must be an integer from %.0f to %.0f\", lo, hi)\n}\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			found := UntranslatedStatus("probe.go", header+tc.body)
			if len(found) != 1 || !strings.Contains(found[0], "Status is written in code") {
				t.Fatalf("UntranslatedStatus = %q, want one finding", found)
			}
		})
	}
	for _, clean := range []string{
		"func (e *Editor) Save() {\n\te.Status = i18n.T(i18n.KeyThemeEditorSaved)\n}\n",
		"func (e *Editor) Save() {\n\te.Status, e.StatusDetail = i18n.T(i18n.KeyThemeEditorSaved), \"\"\n}\n",
		"func (e *Editor) Save(err error) {\n\te.failStatus(err)\n}\n",
	} {
		if found := UntranslatedStatus("probe.go", header+clean); len(found) != 0 {
			t.Fatalf("clean site was refused: %q\n%s", found, clean)
		}
	}
}

// A store that words its own refusals puts copy in the wrong layer: the
// sentence has to be translated, and the store cannot know the interface
// language. Moving the sentence back in has to be what turns this red.
func TestGuardCatchesStoreCopy(t *testing.T) {
	const header = "package custompanel\n"
	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "errors.New literal", body: "func Validate() error {\n\treturn errors.New(\"Panel title is required\")\n}\n"},
		{name: "fmt.Errorf literal", body: "func Validate() error {\n\treturn fmt.Errorf(\"Panel needs 1–64 widgets\")\n}\n"},
		{name: "fmt.Errorf with a format", body: "func Validate(kind string) error {\n\treturn fmt.Errorf(\"Unsupported widget kind %q\", kind)\n}\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			found := StoreCopy("probe.go", header+tc.body)
			if len(found) != 1 || !strings.Contains(found[0], "words a refusal itself") {
				t.Fatalf("StoreCopy = %q, want one finding", found)
			}
		})
	}
	for _, clean := range []string{
		"func Validate() error {\n\treturn refuse(ProblemMissingTitle, \"panel title is required\")\n}\n",
		"func Validate(err error) error {\n\treturn refuse(ProblemJSON, \"%s\", err)\n}\n",
		"func Load(path string) ([]byte, error) {\n\treturn os.ReadFile(path)\n}\n",
	} {
		if found := StoreCopy("probe.go", header+clean); len(found) != 0 {
			t.Fatalf("clean site was refused: %q\n%s", found, clean)
		}
	}
}

// Every problem the store can name needs a sentence, or a precise refusal
// quietly becomes the generic failure line again.
func TestEveryStoreProblemHasASentence(t *testing.T) {
	root := repoRoot(t)
	read := func(rel string) string {
		t.Helper()
		src, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		return string(src)
	}
	store := read("internal/store/custompanel/refusal.go")
	view := read("internal/ui/custompanel/failure.go")
	if found := UnnamedProblems(store, view); len(found) != 0 {
		t.Fatalf("the panel store can name a problem the view cannot word:\n%s", strings.Join(found, "\n"))
	}
	unnamed := strings.Replace(view, "panel.ProblemFontSize:", "panel.ProblemUnknown:", 1)
	if unnamed == view {
		t.Fatal("the probe did not change the view map")
	}
	if found := UnnamedProblems(store, unnamed); len(found) != 1 || !strings.Contains(found[0], "ProblemFontSize has no sentence") {
		t.Fatalf("a problem without copy was not reported: %q", found)
	}
}

// The cause and the sentence are different jobs. Drawing the cause as the
// message puts the evidence back where the copy belongs.
func TestGuardCatchesTheCauseDrawnAsTheMessage(t *testing.T) {
	const header = "package probe\n"
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{
			name: "alert message is the cause",
			body: "func draw(c *ui.Context, failure Failure) {\n\twidgets.Alert(c, widgets.AlertOptions{ID: \"probe\", Message: failure.Detail})\n}\n",
			want: "Alert Message is the cause",
		},
		{
			name: "alert sentence is the cause",
			body: "func draw(c *ui.Context, failure Failure) {\n\tsettingsAlert(c, \"probe\", failure.failed(), failure.detail, \"\")\n}\n",
			want: "settingsAlert draws the cause as its message",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			found := Violations("probe.go", header+tc.body)
			if len(found) != 1 || !strings.Contains(found[0], tc.want) {
				t.Fatalf("Violations = %q, want one naming %q", found, tc.want)
			}
		})
	}
}
