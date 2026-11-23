package files

import (
	"context"
	"errors"
	"fmt"
	"strconv"
)

// Owner pointers are identity tokens only. Workers never dereference UI state.
type imageReadOwner struct {
	tab                          *FileTab
	binding, session, root, path string
	epoch                        uint64
}

// stopImageRead is called on the UI thread. Unfinished previews release their
// lease (a queued decode never runs) and are removed; completed pictures stay
// in the shared media cache and per-tab view/draft state survives.
func (t *Tabs) stopImageRead(closeQueue bool) {
	if t == nil {
		return
	}
	for _, w := range t.Workspaces {
		if w == nil {
			continue
		}
		for _, tab := range w.Open {
			tab.stopMarkdownImages()
			if tab.image != nil && tab.image.loading {
				tab.image.release()
				tab.image = nil
			}
		}
	}
	if t.imageReadTab != nil && t.imageReadTab.image != nil && t.imageReadTab.image.loading {
		t.imageReadTab.image.release()
		t.imageReadTab.image = nil
	}
	t.imageReadTab = nil
}

func imageOwner(m Model, tab *FileTab) imageReadOwner {
	return imageReadOwner{tab: tab, binding: m.BindingID, session: m.SessionID, root: m.WorkspaceRoot, path: tab.Path, epoch: m.ImageGeneration}
}

// mediaID names a tab image's content source: the binding, session, root,
// path and generation that authorize its read, and the delivery declaration
// when one owns the bytes. Equal IDs share one decoded picture.
func mediaID(owner imageReadOwner, delivery any) string {
	id := "files\x00" + owner.binding + "\x00" + owner.session + "\x00" + owner.root + "\x00" + owner.path + "\x00" + strconv.FormatUint(owner.epoch, 10)
	if delivery != nil {
		id += fmt.Sprintf("\x00delivery:%p", delivery)
	}
	return id
}

// Painting only prepares the request; file reads, validation and decoding
// run on media workers under the tab's lease.
func readImagePreview(m Model, tab *FileTab) *imagePreview {
	if m.Tabs == nil || tab == nil {
		return &imagePreview{err: errors.New("Image preview requires a resource tab")}
	}
	t := m.Tabs
	owner := imageOwner(m, tab)
	if tab.image != nil && tab.image.owner != owner {
		tab.image.release()
		tab.image = nil
	}
	if tab.image == nil {
		root, path := owner.root, owner.path
		resolve := previewPathResolver(m, tab)
		localSource := tab.deliverySource
		readRemote := m.ReadImageContext
		sourceCurrent := localSource == nil || localSource.current(m)
		var delivery any
		if localSource != nil {
			delivery = localSource
		}
		read := func(ctx context.Context) ([]byte, error) {
			if !sourceCurrent {
				return nil, deliverySourceChanged()
			}
			if localSource != nil {
				return localSource.readBytes(ctx, imageByteLimit(path))
			}
			if readRemote != nil {
				return readRemote(ctx, path, imageByteLimit(path))
			}
			if _, err := resolve(); err != nil {
				return nil, err
			}
			return readWorkspaceImage(ctx, root, path)
		}
		retainImagePreview(m, tab, newImagePreview(owner, mediaID(owner, delivery), path, m.notify, read))
		t.imageReadTab = tab
	}
	tab.image.refresh()
	return tab.image
}
