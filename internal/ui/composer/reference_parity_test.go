package composer

import "testing"

func TestReferenceQuotedToken(t *testing.T) {
	q, tok, ok := MentionToken(`see @"my folder/next`)
	if !ok || q != "my folder/next" || tok.Start != 4 {
		t.Fatalf("quoted token = %q %+v %v", q, tok, ok)
	}
}
func TestReferenceCaretAndFormatting(t *testing.T) {
	draft := "前缀 @src/name tail"
	caret := len("前缀 @src/na")
	q, tok, ok := MentionTokenAt(draft, caret)
	if !ok || q != "src/na" || ReplaceToken(draft, tok, "@other ") != "前缀 @other me tail" {
		t.Fatalf("caret %q %+v %v", q, tok, ok)
	}
	for _, tc := range []struct {
		path        string
		dir, quoted bool
		want        string
		ok          bool
	}{
		{"my file.txt", false, false, `@"my file.txt"`, true}, {"my folder", true, false, `@"my folder/`, true},
		{"plain", false, true, `@"plain"`, true}, {"dir/", true, false, "@dir/", true},
		{`bad"quote`, false, false, "", false}, {"bad\npath", false, false, "", false},
	} {
		got, ok := FormatFileMention(tc.path, tc.dir, tc.quoted)
		if got != tc.want || ok != tc.ok {
			t.Errorf("format %q = %q %v", tc.path, got, ok)
		}
	}
	for _, draft := range []string{`a@b`, `@"completed"`, `@[same title](dsh-session:ImEi)`, "@ended text"} {
		if _, _, ok := MentionToken(draft); ok {
			t.Errorf("unexpected active token %q", draft)
		}
	}
}
