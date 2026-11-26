package ui

import (
	"bytes"
	"testing"
	"time"
)

func TestMetallicPaintButtonHoverSweepAndGates(t *testing.T) {
	now := time.Unix(1_700_000_000, 0) // Cos(2*pi*unix/4.5) varies; we set exact times via tt.SetTime
	enabled := true
	role := RoleAuto

	build := func(metallic bool, reduced bool) *Tester {
		tt := NewTester(func(c *Context) {
			SetDecorativeEffects(c, DecorativeEffects{MetallicButtons: metallic})
			Column(c).Fill().Padding(24).Children(func() {
				b := Button(c, "MetallicTarget").Background(c.Theme().Surface)
				if !enabled {
					b = b.Disabled(true)
				}
				if role != RoleAuto {
					b = b.Role(role)
				}
			})
		}, 320, 140)
		tt.SetPreferences(Preferences{ReduceMotion: reduced})
		tt.SetTime(now)
		return tt
	}

	// 1. Enabled Button hovered with MetallicButtons=true animates across 4.5s sweep.
	tt := build(true, false)
	box, ok := tt.Find("MetallicTarget")
	if !ok {
		t.Fatal("MetallicTarget button not found")
	}
	cx, cy := box.X+box.W/2, box.Y+box.H/2
	tt.Move(cx, cy)

	tt.SetTime(time.Unix(0, 0)) // phase = 0
	p0 := append([]byte(nil), tt.Pixels()...)
	tt.SetTime(time.Unix(2, 250_000_000)) // 2.25s => phase = 1
	p1 := append([]byte(nil), tt.Pixels()...)
	if bytes.Equal(p0, p1) {
		t.Fatal("hovered metallic button did not animate sweep between t=0 and t=2.25s")
	}
	tt.h.requested.Store(false)
	tt.rt.runFrame()
	if !tt.h.requested.Load() {
		t.Fatal("hovered metallic button stopped requesting animation frames while ReduceMotion=false")
	}

	// 2. ReduceMotion retains static midpoint sheen (differs from MetallicButtons=false) without continuous frames.
	plain := build(false, true)
	pBox, _ := plain.Find("MetallicTarget")
	plain.Move(pBox.X+pBox.W/2, pBox.Y+pBox.H/2)
	plainPix := append([]byte(nil), plain.Pixels()...)

	reduced := build(true, true)
	rBox, _ := reduced.Find("MetallicTarget")
	reduced.Move(rBox.X+rBox.W/2, rBox.Y+rBox.H/2)
	reduced.h.requested.Store(false)
	reduced.rt.runFrame()
	if reduced.h.requested.Load() {
		t.Fatal("metallic button still requested animation frame under ReduceMotion=true")
	}
	if bytes.Equal(plainPix, reduced.Pixels()) {
		t.Fatal("ReduceMotion metallic button lost static sheen")
	}

	// 3. Disabled buttons and non-button roles (disclosure, menu item, popup select) must not paint sheen.
	for _, tc := range []struct {
		name    string
		enabled bool
		role    Role
	}{
		{"disabled", false, RoleAuto},
		{"disclosure", true, RoleDisclosure},
		{"menuitem", true, RoleMenuItem},
		{"popup", true, RolePopUpButton},
	} {
		t.Run(tc.name, func(t *testing.T) {
			enabled, role = tc.enabled, tc.role
			off := build(false, false)
			bOff, _ := off.Find("MetallicTarget")
			off.Move(bOff.X+bOff.W/2, bOff.Y+bOff.H/2)
			off.SetTime(time.Unix(2, 250_000_000))
			offPix := append([]byte(nil), off.Pixels()...)

			on := build(true, false)
			bOn, _ := on.Find("MetallicTarget")
			on.Move(bOn.X+bOn.W/2, bOn.Y+bOn.H/2)
			on.SetTime(time.Unix(2, 250_000_000))
			if !bytes.Equal(offPix, on.Pixels()) {
				t.Fatalf("%s unexpectedly painted metallic hover sheen", tc.name)
			}
		})
	}
}
