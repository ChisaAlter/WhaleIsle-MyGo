package openui

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
)

// The embedded viewer is only as good as the checkout it came from. Git
// normalizes line endings on the way in, so a worktree whose asset was rewritten
// after checkout reports a clean `git status` and still fails provenance with the
// same message a substituted asset would give. This compares the bytes on disk
// with the pin and names the line endings when they are the reason.
func TestWhaleViewerAssetMatchesItsPin(t *testing.T) {
	raw, err := os.ReadFile("viewer/v2/viewer.html")
	if err != nil {
		t.Fatal(err)
	}
	if sum := assetDigest(raw); sum != whaleViewerSHA256 {
		if assetDigest(bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))) == whaleViewerSHA256 {
			t.Fatalf("viewer/v2/viewer.html is checked out with CRLF line endings (%s) while the app pins the LF bytes (%s): run `git config core.autocrlf input` in this checkout and re-checkout the file", sum, whaleViewerSHA256)
		}
		t.Fatalf("viewer/v2/viewer.html is %s on disk, pinned %s", sum, whaleViewerSHA256)
	}
}

// The rest of the viewer tree is embedded byte-for-byte too, and a converted
// checkout would hand the WebView different JavaScript than the pinned viewer
// expects. .gitattributes keeps the tree unchecked out as -text; this catches a
// worktree that predates that rule.
func TestWhaleViewerAssetsCarryNoConvertedLineEndings(t *testing.T) {
	for _, name := range []string{
		"viewer/v2/viewer.html",
		"viewer/v2/host.cjs",
		"viewer/v2/native-adapter.js",
		"viewer/v2/assets/echarts-core.js",
		"viewer/v2/assets/echarts-full.js",
		"viewer/v2/assets/icons.js",
		"viewer/v2/assets/mermaid.js",
		"viewer/v2/assets/standalone-runtime.js",
		"viewer/v2/assets/three.js",
	} {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(raw, []byte("\r\n")) {
			t.Fatalf("%s carries CRLF line endings; the viewer tree is embedded byte-for-byte, so re-checkout the file with core.autocrlf disabled", name)
		}
	}
}

func assetDigest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
