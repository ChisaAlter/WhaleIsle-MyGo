package composer

import (
	"testing"
	"whaleisle/internal/store/session"
)

func TestGroupReplyDraftPreservesCancellation(t *testing.T) {
	d := NewDraftState("original")
	d.Attachments = []session.ComposerAttachment{{Name: "old.png"}}
	if !d.BeginGroupReply("t1") {
		t.Fatal("reply rejected")
	}
	if d.Text != "/dshbot-thread-reply t1 " || d.BeginGroupReply("t2") {
		t.Fatal(d.Text)
	}
	d.Text += "new"
	d.Attachments = nil
	d.CancelGroupReply()
	if d.Text != "original" || len(d.Attachments) != 1 || d.GroupReply != nil {
		t.Fatal(d)
	}
}

func TestGroupMentionCandidatesKeepStableName(t *testing.T) {
	items := buildSuggestItems(Model{GroupMembers: []session.GroupMember{{ID: "b", Name: "stable", Title: "Display"}}}, SuggestMention, "display")
	if len(items) != 1 || items[0].insert != "@stable " {
		t.Fatal(items)
	}
	items = buildSuggestItems(Model{GroupMembers: []session.GroupMember{}}, SuggestMention, "all")
	if len(items) != 1 || items[0].insert != "@everyone " {
		t.Fatal(items)
	}
	if got := buildSuggestItemsWithQuote(Model{GroupMembers: []session.GroupMember{}}, SuggestMention, "all", true); len(got) != 0 {
		t.Fatal(got)
	}
}
