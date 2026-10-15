package composer

import "testing"

func TestLocalModelCommand(t *testing.T) {
	var p ModelCommandState
	if handled, err := p.Handle(" /MoDeL ", "a", "", false, 0); !handled || err != "" || !p.Visible("a") {
		t.Fatalf("local command not opened: %v %q %+v", handled, err, p)
	}
	if p.Visible("b") {
		t.Fatal("picker leaked across sessions")
	}
	p.Sync("b")
	if p.Visible("a") {
		t.Fatal("old picker survived navigation")
	}
	for _, text := range []string{"/model foo", "/model\nhello", "hello /model"} {
		if handled, _ := p.Handle(text, "a", "", false, 0); handled {
			t.Fatalf("claimed normal input %q", text)
		}
	}
	if handled, _ := p.Handle("/model", "a", "", true, 0); handled {
		t.Fatal("claimed edit")
	}
	for _, tc := range []struct {
		id, target  string
		attachments int
	}{{"", "", 0}, {"a", "b", 0}, {"a", "", 1}} {
		handled, err := p.Handle("/model", tc.id, tc.target, false, tc.attachments)
		if !handled || err == "" || p.Open {
			t.Fatalf("unsafe local command escaped: %+v %v %q", tc, handled, err)
		}
	}
	if LocalSlashAction("/model", StubSlashCommands()) != "model" {
		t.Fatal("local entry missing")
	}
	catalog := MergeSlashCommands(StubSlashCommands(), []SlashCommand{{Line: "/model", Description: "remote"}})
	if LocalSlashAction("/model", catalog) != "model" {
		t.Fatal("Host catalog overrode local entry")
	}
}
