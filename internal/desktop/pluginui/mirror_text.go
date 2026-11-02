package pluginui

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"unicode/utf16"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

//go:embed mirror_text_guard.js
var mirrorTextGuardSource []byte

type mirrorTextSelection struct {
	Start    int  `json:"start"`
	End      int  `json:"end"`
	Reversed bool `json:"reversed"`
}
type mirrorTextRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}
type mirrorTextFocus struct {
	DocumentKey string              `json:"documentKey"`
	Subject     Subject             `json:"subject"`
	Handle      string              `json:"handle"`
	Entry       string              `json:"entry"`
	Turn        int                 `json:"turn"`
	Seq         uint64              `json:"seq"`
	Focus       string              `json:"focus"`
	Value       string              `json:"value"`
	Selection   mirrorTextSelection `json:"selection"`
	Marked      *mirrorTextRange    `json:"marked"`
	Caret       MirrorRect          `json:"caret"`
}

func (f *mirrorTextFocus) same(g *mirrorTextFocus) bool {
	return f != nil && g != nil && f.DocumentKey == g.DocumentKey && f.Subject.equal(g.Subject) && f.Handle == g.Handle && f.Entry == g.Entry && f.Turn == g.Turn && f.Seq == g.Seq && f.Focus == g.Focus
}
func (f *mirrorTextFocus) valid() bool {
	return validMirrorTextFocus(f)
}

type mirrorTextJob struct {
	event  mygo.BrowserTextEvent
	state  *mirrorTextState
	unmark bool
	focus  *mirrorTextFocus
	source *mygo.Window
	page   uint64
	client *MirrorTextClient
}
type mirrorTextState struct {
	clients   map[string]*MirrorTextClient
	queue     []mirrorTextJob
	busy      bool
	focus     *mirrorTextFocus
	closed    bool
	lastInput time.Time
}

// MirrorTextClient supplies native IME queries from the original focused DOM
// element. It holds only a bounded text/selection projection, never a second
// editor/React owner. The original trusted browser input commits every edit.
type MirrorTextClient struct {
	owner         *Owner
	handle        string
	width, height float32
	row           MirrorRect
	focus         *mirrorTextFocus
	source        *mygo.Window
	page          uint64
	value         string
	selection     ui.TextInputSelection
	marked        *ui.TextInputRange
	pending       int
	updated       time.Time
}

