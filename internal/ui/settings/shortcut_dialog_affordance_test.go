package settings

import (
	"image"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/shortcuts"
)

// TestShortcutDialogDropsDownLikeTheReference pins the reference dialog's
// translateY(30px): the 600 DIP panel centered in a 920 DIP viewport starts at
// 190, not 160, and its 28 DIP close box keeps the reference's 20 DIP inset.
func TestShortcutDialogDropsDownLikeTheReference(t *testing.T) {
	e := shortcuts.NewEditor(nil, nil, func(map[string]string) error { return nil })
	tt := shortcutDialogFixture(t, e, 1440, 920)

	closeBox, ok := tt.Find(providerCopy("关闭快捷键", "Close keyboard shortcuts"))
	if !ok {
		t.Fatalf("shortcut dialog has no close box: %v", tt.Texts())
	}
	// (920-600)/2 centered + 30 dropped = 190; the close box starts 20 below.
	wantTop := float32((920-600)/2 + shortcutDialogOffsetY + 20)
	if math.Abs(float64(closeBox.Y-wantTop)) > 1.5 {
		t.Fatalf("dialog top is %.1f DIPs, want %.0f (reference translateY(30px))", closeBox.Y-20, wantTop-20)
	}
	// The close box shares the title's center line: 28 DIP tall at 20, title
	// 24 line height at 22.
	if closeBox.H < 27 || closeBox.H > 29 {
		t.Fatalf("close box is %.1f DIPs tall, want 28", closeBox.H)
	}
}

// darkPixels counts the pixels darker than threshold inside r.
func darkPixels(img *image.RGBA, r ui.Rect, threshold uint32) int {
	count := 0
	for y := int(r.Y); y < int(r.Y+r.H); y++ {
		for x := int(r.X); x < int(r.X+r.W); x++ {
			c := img.RGBAAt(x, y)
			if (uint32(c.R)+uint32(c.G)+uint32(c.B))/3 < threshold {
				count++
			}
		}
	}
	return count
}

// TestShortcutDialogRowHoverRevealsEditPencil covers the reference row
// affordance: a 20x26 slot left of the key badge that holds the 12 DIP pencil
// only while the pointer is over the row. The slot stays laid out while hidden,
// so the badge does not move as the pointer enters and leaves the row.
func TestShortcutDialogRowHoverRevealsEditPencil(t *testing.T) {
	e := shortcuts.NewEditor(map[string]string{"session.new": "Mod+N"}, nil, func(map[string]string) error { return nil })
	tt := shortcutDialogFixture(t, e, 1440, 920)

	slot, ok := tt.Find("row-edit-session.new")
	if !ok {
		t.Fatalf("row has no hover action slot: %v", tt.Texts())
	}
	if slot.W != shortcutRowActionSize || slot.H != shortcutBadgeHeight {
		t.Fatalf("hover slot is %.0fx%.0f, want %dx%d", slot.W, slot.H, shortcutRowActionSize, shortcutBadgeHeight)
	}
	badge, ok := tt.Find(shortcutDisplay("Mod+N"))
	if !ok {
		t.Fatalf("row has no key badge: %v", tt.Texts())
	}
	if !onSameLine(slot, badge) {
		t.Fatalf("hover slot at y=%.0f is not on the badge's line y=%.0f", slot.Y, badge.Y)
	}
	// The badge text sits 12 DIP inside its capsule, and the capsule follows the
	// 20 DIP slot with the row's 8 DIP gap.
	if gap := badge.X - (slot.X + slot.W); math.Abs(float64(gap-20)) > 1.5 {
		t.Fatalf("badge text is %.1f DIPs from the hover slot, want 20 (8 gap + 12 padding)", gap)
	}
	if dark := darkPixels(tt.Image(), slot, 150); dark != 0 {
		t.Fatalf("pencil is painted while the row is not hovered (%d dark pixels)", dark)
	}

	label, ok := tt.Find("shortcut-session.new")
	if !ok {
		t.Fatal("row label not found")
	}
	tt.Move(label.X+8, label.Y+label.H/2)
	for range 2 {
		tt.Frame()
	}
	hidden := tt.Image()
	hovered := darkPixels(hidden, slot, 150)
	if hovered == 0 {
		t.Fatal("hovering the row did not reveal the edit pencil")
	}

	// Leaving the row hides the pencil again and leaves the slot where it was.
	tt.Move(label.X+8, label.Y-60)
	for range 2 {
		tt.Frame()
	}
	if dark := darkPixels(tt.Image(), slot, 150); dark != 0 {
		t.Fatalf("pencil stayed visible after the pointer left the row (%d dark pixels)", dark)
	}
	if moved, ok := tt.Find("row-edit-session.new"); !ok || moved.X != slot.X {
		t.Fatalf("hover slot moved while toggling the pencil: %.0f -> %.0f", slot.X, moved.X)
	}
}

