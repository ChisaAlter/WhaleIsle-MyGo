package session

import (
	"reflect"
	"testing"
)

func TestImageRefsAllImagesOnSameLineAndCodeExamples(t *testing.T) {
	body := "![one](./a_(1).png) ![two](<C:/My pics/b (2).png>)\n\n`![example](example.png)`\n\n```md\n[image] attachment:code\n```\n\n[image] attachment:real"
	want := []string{"./a_(1).png", "C:/My pics/b (2).png", "attachment:real"}
	if got := ImageRefs(body); !reflect.DeepEqual(got, want) {
		t.Fatalf("refs=%q", got)
	}
}

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
