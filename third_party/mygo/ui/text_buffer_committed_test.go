package ui

import "testing"

func TestCommittedSnapshotPreservesNativeReplacementOriginAcrossFrames(t *testing.T) {
	for _, finish := range []string{"replace", "unmark"} {
		t.Run(finish, func(t *testing.T) {
			buffer := NewTextBuffer("abcd")
			tester := coreNewTester(func(c *context) {
				coreTextAreaBuffer(c, buffer).Fill().AutoFocus()
			}, 300, 140)
			defer tester.rt.close()
			client := tester.NativeTextInputClient()
			if client == nil {
				t.Fatal("native client missing")
			}
			end := TextInputRange{Start: 4, End: 4}
			client.ReplaceText(&end, "!") // committed, still no widget build
			before := buffer.CommittedSnapshot()
			selection := TextInputRange{Start: 1, End: 3}
			client.SetMarkedText(&selection, "中文", TextInputRange{Start: 2, End: 2})
			if buffer.String() != "ad!" || before.String() != "abcd!" || buffer.CommittedSnapshot() != before {
				t.Fatal("preedit deletion escaped committed capture, or lost earlier committed native input")
			}
			tester.Frame()
			client.SetMarkedText(&selection, "候选", TextInputRange{Start: 2, End: 2})
			if buffer.String() != "ad!" || buffer.CommittedSnapshot() != before {
				t.Fatal("explicit-range candidate staging became a false committed snapshot")
			}
			client.SetMarkedText(nil, "中文候选", TextInputRange{Start: 4, End: 4})
			if buffer.CommittedSnapshot() != before {
				t.Fatal("widget build or another preedit update replaced the transaction origin")
			}
			want := "a中文d!"
			if finish == "replace" {
				client.ReplaceText(nil, "中文")
			} else {
				client.UnmarkText()
				want = "a中文候选d!"
			}
			if got := buffer.CommittedSnapshot(); got.String() != want || got != buffer.Snapshot() {
				t.Fatal("real commit retained an obsolete preedit capture")
			}
		})
	}
}

func TestCommittedSnapshotExternalReplacementCancelsOldNativeCapture(t *testing.T) {
	for _, replace := range []string{"set", "restore"} {
		t.Run(replace, func(t *testing.T) {
			buffer := NewTextBuffer("abcd")
			tester := coreNewTester(func(c *context) {
				coreTextAreaBuffer(c, buffer).Fill().AutoFocus()
			}, 300, 140)
			defer tester.rt.close()
			client := tester.NativeTextInputClient()
			if client == nil {
				t.Fatal("native client missing")
			}
			selection := TextInputRange{Start: 1, End: 3}
			client.SetMarkedText(&selection, "中文", TextInputRange{Start: 2, End: 2})
			if replace == "set" {
				buffer.Set("external replacement")
			} else {
				buffer.Restore(NewTextBuffer("external replacement").Snapshot())
			}
			if got := buffer.CommittedSnapshot(); got.String() != "external replacement" || got != buffer.Snapshot() {
				t.Fatal("external replacement retained old IME recovery metadata")
			}
			// Native prepare synchronizes the external replacement before queries;
			// loadBuffer must cancel old composition without overwriting new data.
			if _, marked := client.MarkedRange(); marked {
				t.Fatal("native editor retained replaced preedit")
			}
			client.UnmarkText()
			if buffer.CommittedSnapshot().String() != "external replacement" {
				t.Fatal("old native callback overwrote the external document")
			}
		})
	}
}

func TestCommittedSnapshotOtherEditorCannotClearCurrentPreeditOwner(t *testing.T) {
	buffer := NewTextBuffer("abcd")
	focusSecond := false
	tester := coreNewTester(func(c *context) {
		coreColumn(c).Fill().Children(func() {
			coreTextAreaBuffer(c, buffer).Height(70).AutoFocus()
			second := coreTextAreaBuffer(c, buffer).Height(70)
			if focusSecond {
				second.Focus()
			}
		})
	}, 300, 160)
	defer tester.rt.close()
	old := tester.NativeTextInputClient()
	if old == nil {
		t.Fatal("first native owner missing")
	}
	focusSecond = true
	tester.Frame()
	current := tester.NativeTextInputClient()
	if current == nil {
		t.Fatal("replacement native owner missing")
	}
	selection := TextInputRange{Start: 1, End: 3}
	current.SetMarkedText(&selection, "中文", TextInputRange{Start: 2, End: 2})
	before := buffer.CommittedSnapshot()
	tester.Frame() // first editor also syncs the shared buffer here
	old.UnmarkText()
	if before.String() != "abcd" || buffer.CommittedSnapshot() != before {
		t.Fatal("inactive editor synchronization/callback cleared the current transaction origin")
	}
	current.ReplaceText(nil, "中文")
	if buffer.CommittedSnapshot().String() != "a中文d" {
		t.Fatal("current owner could not commit after stale callback")
	}
}
