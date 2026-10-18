package browserpreview

import "testing"

func TestTabsKeepDraftsAndSelection(t *testing.T) {
	b := NewController(nil)
	b.NewTab("")
	b.current().Draft = "https://first.example"
	first := b.current()
	b.NewTab("")
	b.current().Draft = "https://second.example"
	b.Select(0)
	if b.current() != first || b.current().Draft != "https://first.example" {
		t.Fatal("tab state was lost")
	}
	b.CloseTab(0)
	if b.current().Draft != "https://second.example" {
		t.Fatal("wrong remaining tab")
	}
	b.CloseTab(0)
	if len(b.Tabs) != 1 || b.current().Host != nil {
		t.Fatal("last tab must leave an empty tab")
	}
	b.Close()
}
