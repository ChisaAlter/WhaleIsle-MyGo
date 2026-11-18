package composer

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/store/session"
)

// PendingFiles is the recoverable user input, never a cached Host receipt.
type PendingFiles struct {
	Paths       []string
	RegularOnly bool
}

// FileIntake belongs to one stable draft. Only the UI thread reads or changes
// it; the worker receives a copy of Paths and returns through the shell owner.
type FileIntake struct {
	PendingFiles
	Busy, Ready bool
	Error       string
	Apply       func()
	Retry       func()
	cancel      context.CancelFunc
	client      *DraftClient
	revision    uint64
	selection   ui.TextInputRange
	editSeq     float64
}

func (d *DraftState) FileIntake() *FileIntake { return d.fileIntake }

func (d *DraftState) BeginFileIntake(paths []string, regularOnly bool, cancel context.CancelFunc) *FileIntake {
	if d == nil || d.Sending || d.Preparing > 0 || len(paths) == 0 {
		if cancel != nil {
			cancel()
		}
		return nil
	}
	if d.Client == nil {
		d.Client = NewDraftClient(&d.Text)
	}
	op := &FileIntake{PendingFiles: PendingFiles{append([]string(nil), paths...), regularOnly}, Busy: true,
		cancel: cancel, client: d.Client, revision: d.Client.Revision(), selection: d.Client.Selection().Range, editSeq: d.EditSeq}
	d.fileIntake = op
	d.Preparing++
	d.Error = ""
	return op
}

// CompleteFileIntake keeps a ready result attached to its draft if the user
// edited or moved the caret. An explicit Apply then uses the current selection.
func (d *DraftState) CompleteFileIntake(op *FileIntake, err error) bool {
	if d.fileIntake != op {
		return false
	}
	op.Busy = false
	if op.cancel != nil {
		op.cancel()
		op.cancel = nil
	}
	if err != nil {
		op.Error = errcopy.Sentence(err)
		op.Ready = false
		return false
	}
	op.Ready = true
	if d.Client != op.client || d.EditSeq != op.editSeq || d.Client.Revision() != op.revision || d.Client.Selection().Range != op.selection {
		return false
	}
	_, marked := d.Client.MarkedRange()
	return !marked
}

// ApplyFileIntake performs one undoable reference insertion and then publishes
// uploads as a single UI-owned transition. No file access occurs here.
func (d *DraftState) ApplyFileIntake(op *FileIntake, references []string, uploads []session.ComposerAttachment) ([]int, bool) {
	if d.fileIntake != op || !op.Ready || d.Sending || d.EditSeq != op.editSeq {
		return nil, false
	}
	if len(references) > 0 && !d.InsertPreparedReferences(references) {
		op.Error = composerText("请先完成输入法输入，再添加文件引用。", "Finish composing text before adding file references.")
		return nil, false
	}
	indices := make([]int, 0, len(uploads))
	for _, attachment := range uploads {
		indices = append(indices, len(d.Attachments))
		d.Attachments = append(d.Attachments, attachment)
	}
	d.CancelFileIntake()
	return indices, true
}

func (d *DraftState) CancelFileIntake() {
	if d == nil || d.fileIntake == nil {
		return
	}
	if d.fileIntake.cancel != nil {
		d.fileIntake.cancel()
	}
	d.fileIntake = nil
	if d.Preparing > 0 {
		d.Preparing--
	}
}

// InsertPreparedReferences uses the current native text selection and undo
// owner. Callers that prepare asynchronously must first validate their ticket.
func (d *DraftState) InsertPreparedReferences(references []string) bool {
	if len(references) == 0 {
		return true
	}
	if d.Client == nil {
		d.Client = NewDraftClient(&d.Text)
	}
	if _, marked := d.Client.MarkedRange(); marked {
		return false
	}
	selection := d.Client.Selection().Range
	start, end := ui.UTF16ByteOffset(d.Text, selection.Start), ui.UTF16ByteOffset(d.Text, selection.End)
	replacement := strings.Join(references, " ")
	if start > 0 {
		r, _ := utf8.DecodeLastRuneInString(d.Text[:start])
		if !unicode.IsSpace(r) {
			replacement = " " + replacement
		}
	}
	if end == len(d.Text) {
		replacement += " "
	} else {
		r, _ := utf8.DecodeRuneInString(d.Text[end:])
		if !unicode.IsSpace(r) {
			replacement += " "
		}
	}
	d.Client.ReplaceText(&selection, replacement)
	d.EditFocus = true
	return true
}

func paintFileIntake(c *ui.Context, d *DraftState, muted ui.Color) {
	op := d.FileIntake()
	if op == nil {
		return
	}
	label := composerText("正在准备文件…", "Preparing files…")
	if !op.Busy {
		label = composerText("文件已保留，确认后添加到输入框。", "Files retained. Add them when you are ready.")
	}
	if op.Error != "" {
		label = op.Error
	}
	ui.Row(c.Key("file-intake")).FillWidth().Gap(8).Padding(4, 14).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(muted).Grow(1)
		if op.Ready && op.Apply != nil && ui.Button(c, composerText("添加文件", "Add files")).Clicked() {
			op.Apply()
		} else if !op.Busy && !op.Ready && op.Retry != nil && ui.Button(c, composerText("重试", "Retry")).Clicked() {
			op.Retry()
		}
		if ui.Button(c, composerText("取消", "Cancel")).Clicked() {
			d.CancelFileIntake()
		}
	})
}
