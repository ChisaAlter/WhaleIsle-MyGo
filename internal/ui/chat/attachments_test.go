package chat

import (
	"image"
	"image/color"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/store/session"
)

func TestUserFileAttachmentsReadStructuredContent(t *testing.T) {
	raw := []byte(`{"type":"user/message","seq":3,"data":{"content":[
		{"type":"text","text":"see attached"},
		{"type":"file","attachment":{"attachmentId":"att-1","name":"report.pdf","mediaType":"application/pdf","bytes":1572864}},
		{"type":"image","attachment":{"attachmentId":"att-2","name":"shot.png","mediaType":"image/png","bytes":2048}},
		{"type":"file","attachment":{"attachmentId":"att-3","mediaType":"text/csv"}}
	]}}`)
	files := userFileAttachments(session.Message{Raw: raw})
	if len(files) != 2 {
		t.Fatalf("file parts = %d, want 2 (images render elsewhere)", len(files))
	}
	if files[0].Name != "report.pdf" || files[0].Bytes != 1572864 || files[0].MediaType != "application/pdf" {
		t.Fatalf("first part lost metadata: %+v", files[0])
	}
	if files[1].Name != "att-3" {
		t.Fatalf("nameless part should fall back to its id: %+v", files[1])
	}
	if got := userFileAttachments(session.Message{}); got != nil {
		t.Fatalf("message without raw content produced %+v", got)
	}
	bare := []byte(`{"type":"user/message","seq":4,"data":{"content":[{"type":"text","text":"hi"}]}}`)
	if got := userFileAttachments(session.Message{Raw: bare}); len(got) != 0 {
		t.Fatalf("text-only message produced cards: %+v", got)
	}
}

func TestAttachmentMetaShowsExtensionAndSize(t *testing.T) {
	cases := []struct {
		name      string
		mediaType string
		bytes     float64
		want      string
	}{
		{name: "report.pdf", mediaType: "application/pdf", bytes: 1572864, want: "pdf · 1.5 MB"},
		{name: "data.csv", mediaType: "text/csv", bytes: 2048, want: "csv · 2.0 KB"},
		{name: "Makefile", mediaType: "text/plain", bytes: 12, want: "plain · 12 B"},
		{name: "archive.tar.gz", mediaType: "application/gzip", bytes: 12 * 1024 * 1024, want: "gz · 12 MB"},
		{name: "notes", mediaType: "", bytes: 0, want: ""},
	}
	for _, tc := range cases {
		if got := attachmentMeta(tc.name, tc.mediaType, tc.bytes); got != tc.want {
			t.Errorf("attachmentMeta(%q)=%q want %q", tc.name, got, tc.want)
		}
	}
}

func TestAttachmentCardIsTheOriginal240By64(t *testing.T) {
	file := userFileAttachment{Name: "report.pdf", MediaType: "application/pdf", Bytes: 1572864}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Padding(10).Children(func() {
			attachmentCardView(c, file, ui.RGB(0, 0, 0), ui.RGB(90, 90, 90), ui.RGB(220, 220, 220), ui.RGB(245, 245, 245))
		})
	}, 400, 120)
	card, ok := tt.Find("Attachment report.pdf")
	if !ok {
		t.Fatalf("card missing: %q", tt.Texts())
	}
	if card.W != attachmentCardWidth || card.H != attachmentCardHeight {
		t.Fatalf("card = %vx%v, want %vx%v", card.W, card.H, attachmentCardWidth, attachmentCardHeight)
	}
	name, ok := tt.Find("report.pdf")
	if !ok {
		t.Fatal("file name missing")
	}
	meta, ok := tt.Find("pdf · 1.5 MB")
	if !ok {
		t.Fatalf("meta line missing: %q", tt.Texts())
	}
	if meta.Y <= name.Y {
		t.Fatal("meta line should sit under the name")
	}
}

func TestInlineImageBoundsKeepNaturalRatioWithinClamp(t *testing.T) {
	cases := []struct {
		width, height int
		wantW, wantH  float32
	}{
		{400, 100, 240, 60},
		{100, 100, 240, 240},
		{100, 400, 60, 240},
		{100, 4000, 60, 240}, // taller than 1:4 clamps to the floor
		{4000, 100, 240, 60}, // wider than 4:1 clamps to the ceiling
	}
	for _, tc := range cases {
		img := image.NewRGBA(image.Rect(0, 0, tc.width, tc.height))
		img.Set(0, 0, color.RGBA{A: 255})
		w, h := inlineImageBounds(ui.NewBitmap(img))
		if w != tc.wantW || h != tc.wantH {
			t.Errorf("%dx%d -> %vx%v, want %vx%v", tc.width, tc.height, w, h, tc.wantW, tc.wantH)
		}
	}
}
