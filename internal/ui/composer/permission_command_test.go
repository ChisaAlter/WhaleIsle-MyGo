package composer

import (
	"testing"
	"whaleisle/internal/store/session"
)

func TestBarePermissionCommandStaysLocal(t *testing.T) {
	d := NewDraftState(" /PeRmIsSiOn ")
	if !d.HandleModelCommand("s") {
		t.Fatal("bare permission would reach Host command path")
	}
	if d.Text != "" {
		t.Fatal("accepted command not cleared")
	}
	for _, text := range []string{"/permission safe", "/permission\nmessage"} {
		d.Text = text
		if d.HandleModelCommand("s") {
			t.Fatalf("claimed argument-bearing command %q", text)
		}
	}
	d.Text = "/permission"
	d.TargetID = "other"
	if !d.HandleModelCommand("s") || d.Error == "" || d.Text == "" {
		t.Fatal("wrong-session request escaped or lost draft")
	}
}

func TestPermissionCommandOwnershipAndCatalog(t *testing.T) {
	d := NewDraftState("/permission")
	d.HandleModelCommand("a")
	if !d.PermissionCommand.Visible("a") || d.PermissionCommand.Visible("b") {
		t.Fatal("permission picker ownership")
	}
	d.PermissionCommand.Sync("b")
	if d.PermissionCommand.Open {
		t.Fatal("picker survived navigation")
	}
	d.Text = "/permission"
	d.Attachments = append(d.Attachments, session.ComposerAttachment{Path: "file"})
	if !d.HandleModelCommand("a") || d.Error == "" || d.PermissionCommand.Open {
		t.Fatal("attachment command accepted")
	}
	if LocalSlashAction("/permission", MergeSlashCommands(StubSlashCommands(), []SlashCommand{{Line: "/permission", Description: "remote"}})) != "permission" {
		t.Fatal("remote catalog replaced local entry")
	}
}
