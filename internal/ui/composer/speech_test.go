package composer

import "testing"

func TestAppendTranscript(t *testing.T) {
	draft := "hi"
	AppendTranscript(&draft, " there ")
	if draft != "hi there" {
		t.Fatalf("%q", draft)
	}
}
