package terminal

import "testing"

func TestTerminalLinkTargetAtPrintedColumn(t *testing.T) {
	for _, tt := range []struct{ text, inside, want string }{
		{"ready http://localhost:3000/a then ./src/main.go:42", "localhost", "http://localhost:3000/a"},
		{"error ./src/main.go:42:7", "main.go", "./src/main.go:42:7"},
		{`error C:\work\src\main.go:21:4`, "main.go", `C:\work\src\main.go:21:4`},
		{"ftp://example.test/src/a.go", "a.go", ""},
		{"http:/private/file.txt", "file.txt", ""},
	} {
		cols := make([]int, len([]rune(tt.text)))
		for i := range cols {
			cols[i] = i
		}
		at := -1
		for i := 0; i+len(tt.inside) <= len(tt.text); i++ {
			if tt.text[i:i+len(tt.inside)] == tt.inside {
				at = i
				break
			}
		}
		if got := TargetAt(tt.text, cols, at); got != tt.want {
			t.Fatalf("%q @ %d = %q, want %q", tt.text, at, got, tt.want)
		}
	}
	// The wide prefix occupies two cells, not two bytes. The hit map is the SDK's.
	if got := TargetAt("中 ./a.go:7", []int{0, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 6); got != "./a.go:7" {
		t.Fatal(got)
	}
}
