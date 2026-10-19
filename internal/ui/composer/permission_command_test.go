package composer

import (
	"testing"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/store/session"
)

func TestBarePermissionCommandStaysLocal(t *testing.T) {
	snap := nativeCommandSnapshot(t)
	d := NewDraftState(" /permission ")
	if !d.HandleNativeCommand(snap, nil) || d.Text != " /permission " || !d.PermissionCommand.Visible("a") {
		t.Fatal("bare permission failed to retain its token for selection")
	}
	d.RetireNativeCommands()
	for _, text := range []string{"/permission safe", "/permission\nmessage"} {
		d.Text = text
		if d.HandleNativeCommand(snap, nil) {
			t.Fatalf("claimed argument-bearing Host command %q", text)
		}
	}
	d.Text = "/permission"
	d.TargetID = "other"
	if !d.HandleNativeCommand(snap, nil) || d.Error == "" || d.Text == "" {
		t.Fatal("wrong-session request escaped or lost draft")
	}
}
func TestPermissionCommandOwnershipAndCatalog(t *testing.T) {
	snap := nativeCommandSnapshot(t)
	d := NewDraftState("/permission")
	d.HandleNativeCommand(snap, nil)
	if !d.PermissionCommand.Visible("a") || d.PermissionCommand.Visible("b") {
		t.Fatal("permission picker ownership")
	}
	d.PermissionCommand.Sync("b")
	if d.PermissionCommand.Open {
		t.Fatal("picker survived navigation")
	}
	d.Attachments = []session.ComposerAttachment{{Path: "file"}}
	if !d.HandleNativeCommand(snap, nil) || d.Error == "" || d.PermissionCommand.Open || d.Text != "/permission" {
		t.Fatal("attachment command accepted or lost token")
	}
	d.Attachments = nil
	snap.SlashCommands = []api.ListResultItem2{{Name: "feedback"}}
	if d.HandleNativeCommand(snap, nil) {
		t.Fatal("permission decorator manufactured a missing Host row")
	}
}
