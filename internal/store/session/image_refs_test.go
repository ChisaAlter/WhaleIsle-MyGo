package session

import "testing"

func TestImageRefs(t *testing.T) {
	body := "[image] C:\\tmp\\a.png\nhello\n![x](assets/b.jpg)"
	refs := ImageRefs(body)
	if len(refs) != 2 {
		t.Fatalf("refs = %v", refs)
	}
	if BodyWithoutImagePlaceholders(body) != "hello\n![x](assets/b.jpg)" {
		t.Fatalf("body = %q", BodyWithoutImagePlaceholders(body))
	}
}
