//go:build release

package ui

func registerWindowHost(win any, h *windowHost) {}
func unregisterWindowHost(win any)              {}

// WindowDriver is not available in release builds.
type WindowDriver struct{}

// DriveWindow returns nil in release builds.
func DriveWindow(win any) *WindowDriver {
	return nil
}

func (d *WindowDriver) Frame()                      {}
func (d *WindowDriver) Find(s string) (Rect, bool)  { return Rect{}, false }
func (d *WindowDriver) FindAll(s string) []Rect     { return nil }
func (d *WindowDriver) Texts() []string             { return nil }
func (d *WindowDriver) HasText(s string) bool       { return false }
func (d *WindowDriver) Cursor() Cursor              { return CursorDefault }
func (d *WindowDriver) Focused(s string) bool       { return false }
func (d *WindowDriver) Move(x, y float32)           {}
func (d *WindowDriver) Hover(s string) error        { return nil }
func (d *WindowDriver) ClickAt(x, y float32)        {}
func (d *WindowDriver) Click(s string) error        { return nil }
func (d *WindowDriver) Scroll(x, y, dx, dy float32) {}
func (d *WindowDriver) Key(mods Modifiers, key Key) {}
func (d *WindowDriver) Type(s string)               {}
