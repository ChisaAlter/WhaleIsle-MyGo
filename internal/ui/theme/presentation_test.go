package theme

import (
	"reflect"
	"testing"
)

func TestConversationSemanticRolesMatchCurrentOriginal(t *testing.T) {
	for _, tc := range []struct {
		id   string
		want map[string]string
	}{
		{IDDeepseekLight, map[string]string{"Bubble": "rgb(237, 243, 254)", "InputMajor": "rgba(255, 255, 255, 0.8)", "ButtonInfoFill": "rgb(65, 118, 230)", "SelectorFill": "rgb(245, 246, 247)"}},
		{IDDeepseekDark, map[string]string{"Bubble": "rgb(44, 44, 46)", "InputMajor": "rgba(44, 44, 46, 0.8)", "ButtonInfoFill": "rgb(122, 170, 255)", "SelectorFill": "rgb(53, 54, 56)"}},
	} {
		v := reflect.ValueOf(Resolve(tc.id).Active(false))
		for name, want := range tc.want {
			f := v.FieldByName(name)
			if !f.IsValid() {
				t.Errorf("%s role %s missing", tc.id, name)
				continue
			}
			if got := f.String(); got != want {
				t.Errorf("%s.%s=%q want %q", tc.id, name, got, want)
			}
		}
	}
}

func TestNativeInteractionRolesMatchOriginal(t *testing.T) {
	for _, tc := range []struct {
		id, pressed, infoHover string
	}{
		{IDDeepseekLight, "rgba(38, 49, 72, 0.1)", "rgb(122, 170, 255)"},
		{IDDeepseekDark, "rgba(255, 255, 255, 0.14)", "rgb(65, 118, 230)"},
	} {
		c := Resolve(tc.id).Active(false)
		if c.Pressed != tc.pressed || c.ButtonInfoHover != tc.infoHover {
			t.Errorf("%s pressed/hover=%q/%q want %q/%q", tc.id, c.Pressed, c.ButtonInfoHover, tc.pressed, tc.infoHover)
		}
	}
}

func TestCustomFamilyInteractionHoverUsesItsAccent(t *testing.T) {
	for _, tc := range []struct {
		dark bool
		want string
	}{
		{false, "#345475"},
		{true, "#6282a3"},
	} {
		c := palette(Seed{Background: "#ffffff", Foreground: "#102030", Accent: "#40668f"}, tc.dark)
		if c.ButtonInfoHover != tc.want {
			t.Errorf("dark=%t custom hover=%q want %q", tc.dark, c.ButtonInfoHover, tc.want)
		}
	}
}

func TestComposerCaptionRoleMatchesCurrentOriginal(t *testing.T) {
	for _, tc := range []struct{ id, want string }{{IDDeepseekLight, "#adb2b8"}, {IDDeepseekDark, "#81858c"}} {
		f := reflect.ValueOf(Resolve(tc.id).Active(false)).FieldByName("Caption")
		if !f.IsValid() {
			t.Errorf("%s composer caption role missing", tc.id)
			continue
		}
		if f.String() != tc.want {
			t.Errorf("%s caption=%q want%q", tc.id, f.String(), tc.want)
		}
	}
}
