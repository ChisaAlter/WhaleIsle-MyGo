package files

import "testing"

func TestB24MarkdownMetadata(t *testing.T) {
	source, body, ok := splitMarkdownMetadata("\ufeff---\r\ntitle: 'Hello'\r\nweight: 1.0\r\nsummary: >\r\n  first\r\n  second\r\ntags: [one, two]\r\n...\r\n# Body")
	if !ok || body != "# Body" {
		t.Fatalf("split %q %v", body, ok)
	}
	fields, valid := parseMetadataFields(source)
	if !valid || len(fields) != 4 || fields[0].value != "Hello" || fields[1].value != "1.0" || fields[2].value != "first second" || fields[3].value != "[one, two]" {
		t.Fatalf("fields %#v %v", fields, valid)
	}
	for _, source := range []string{"- one\n- two", "bad: [", "key: one\nkey: two"} {
		if _, valid := parseMetadataFields(source); valid {
			t.Fatalf("should show verbatim: %q", source)
		}
	}
	for _, text := range []string{"prefix\n---\na: b\n---", "---\na: b"} {
		if _, body, ok := splitMarkdownMetadata(text); ok || body != text {
			t.Fatal("non-frontmatter changed")
		}
	}
	if fields, ok := parseMetadataFields(""); !ok || len(fields) != 0 {
		t.Fatal("empty metadata")
	}
}
