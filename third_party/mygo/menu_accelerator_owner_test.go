package mygo

import "testing"

func TestMenuSuppressAcceleratorRetainsRoleAndKeycap(t *testing.T) {
	menu := NewMenu([]*MenuItem{
		{ID: "normal", Role: RoleToggleDevTools},
		{ID: "page-owned", Label: "Developer tools", Role: RoleToggleDevTools, SuppressAccelerator: true},
		{ID: "explicit", Label: "App command    Mod+Alt+F8", Accelerator: "Ctrl+Alt+F8", SuppressAccelerator: true},
	})
	normal, owned := menu.ItemByID("normal"), menu.ItemByID("page-owned")
	if normal.Accelerator == "" {
		t.Fatal("ordinary role lost its default accelerator")
	}
	if owned.Accelerator != "" || owned.Role != RoleToggleDevTools {
		t.Fatal("suppression lost role or restored default accelerator")
	}
	owned.SetAccelerator("Ctrl+Shift+I")
	snapshot := menu.snapshot()
	if snapshot.Items[0].Accelerator == "" {
		t.Fatal("ordinary role lost native registration")
	}
	if snapshot.Items[1].Accelerator != "" || snapshot.Items[1].Role != string(RoleToggleDevTools) {
		t.Fatal("late setter restored native accelerator or lost click role")
	}
	if snapshot.Items[2].Accelerator != "" || snapshot.Items[2].Label != "App command    Mod+Alt+F8" {
		t.Fatal("native suppression changed the app-managed keycap")
	}
}
