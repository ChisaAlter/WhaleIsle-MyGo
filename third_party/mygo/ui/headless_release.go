//go:build release

package ui

import (
	"image"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo/internal/platform"
	"github.com/egoist/mygo/internal/raster"
	"github.com/egoist/mygo/internal/scene"
	"github.com/egoist/mygo/transfer"
)

type headless struct {
	w, h, scale float32
	img         raster.Renderer
	requested   atomic.Bool
	dark        bool
	prefs       platform.Preferences
	clipboard   string
	cursor      Cursor
	ime         platform.TextInputState
	opened      []string
	openErr     error
	later       []func()
	bar         TitleBar
	access      *platform.AccessTree
	told        []string
	announced   []string
	menu        *platform.Menu
	menuAt      [2]float32
	chosen      func(int)
	hz          float32
	hidden      bool
	material    bool
	reading     func()
	last        *scene.Scene
	dragData    transfer.Data
	dragLocal   any
	dragOptions transfer.DragOptions
}

func (h *headless) size() (w, ht, scale float32)                  { return 0, 0, 1 }
func (h *headless) pixels() bool                                  { return false }
func (h *headless) refreshRate() float32                          { return 0 }
func (h *headless) occluded() bool                                { return false }
func (h *headless) present(s *scene.Scene)                        {}
func (h *headless) requestFrame()                                 {}
func (h *headless) setCursor(Cursor)                              {}
func (h *headless) setTextInput(t platform.TextInputState)        {}
func (h *headless) updateAccessibility(tree *platform.AccessTree) {}
func (h *headless) readClipboard() string                         { return "" }
func (h *headless) writeClipboard(string)                         {}
func (h *headless) startDrag()                                    {}
func (h *headless) startDataDrag(transfer.Data, any, transfer.DragOptions, float32, float32) error {
	return nil
}
func (h *headless) cancelDataDrag()                                            {}
func (h *headless) finishDataDrag(r transfer.Result)                           {}
func (h *headless) setDropFormats([]transfer.Format)                           {}
func (h *headless) titleBarDoubleClicked()                                     {}
func (h *headless) isDark() bool                                               { return false }
func (h *headless) preferences() platform.Preferences                          { return platform.Preferences{} }
func (h *headless) titleBar() TitleBar                                         { return TitleBar{} }
func (h *headless) vibrancy() bool                                             { return false }
func (h *headless) invalidate()                                                {}
func (h *headless) post(fn func())                                             {}
func (h *headless) openURL(url string, done func(error))                       {}
func (h *headless) popupMenu(m *platform.Menu, x, y float32, chosen func(int)) {}
func (h *headless) image() *image.RGBA                                         { return nil }

// InteractiveBox describes an element of the last frame that takes the
// pointer or the focus, for audits of a view's interaction.
type InteractiveBox struct {
	Rect   Rect
	Label  string
	Role   Role
	Kinds  []string
	Cursor Cursor
}

func (b InteractiveBox) String() string { return "" }

// Tester is not supported in release builds.
type Tester struct {
	rt *engine
	h  *headless
}

func (t *Tester) send(ev platform.SurfaceEvent) {}
func (t *Tester) settle()                       {}

func coreNewTester(view func(c *context), width, height int) *Tester {
	panic("ui: NewTester is not supported in release builds")
}

func coreRender(view func(c *context), width, height int, scale float32) *image.RGBA {
	panic("ui: Render is not supported in release builds")
}

func (t *Tester) SetSize(width, height int)                    {}
func (t *Tester) SetScale(scale float32)                       {}
func (t *Tester) SetTitleBar(bar TitleBar)                     {}
func (t *Tester) SetVibrancy(shows bool)                       {}
func (t *Tester) SetDark(dark bool)                            {}
func (t *Tester) SetPreferences(p Preferences)                 {}
func (t *Tester) Frame()                                       {}
func (t *Tester) SetTime(now time.Time)                        {}
func (t *Tester) NativeTextInputClient() TextInputClient       { return nil }
func (t *Tester) Image() *image.RGBA                           { return nil }
func (t *Tester) Pixels() []byte                               { return nil }
func (t *Tester) Find(s string) (Rect, bool)                   { return Rect{}, false }
func (t *Tester) FindAll(s string) []Rect                      { return nil }
func (t *Tester) Texts() []string                              { return nil }
func (t *Tester) HasText(s string) bool                        { return false }
func (t *Tester) Click(s string) error                         { return nil }
func (t *Tester) ClickAt(x, y float32)                         {}
func (t *Tester) ClickWith(mods Modifiers, s string) error     { return nil }
func (t *Tester) ClickAtWith(mods Modifiers, x, y float32)     {}
func (t *Tester) RightClick(s string) error                    { return nil }
func (t *Tester) RightClickAt(x, y float32)                    {}
func (t *Tester) Menu() []string                               { return nil }
func (t *Tester) ChooseMenuItem(path ...string) error          { return nil }
func (t *Tester) CloseMenu()                                   {}
func (t *Tester) Press(x, y float32)                           {}
func (t *Tester) Release(x, y float32)                         {}
func (t *Tester) Move(x, y float32)                            {}
func (t *Tester) Scroll(x, y, dx, dy float32)                  {}
func (t *Tester) HoldModifiers(mods Modifiers)                 {}
func (t *Tester) Key(mods Modifiers, key Key)                  {}
func (t *Tester) TypeKey(mods Modifiers, key Key, text string) {}
func (t *Tester) Type(s string)                                {}
func (t *Tester) SetFocused(focused bool)                      {}
func (t *Tester) Compose(text string, caret int)               {}
func (t *Tester) Command(name string)                          {}
func (t *Tester) TextCaret() (Rect, bool)                      { return Rect{}, false }
func (t *Tester) Clipboard() string                            { return "" }
func (t *Tester) SetClipboard(s string)                        {}
func (t *Tester) Cursor() Cursor                               { return CursorDefault }
func (t *Tester) OpenedURLs() []string                         { return nil }
func (t *Tester) FailOpenURL(err error)                        {}
func (t *Tester) Announcements() []string                      { return nil }
func (t *Tester) Focused(s string) bool                        { return false }
func (t *Tester) SetValue(s, value string) error               { return nil }
func (t *Tester) PhysicalKeyDown(code string, key Key, mods Modifiers, repeat, blocked, dead bool) {
}
func (t *Tester) PhysicalKeyUp(code string, key Key, mods Modifiers, blocked bool) {}
func (t *Tester) Interactives() []InteractiveBox                                   { return nil }
