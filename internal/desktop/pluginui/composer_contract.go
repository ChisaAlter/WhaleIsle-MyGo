package pluginui

import (
	"context"
	"io"
)

// ComposerBinding is minted by the native owner from its captured target, not
// from a page URL. Revision is the native DraftClient revision at capture.
type ComposerBinding struct {
	Subject  Subject `json:"subject"`
	Binding  string  `json:"binding"`
	Revision uint64  `json:"revision"`
	TargetID string  `json:"targetId"`
	EditSeq  float64 `json:"editSeq"`
}

// ComposerDraft contains names and opaque preparation IDs, never local paths,
// Host upload receipts, credentials, or another session's saved draft.
type ComposerDraft struct {
	Owner          ComposerBinding `json:"owner"`
	Text           string          `json:"text"`
	Attachments    []ComposerFile  `json:"attachments"`
	Sending        bool            `json:"sending"`
	Preparing      bool            `json:"preparing"`
	SelectionStart int             `json:"selectionStart"`
	SelectionEnd   int             `json:"selectionEnd"`
}
type ComposerFile struct {
	ID        uint64 `json:"id"`
	Name      string `json:"name"`
	Image     bool   `json:"image"`
	Error     string `json:"error"`
	Size      int64  `json:"size,omitempty"`
	MediaType string `json:"mediaType"`
	Loaded    int64  `json:"loaded,omitempty"`
	Preparing bool   `json:"preparing,omitempty"`
}
type ComposerReplacement struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Text  string `json:"text"`
}
type ComposerEdit struct {
	Owner        ComposerBinding      `json:"owner"`
	Text         *string              `json:"text,omitempty"`
	Replace      *ComposerReplacement `json:"replace,omitempty"`
	RemoveFileID uint64               `json:"removeFileId,omitempty"`
	RetryFileID  uint64               `json:"retryFileId,omitempty"`
	PickFiles    bool                 `json:"pickFiles,omitempty"`
	PickImage    bool                 `json:"pickImage,omitempty"`
}
type ComposerGesture struct {
	Owner       ComposerBinding `json:"owner"`
	Accelerated bool            `json:"accelerated"`
}

// Apply and Submit recheck the entire binding and draft revision before using
// DraftState / CaptureComposerGestureTarget. Stale actions change nothing.
type ComposerBridge interface {
	Read(Subject) (ComposerDraft, error)
	Apply(ComposerEdit) error
	Submit(ComposerGesture) error
}

// ComposerFileContent is an ordered, one-shot file stream. It carries a display
// name and media type, never a browser registry ID or a native filesystem path.
// Size is -1 when the transport cannot know it before reading the stream.
type ComposerFileContent struct {
	Name, MediaType string
	Size            int64
	Content         io.ReadCloser
}

type ComposerFileIterator interface {
	Next() (ComposerFileContent, error) // io.EOF ends the batch.
}

// ComposerFileTransfer is optional for owners without attachment support. The
// receiver captures one binding/target before reading, streams into the existing
// attachment cache, then validates it again before appending the batch. Failed
// or expired imports clean their uncommitted cache entries. ReadFile authorizes
// only an extant attachment's opaque PreparationID under this exact binding.
type ComposerFileTransfer interface {
	ImportFiles(context.Context, ComposerBinding, ComposerFileIterator) error
	ReadFile(context.Context, ComposerBinding, uint64) (ComposerFileContent, error)
}