func (o *Owner) mirrorTextLocked() *mirrorTextState {
	if o.mirror == nil {
		return nil
	}
	if o.mirror.text == nil {
		o.mirror.text = &mirrorTextState{clients: map[string]*MirrorTextClient{}}
	}
	return o.mirror.text
}
func (o *Owner) MirrorTextClient(f *MirrorFrame, row MirrorOccurrence) *MirrorTextClient {
	if o == nil || f == nil {
		return nil
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.mirrorLayoutCurrentLocked(f) {
		return nil
	}
	state := o.mirrorTextLocked()
	client := state.clients[row.Handle]
	if client == nil {
		client = &MirrorTextClient{owner: o, handle: row.Handle}
		state.clients[row.Handle] = client
	}
	client.row = row.Rect
	focus := f.textFocus
	if o.mirrorCurrentLocked(f) && focus.valid() && focus.Handle == row.Handle {
		if client.pending == 0 || !client.focus.same(focus) {
			client.applyTextSampleLocked(focus, f.textAt)
		}
		client.source, client.page = f.source, f.page
	} else if client.pending == 0 && o.mirrorCurrentLocked(f) {
		client.focus = nil
		client.marked = nil
	}
	return client
}
func (c *MirrorTextClient) applyFocusLocked(f *mirrorTextFocus) {
	if !f.valid() {
		c.focus = nil
		c.marked = nil
		return
	}
	copy := *f
	c.focus = &copy
	c.value = f.Value
	c.selection = ui.TextInputSelection{Range: ui.TextInputRange{Start: f.Selection.Start, End: f.Selection.End}, Reversed: f.Selection.Reversed}
	c.marked = nil
	if f.Marked != nil {
		r := ui.TextInputRange{Start: f.Marked.Start, End: f.Marked.End}
		c.marked = &r
	}
}
func (c *MirrorTextClient) SetGeometry(width, height float32) {
	if c == nil {
		return
	}
	c.owner.mu.Lock()
	c.width, c.height = width, height
	c.owner.mu.Unlock()
}
func (c *MirrorTextClient) TextForRange(r ui.TextInputRange) (string, ui.TextInputRange) {
	c.owner.mu.Lock()
	defer c.owner.mu.Unlock()
	start := ui.UTF16ByteOffset(c.value, r.Start)
	end := ui.UTF16ByteOffset(c.value, max(r.Start, r.End))
	return c.value[start:end], ui.TextInputRange{Start: ui.UTF16Len(c.value[:start]), End: ui.UTF16Len(c.value[:end])}
}
func (c *MirrorTextClient) Selection() ui.TextInputSelection {
	c.owner.mu.Lock()
	defer c.owner.mu.Unlock()
	return c.selection
}
func (c *MirrorTextClient) MarkedRange() (ui.TextInputRange, bool) {
	c.owner.mu.Lock()
	defer c.owner.mu.Unlock()
	if c.marked == nil {
		return ui.TextInputRange{}, false
	}
	return *c.marked, true
}
func (c *MirrorTextClient) replacementLocked(r *ui.TextInputRange) ui.TextInputRange {
	if r != nil {
		return *r
	}
	if c.marked != nil {
		return *c.marked
	}
	return c.selection.Range
}
func (c *MirrorTextClient) optimisticReplaceLocked(r ui.TextInputRange, text string) {
	start := ui.UTF16ByteOffset(c.value, r.Start)
	end := ui.UTF16ByteOffset(c.value, max(r.Start, r.End))
	c.value = c.value[:start] + text + c.value[end:]
	caret := ui.UTF16Len(c.value[:start]) + ui.UTF16Len(text)
	c.selection = ui.TextInputSelection{Range: ui.TextInputRange{Start: caret, End: caret}}
	c.updated = time.Now()
}
func (c *MirrorTextClient) ReplaceText(r *ui.TextInputRange, text string) {
	c.owner.mu.Lock()
	replace := c.replacementLocked(r)
	// InsertText replaces the actual DOM selection or current composition.
	// A refused unrelated replacement must not mutate the local projection.
	if r != nil && (!c.focus.valid() || replace.Start != c.focus.Selection.Start || replace.End != c.focus.Selection.End) {
		c.owner.mu.Unlock()
		return
	}
	c.optimisticReplaceLocked(replace, text)
	c.marked = nil
	c.owner.mu.Unlock()
	c.enqueue(mygo.BrowserTextEvent{Kind: mygo.BrowserTextInsertText, Text: text})
}
func (c *MirrorTextClient) SetMarkedText(r *ui.TextInputRange, text string, selected ui.TextInputRange) {
	c.owner.mu.Lock()
	replace := c.replacementLocked(r)
	c.optimisticReplaceLocked(replace, text)
	marked := ui.TextInputRange{Start: replace.Start, End: replace.Start + ui.UTF16Len(text)}
	c.marked = &marked
	c.selection = ui.TextInputSelection{Range: ui.TextInputRange{Start: replace.Start + selected.Start, End: replace.Start + selected.End}}
	c.owner.mu.Unlock()
	event := mygo.BrowserTextEvent{Kind: mygo.BrowserTextComposition, Text: text, SelectionStart: selected.Start, SelectionEnd: selected.End}
	if r != nil {
		start, end := r.Start, r.End
		event.ReplacementStart, event.ReplacementEnd = &start, &end
	}
	c.enqueue(event)
}
func (c *MirrorTextClient) UnmarkText() {
	c.owner.mu.Lock()
	if c.marked == nil {
		c.owner.mu.Unlock()
		return
	}
	r := *c.marked
	start, end := ui.UTF16ByteOffset(c.value, r.Start), ui.UTF16ByteOffset(c.value, r.End)
	text := c.value[start:end]
	c.marked = nil
	c.owner.mu.Unlock()
	c.enqueueJob(mygo.BrowserTextEvent{Kind: mygo.BrowserTextInsertText, Text: text}, true)
}
func (c *MirrorTextClient) BoundsForRange(r ui.TextInputRange) (ui.Rect, ui.TextInputRange, bool) {
	c.owner.mu.Lock()
	defer c.owner.mu.Unlock()
	if !c.focus.valid() || c.row.W <= 0 || c.row.H <= 0 || c.width <= 0 || c.height <= 0 {
		return ui.Rect{}, r, false
	}
	caret := c.focus.Caret
	rect := ui.Rect{X: float32((caret.X-c.row.X)/c.row.W) * c.width, Y: float32((caret.Y-c.row.Y)/c.row.H) * c.height, W: max(1, float32(caret.W/c.row.W)*c.width), H: max(1, float32(caret.H/c.row.H)*c.height)}
	caretIndex := c.selection.Range.End
	if c.selection.Reversed {
		caretIndex = c.selection.Range.Start
	}
	if r.Start != r.End || r.Start != caretIndex {
		return ui.Rect{}, r, false
	}
	return rect, ui.TextInputRange{Start: caretIndex, End: caretIndex}, true
}
func (c *MirrorTextClient) IndexForPoint(p ui.Point) (int, bool) {
	c.owner.mu.Lock()
	defer c.owner.mu.Unlock()
	if !c.focus.valid() || c.row.W <= 0 || c.row.H <= 0 {
		return 0, false
	}
	caret := c.focus.Caret
	x, y := float32((caret.X-c.row.X)/c.row.W)*c.width, float32((caret.Y-c.row.Y)/c.row.H)*c.height
	h := float32(caret.H/c.row.H) * c.height
	index := c.selection.Range.End
	if c.selection.Reversed {
		index = c.selection.Range.Start
	}
	return index, p.X >= x-2 && p.X <= x+3 && p.Y >= y && p.Y <= y+h
}

func (c *MirrorTextClient) enqueue(event mygo.BrowserTextEvent) bool {
	return c.enqueueJob(event, false)
}
func (c *MirrorTextClient) enqueueJob(event mygo.BrowserTextEvent, unmark bool) bool {
	o := c.owner
	o.mu.Lock()
	state := o.mirrorTextLocked()
	if state == nil || state.closed || !c.focus.valid() || c.source == nil || o.closed || o.masked || !o.state.Connected || o.state.Page != c.page || !o.selected.equal(c.focus.Subject) {
		o.mu.Unlock()
		return false
	}
	if len(state.queue) >= 32 {
		state.queue = nil
		for _, client := range state.clients {
			client.pending = 0
			client.focus = nil
			client.marked = nil
		}
		o.mirror.error = "plugin mirror: native text input queue exceeded"
		o.mu.Unlock()
		return false
	}
	state.lastInput = time.Now()
	focus := *c.focus
	state.queue = append(state.queue, mirrorTextJob{event: event, state: state, unmark: unmark, focus: &focus, source: c.source, page: c.page, client: c})
	c.pending++
	if state.busy {
		o.mu.Unlock()
		return true
	}
	state.busy = true
	o.mu.Unlock()
	go o.textWorker(state)
	return true
}
func (o *Owner) textWorker(state *mirrorTextState) {
	for {
		o.mu.Lock()
		if o.closed || o.mirror == nil || o.mirror.text != state || len(state.queue) == 0 || state.closed {
			state.busy = false
			o.mu.Unlock()
			o.notify()
			return
		}
		job := state.queue[0]
		state.queue = state.queue[1:]
		o.mu.Unlock()
		focus, err := o.dispatchText(job)
		o.mu.Lock()
		if o.closed || o.mirror == nil || o.mirror.text != state {
			state.busy = false
			o.mu.Unlock()
			return
		}
		o.mirror.pixelDirty = true
		job.client.pending = max(0, job.client.pending-1)
		if err != nil {
			o.mirror.error = err.Error()
			state.queue = nil
			for _, client := range state.clients {
				client.pending = 0
				client.focus = nil
				client.marked = nil
			}
		} else if job.client.pending == 0 {
			job.client.applyFocusLocked(focus)
			job.client.updated = time.Now()
		}
		o.mu.Unlock()
		o.notify()
	}
}
func (o *Owner) textCurrent(job mirrorTextJob) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return !o.closed && !o.masked && !o.incompatible && o.mirror != nil && o.mirror.text == job.state && !job.state.closed && o.state.Connected && o.state.Page == job.page && o.selected.equal(job.focus.Subject) && job.client.focus.same(job.focus)
}
func textTypes(event mygo.BrowserTextEvent) []string {
	switch event.Kind {
	case mygo.BrowserTextKeyDown:
		return []string{"keydown", "beforeinput", "input"}
	case mygo.BrowserTextKeyUp:
		return []string{"keyup"}
	case mygo.BrowserTextInsertText:
		return []string{"beforeinput", "input", "compositionupdate", "compositionend"}
	case mygo.BrowserTextComposition:
		return []string{"compositionstart", "compositionupdate", "compositionend", "beforeinput", "input"}
	case mygo.BrowserTextCancelComposition:
		return []string{"compositionend", "beforeinput", "input"}
	}
	return nil
}
func (o *Owner) dispatchText(job mirrorTextJob) (*mirrorTextFocus, error) {
	if !o.textCurrent(job) {
		return nil, errSessionUnavailable
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	o.mu.Lock()
	state := o.mirror
	o.mu.Unlock()
	if err := o.awaitMirrorCapture(ctx, state); err != nil {
		return nil, err
	}
	now, err := mirrorEval[*mirrorTextFocus](ctx, job.source.Page(), "window.__whaleInlineMirror.textFocus()")
	if err != nil {
		return nil, err
	}
	if !job.focus.same(now) {
		return nil, errors.New("plugin mirror: original text focus changed")
	}
	if job.unmark && (now.Marked == nil || job.focus.Marked == nil || *now.Marked != *job.focus.Marked) {
		return now, nil
	}
	nonce := rand.Text()
	arm := map[string]any{"documentKey": now.DocumentKey, "subject": now.Subject, "handle": now.Handle, "entry": now.Entry, "turn": now.Turn, "seq": now.Seq, "focus": now.Focus, "nonce": nonce, "types": textTypes(job.event), "key": job.event.Key, "code": job.event.Code, "endComposition": job.event.Kind == mygo.BrowserTextInsertText || job.event.Kind == mygo.BrowserTextCancelComposition || job.event.Kind == mygo.BrowserTextComposition && job.event.Text == ""}
	raw, _ := json.Marshal(arm)
	accepted, err := mirrorEval[struct {
		OK bool `json:"ok"`
	}](ctx, job.source.Page(), "window.__whaleInlineMirror.textArm("+string(raw)+")")
	if err != nil {
		return nil, err
	}
	if !accepted.OK || !o.textCurrent(job) {
		return nil, errSessionUnavailable
	}
	err = job.source.Page().TextInput(ctx, job.event)
	receipt, receiptErr := mirrorEval[struct {
		OK     bool             `json:"ok"`
		Focus  *mirrorTextFocus `json:"focus"`
		Reason string           `json:"reason"`
	}](ctx, job.source.Page(), "window.__whaleInlineMirror.textDisarm("+fmt.Sprintf("%q", nonce)+")")
	if err != nil {
		return nil, err
	}
	if receiptErr != nil {
		return nil, receiptErr
	}
	if !receipt.OK && job.event.Kind != mygo.BrowserTextKeyUp {
		return nil, fmt.Errorf("plugin mirror: original text handler refused kind %d: %s", job.event.Kind, receipt.Reason)
	}
	return receipt.Focus, nil
}

// MirrorTextInput forwards keys without committed text. Native InputText and
// native TextInputClient commits remain distinct, so Unicode is inserted once.
func (c *MirrorTextClient) HandleInput(event ui.InputEvent) bool {
	if c == nil {
		return false
	}
	if event.Kind == ui.InputBlur || event.Kind == ui.InputCommand {
		return c.handleSpecial(event)
	}
	if event.Kind == ui.InputText {
		return c.enqueue(mygo.BrowserTextEvent{Kind: mygo.BrowserTextInsertText, Text: event.Text})
	}
	if event.Kind == ui.InputCompose {
		runes := []rune(event.Text)
		caret := len(utf16.Encode(runes[:min(len(runes), max(0, event.Caret))]))
		return c.enqueue(mygo.BrowserTextEvent{Kind: mygo.BrowserTextComposition, Text: event.Text, SelectionStart: caret, SelectionEnd: caret})
	}
	if event.Kind != ui.InputKeyDown && event.Kind != ui.InputKeyUp {
		return false
	}
	key, code, vk := mirrorKey(event)
	if key == "" || code == "" {
		return false
	}
	kind := mygo.BrowserTextKeyDown
	if event.Kind == ui.InputKeyUp {
		kind = mygo.BrowserTextKeyUp
	}
	var mods mygo.BrowserTextModifiers
	if event.Mods&ui.Alt != 0 {
		mods |= mygo.BrowserTextAlt
	}
	if event.Mods&ui.Ctrl != 0 {
		mods |= mygo.BrowserTextCtrl
	}
	if event.Mods&ui.Super != 0 {
		mods |= mygo.BrowserTextMeta
	}
	if event.Mods&ui.Shift != 0 {
		mods |= mygo.BrowserTextShift
	}
	return c.enqueue(mygo.BrowserTextEvent{Kind: kind, Key: key, Code: code, WindowsVirtualKeyCode: vk, Modifiers: mods, Repeat: event.Repeat && kind == mygo.BrowserTextKeyDown})
}
