package diffsurface

import "testing"

func TestParsePatchFilesPreservesNamesAndMetadata(t *testing.T) {
	text := "diff --git a/old name.txt b/new name.txt\nsimilarity index 80%\nrename from old name.txt\nrename to new name.txt\n--- a/old name.txt\n+++ b/new name.txt\n@@ -1 +1 @@\n-old\n+new\ndiff --git \"a/\\346\\226\\207.txt\" \"b/\\346\\226\\207.txt\"\n--- \"a/\\346\\226\\207.txt\"\n+++ /dev/null\n@@ -1 +0,0 @@\n-gone\ndiff --git a/image.bin b/image.bin\nBinary files a/image.bin and b/image.bin differ\n"
	files := parsePatchFiles(text)
	if len(files) != 3 || files[0].Path != "new name.txt" || files[1].Path != "文.txt" || files[2].Path != "image.bin" {
		t.Fatalf("files %#v", files)
	}
	if files[0].Source == "" || files[2].Source == "" {
		t.Fatal("metadata omitted")
	}
}
func TestSplitPatchRowsAlignUnequalChangesAndContext(t *testing.T) {
	rows := splitPatchRows("@@ -8,4 +8,3 @@\n context\n-old one\n-old two\n+new\n tail\n\\ No newline at end of file")
	if len(rows) != 6 {
		t.Fatalf("rows %#v", rows)
	}
	if rows[1].OldNumber != 8 || rows[1].NewNumber != 8 || rows[2].Old != "-old one" || rows[2].New != "+new" || rows[3].Old != "-old two" || rows[3].New != "" || rows[4].OldNumber != 11 || rows[4].NewNumber != 10 {
		t.Fatalf("alignment %#v", rows)
	}
}
func TestPatchViewFoldAndCacheState(t *testing.T) {
	var state patchViewState
	files := state.update("diff --git a/a b/a\n--- a/a\n+++ b/a\n@@ -1 +1 @@\n-a\n+b")
	if len(files) != 1 || state.Open["a"] {
		t.Fatal("initial file should be collapsed")
	}
	state.setAll(true)
	if !state.Open["a"] {
		t.Fatal("expand did not open file")
	}
	state.update(state.source)
	if !state.Open["a"] {
		t.Fatal("same diff lost open state")
	}
	state.setAll(false)
	if state.Open["a"] {
		t.Fatal("collapse did not close file")
	}
	if safePatchPath("../outside") || safePatchPath("/absolute") || safePatchPath(".git/config") || !safePatchPath("dir/a b.txt") {
		t.Fatal("unsafe open-file path")
	}
}

func TestBinaryRenameKeepsLiteralPathPrefix(t *testing.T) {
	files := parsePatchFiles("diff --git a/a/old.bin b/b/new.bin\nsimilarity index 100%\nrename from a/old.bin\nrename to b/new.bin")
	if len(files) != 1 || files[0].Path != "b/new.bin" {
		t.Fatalf("binary rename path: %#v", files)
	}
}
