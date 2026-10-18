package pet

import "testing"

func TestDroppedFileMetadataBoundary(t *testing.T) {
	name, ok := droppedFileName("whale-pet://localhost/feed-drop?name=%E5%B0%8F%E9%B1%BC.txt")
	if !ok || name != "小鱼.txt" {
		t.Fatalf("name=%q ok=%v", name, ok)
	}
	for _, raw := range []string{
		"https://localhost/feed-drop?name=a.txt", "whale-pet://elsewhere/feed-drop?name=a.txt",
		"whale-pet://localhost/feed-drop?name=..%2Fsecret", "whale-pet://localhost/feed-drop?name=C%3A%5Csecret",
		"whale-pet://localhost/feed-drop?name=a&name=b", "whale-pet://localhost/feed-drop?name=a&path=b",
		"whale-pet://localhost/feed-drop?name=", "whale-pet://localhost/feed-drop?name=a%00b",
	} {
		if _, ok := droppedFileName(raw); ok {
			t.Fatalf("accepted %q", raw)
		}
	}
}