func onSameLine(a, b ui.Rect) bool {
	return a.Y+a.H/2 >= b.Y-8 && a.Y+a.H/2 <= b.Y+b.H+8
}

// TestShortcutInlineEditorHasNoCancelControl keeps the reference inline editor:
// 恢复默认, 移除 and the recorder, with no extra cancel button and no visible
// instruction line. Leaving the row or choosing another row disarms it.
func TestShortcutInlineEditorHasNoCancelControl(t *testing.T) {
	writes := []map[string]string{}
	e := shortcuts.NewEditor(map[string]string{"session.new": "Mod+N"}, nil, func(patch map[string]string) error {
		copy := map[string]string{}
		for id, value := range patch {
			copy[id] = value
		}
		writes = append(writes, copy)
		return nil
	})
	tt := shortcutDialogFixture(t, e, 900, 800)

	if err := tt.Click("shortcut-settings.open"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("cancel-recording"); ok {
		t.Fatal("inline editor still offers a cancel control the reference has no place for")
	}
	if tt.HasText("取消") {
		t.Fatalf("inline editor still shows a cancel label: %v", tt.Texts())
	}
	if tt.HasText("松开组合键即保存。") {
		t.Fatal("inline editor still shows a visible instruction line")
	}
	if _, ok := tt.Find("capture-settings.open"); !ok {
		t.Fatalf("inline editor is missing its recorder: %v", tt.Texts())
	}

	// Choosing another row moves the editor there and disarms the old recorder
	// without writing anything.
	if err := tt.Click("shortcut-session.new"); err != nil {
		t.Fatal(err)
	}
	if e.Recording != "session.new" || len(writes) != 0 {
		t.Fatalf("switching rows saved or kept the old recorder: %q %v", e.Recording, writes)
	}

	// Clicking the edited row's label closes the editor, the reference's
	// click-outside gesture in a list that has no dead space.
	if err := tt.Click("shortcut-session.new"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if e.Recording != "" {
		t.Fatalf("closing the editor left its recorder armed: %q", e.Recording)
	}
	if _, ok := tt.Find("capture-session.new"); ok {
		t.Fatal("closing the editor did not restore the row's key badge")
	}
	if len(writes) != 0 {
		t.Fatalf("closing the editor wrote a binding: %v", writes)
	}
}

// TestShortcutDialogCloseBoxClosesTheRowEditorFirst pins the reference's
// closeReference (ui-shortcuts/src/client/Reference.tsx:102-108): with a row
// editor armed the close box closes that editor and leaves the reference open,
// so only the second press dismisses it.
func TestShortcutDialogCloseBoxClosesTheRowEditorFirst(t *testing.T) {
	writes := 0
	e := shortcuts.NewEditor(map[string]string{"session.new": "Mod+N"}, nil, func(map[string]string) error {
		writes++
		return nil
	})
	tt := shortcutDialogFixture(t, e, 900, 800)

	if err := tt.Click("shortcut-settings.open"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("capture-settings.open"); !ok {
		t.Fatalf("row did not arm its recorder: %v", tt.Texts())
	}
	if err := tt.Click(providerCopy("关闭快捷键", "Close keyboard shortcuts")); err != nil {
		t.Fatalf("shortcut dialog has no close box: %v", err)
	}
	for range 2 {
		tt.Frame()
	}
	if !tt.HasText("恢复全部默认") {
		t.Fatal("the close box dismissed the reference instead of closing the row editor")
	}
	if e.Recording != "" {
		t.Fatalf("the close box left the recorder armed: %q", e.Recording)
	}
	if _, ok := tt.Find("capture-settings.open"); ok {
		t.Fatal("the close box left the row editor on screen")
	}
	if _, ok := tt.Find("shortcut-settings.open"); !ok {
		t.Fatalf("the closed row lost its key badge: %v", tt.Texts())
	}

	if err := tt.Click(providerCopy("关闭快捷键", "Close keyboard shortcuts")); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		tt.Frame()
	}
	if tt.HasText("恢复全部默认") {
		t.Fatal("the second press did not dismiss the reference")
	}
	if writes != 0 {
		t.Fatalf("closing the row editor wrote a binding: %d", writes)
	}
}

// TestShortcutDialogEscapeClosesTheRowEditorFirst covers the same reference rule
// for Escape: the reference's Modal calls closeReference, which closes the row
// editor before the dialog.
func TestShortcutDialogEscapeClosesTheRowEditorFirst(t *testing.T) {
	e := shortcuts.NewEditor(map[string]string{"session.new": "Mod+N"}, nil, func(map[string]string) error { return nil })
	tt := shortcutDialogFixture(t, e, 900, 800)

	if err := tt.Click("shortcut-settings.open"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("capture-settings.open"); !ok {
		t.Fatalf("row did not arm its recorder: %v", tt.Texts())
	}
	// The search field keeps the editor armed and leaves Escape to the dialog.
	if err := tt.Click("shortcut-search"); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		tt.Frame()
	}
	tt.Key(0, ui.KeyEscape)
	for range 2 {
		tt.Frame()
	}
	if !tt.HasText("恢复全部默认") {
		t.Fatal("Escape dismissed the reference instead of closing the row editor")
	}
	if e.Recording != "" {
		t.Fatalf("Escape left the recorder armed: %q", e.Recording)
	}
	if _, ok := tt.Find("capture-settings.open"); ok {
		t.Fatal("Escape left the row editor on screen")
	}

	tt.Key(0, ui.KeyEscape)
	for range 2 {
		tt.Frame()
	}
	if tt.HasText("恢复全部默认") {
		t.Fatal("the second Escape did not dismiss the reference")
	}
}

// TestShortcutInlineEditorLabelIsNotATabStop keeps the reference's inert row
// label: the edited row's label is not a second control, so Tab from the search
// field reaches the inline actions instead of stopping on the label.
func TestShortcutInlineEditorLabelIsNotATabStop(t *testing.T) {
	e := shortcuts.NewEditor(map[string]string{"session.new": "Mod+N"}, nil, func(map[string]string) error { return nil })
	tt := shortcutDialogFixture(t, e, 900, 800)

	if err := tt.Click("shortcut-settings.open"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("shortcut-search"); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		tt.Frame()
	}
	reachedActions := false
	for range 6 {
		tt.Key(0, ui.KeyTab)
		for range 2 {
			tt.Frame()
		}
		if tt.Focused("shortcut-settings.open") {
			t.Fatal("the edited row's label is a tab stop the reference does not have")
		}
		if tt.Focused("default-settings.open") {
			reachedActions = true
		}
	}
	if !reachedActions {
		t.Fatal("Tab never reached the row's inline actions")
	}
	if !tt.HasText("恢复默认") {
		t.Fatalf("the row editor is no longer armed: %v", tt.Texts())
	}
}

// TestShortcutDialogArmedRecorderEscapeLeavesTheEditor covers Escape while the
// recorder itself holds the focus: mygo delivers a key the dialog registers
// before the focused element's input handler, so the armed row leaves its editor
// instead of recording Escape as an invalid chord and raising the review/retry
// pair.
func TestShortcutDialogArmedRecorderEscapeLeavesTheEditor(t *testing.T) {
	writes := 0
	e := shortcuts.NewEditor(map[string]string{"session.new": "Mod+N"}, nil, func(map[string]string) error {
		writes++
		return nil
	})
	tt := shortcutDialogFixture(t, e, 900, 800)

	if err := tt.Click("shortcut-settings.open"); err != nil {
		t.Fatal(err)
	}
	if !tt.Focused("capture-settings.open") {
		t.Fatalf("the recorder did not take the focus: %v", tt.Texts())
	}
	tt.Key(0, ui.KeyEscape)
	for range 2 {
		tt.Frame()
	}
	if !tt.HasText("恢复全部默认") {
		t.Fatal("Escape on the recorder dismissed the reference")
	}
	if e.Recording != "" || e.Err != "" {
		t.Fatalf("Escape on the recorder left a recording or an error: %q %q", e.Recording, e.Err)
	}
	if tt.HasText("审阅当前配置") {
		t.Fatal("Escape on the recorder raised the invalid-chord recovery pair")
	}
	if _, ok := tt.Find("capture-settings.open"); ok {
		t.Fatal("Escape on the recorder left the editor armed")
	}
	if writes != 0 {
		t.Fatalf("Escape on the recorder wrote a binding: %d", writes)
	}
}

// TestShortcutDialogMaskPressClosesTheRowEditorFirst covers the reference
// Modal's mask click, which runs the same closeReference.
func TestShortcutDialogMaskPressClosesTheRowEditorFirst(t *testing.T) {
	e := shortcuts.NewEditor(map[string]string{"session.new": "Mod+N"}, nil, func(map[string]string) error { return nil })
	tt := shortcutDialogFixture(t, e, 900, 800)

	if err := tt.Click("shortcut-settings.open"); err != nil {
		t.Fatal(err)
	}
	if _, ok := tt.Find("capture-settings.open"); !ok {
		t.Fatalf("row did not arm its recorder: %v", tt.Texts())
	}
	// A press on the mask, 900x800 with the 480x600 card centered around y=160.
	tt.ClickAt(24, 40)
	for range 2 {
		tt.Frame()
	}
	if !tt.HasText("恢复全部默认") {
		t.Fatal("the mask press dismissed the reference instead of the row editor")
	}
	if e.Recording != "" {
		t.Fatalf("the mask press left the recorder armed: %q", e.Recording)
	}
	if _, ok := tt.Find("capture-settings.open"); ok {
		t.Fatal("the mask press left the row editor on screen")
	}

	// With no row editor armed the same press dismisses the reference.
	tt.ClickAt(24, 40)
	for range 2 {
		tt.Frame()
	}
	if tt.HasText("恢复全部默认") {
		t.Fatal("the mask press did not dismiss the reference")
	}
}

// TestShortcutDialogResetConfirmationKeepsEscape covers the footer gesture: the
// reference's Reset All clears the armed row editor as it opens its
// confirmation (Reference.tsx:206), so Escape still closes that confirmation
// alone.
func TestShortcutDialogResetConfirmationKeepsEscape(t *testing.T) {
	writes := 0
	e := shortcuts.NewEditor(map[string]string{"settings.open": "Mod+Alt+F12"}, nil, func(map[string]string) error {
		writes++
		return nil
	})
	tt := shortcutDialogFixture(t, e, 900, 800)

	if err := tt.Click("shortcut-settings.open"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("reset-shortcuts"); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		tt.Frame()
	}
	if !tt.HasText("恢复全部默认快捷键？") {
		t.Fatalf("reset confirmation did not open: %v", tt.Texts())
	}
	if _, ok := tt.Find("capture-settings.open"); ok {
		t.Fatal("Reset All left the row editor armed under its confirmation")
	}
	tt.Key(0, ui.KeyEscape)
	for range 2 {
		tt.Frame()
	}
	if tt.HasText("恢复全部默认快捷键？") {
		t.Fatal("Escape did not close the reset confirmation")
	}
	if !tt.HasText("1 项已自定义") {
		t.Fatal("Escape closed the reference along with its confirmation")
	}
	if writes != 0 || e.ResetPending {
		t.Fatalf("Escape wrote the reset: %d %v", writes, e.ResetPending)
	}
}

// TestShortcutDialogHoverSnapshot writes the pointer-over-row state for the
// visual comparison, next to the resting dialog render.
func TestShortcutDialogHoverSnapshot(t *testing.T) {
	e := shortcuts.NewEditor(map[string]string{"session.new": "Mod+N"}, nil, func(map[string]string) error { return nil })
	tt := shortcutDialogFixture(t, e, 1440, 920)
	label, ok := tt.Find("shortcut-settings.open")
	if !ok {
		t.Fatalf("row label not found: %v", tt.Texts())
	}
	tt.Move(label.X+8, label.Y+label.H/2)
	for range 12 {
		tt.Frame()
		time.Sleep(16 * time.Millisecond)
	}
	if _, ok := tt.Find("row-edit-settings.open"); !ok {
		t.Fatalf("hover state did not render: %v", tt.Texts())
	}
	saveBodySnapshot(t, tt, "settings-shortcuts-hover-1440.png")
}
