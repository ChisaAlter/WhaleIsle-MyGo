package browserpreview

import "testing"

func TestFloatingSourceRequestsFreshSavedRead(t *testing.T) {
	source := DocumentSource{Authority: "host", SessionID: "s", Workspace: "root", Path: "a.pdf", Kind: DocumentPDF}
	b := &Controller{}
	b.OpenDocumentSource(source, true)
	if len(b.Tabs) != 1 || !b.Tabs[0].sourcePending || !b.Tabs[0].floatAfterRead {
		t.Fatal("missing fresh floating intent")
	}
	// No parent window/ready loader: request remains pending, never opens stale capability.
	b.OpenDocumentSource(source, true)
	if len(b.Tabs) != 1 || !b.Tabs[0].sourcePending || !b.Tabs[0].floatAfterRead {
		t.Fatal("reopen lost fresh-read intent")
	}
	b.OpenDocumentSource(source)
	if b.Tabs[0].floatAfterRead {
		t.Fatal("ordinary open retained floating intent")
	}
}
