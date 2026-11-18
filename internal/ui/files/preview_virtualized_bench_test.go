package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// Warm frames retain the parsed line model while the viewport stays fixed.
// This covers the real file View and headless engine layout, not native input,
// the OS clipboard, disk-read latency or GPU presentation.
func BenchmarkReadOnlyPreviewViewport(b *testing.B) {
	for _, count := range []int{1000, 10000, 100000} {
		b.Run(fmt.Sprintf("lines_%d", count), func(b *testing.B) {
			var builder strings.Builder
			for i := 0; i < count; i++ {
				fmt.Fprintf(&builder, "Line %06d unchanged text\n", i)
			}
			text := builder.String()
			root, path := b.TempDir(), "long.txt"
			if err := os.WriteFile(filepath.Join(root, path), []byte(text), 0600); err != nil {
				b.Fatal(err)
			}
			tab := &FileTab{Path: path, Preview: text, Draft: text}
			tabs := &Tabs{Preferences: &PreviewPreferences{WordWrap: true},
				Workspaces: map[string]*WorkspaceTabs{filepath.Clean(root): {Open: []*FileTab{tab}, Active: path}}}
			var list ui.ListState
			editing, message := false, ""
			m := Model{ResourceOnly: true, Tabs: tabs, WorkspaceRoot: root, SelectedFile: &path,
				Preview: &tab.Preview, EditDraft: &tab.Draft, PreviewErr: &message, Editing: &editing, PreviewList: &list}
			tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 720, 360)
			tt.Frame()
			first, last := list.Visible()
			if first != 0 || last <= first || last-first >= 100 {
				b.Fatalf("fixed viewport did not virtualize source: %d..%d", first, last)
			}
			if tt.HasText(fmt.Sprintf("Line %06d unchanged text", count-1)) {
				b.Fatal("offscreen final line was rendered")
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				tt.Frame()
			}
			b.StopTimer()
			b.ReportMetric(float64(last-first), "visible-rows")
		})
	}
}
