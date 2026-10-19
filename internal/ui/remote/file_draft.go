package remote

import (
	"sync"

	"github.com/egoist/mygo/ui"
	remotestore "whaleisle/internal/store/remote"
)

// A buffer is retained by its store resource, independently of the currently
// painted editor or selected session. Store capture never writes a later owner.
type fileTextBuffer struct {
	buffer  *ui.TextBuffer
	mu      sync.Mutex
	text    string
	version uint64
}

func newFileTextBuffer(text string) remotestore.FileDraftBuffer {
	buffer := ui.NewTextBuffer(text)
	return &fileTextBuffer{buffer: buffer, text: text, version: buffer.Version()}
}

func (b *fileTextBuffer) CommittedText() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	snapshot := b.buffer.CommittedSnapshot()
	if snapshot.Version() != b.version {
		b.text, b.version = snapshot.String(), snapshot.Version()
	}
	return b.text
}

func (b *fileTextBuffer) ReplaceText(text string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buffer.Set(text)
	b.text, b.version = text, b.buffer.Version()
}
