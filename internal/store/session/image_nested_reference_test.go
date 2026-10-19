package session

import (
	"encoding/json"
	"testing"
)

func TestUserImageStandardNestedReferenceAndLegacyShapes(t *testing.T) {
	for _, tt := range []struct{ name, block, want string }{
		{"current nested reference", `{"type":"image","attachment":{"attachmentId":"owned-image","mediaType":"image/png","bytes":123,"width":96,"height":64,"name":"display-only.png"}}`, "attachment:owned-image"},
		{"legacy direct reference", `{"type":"image","attachmentId":"legacy-image"}`, "attachment:legacy-image"},
		{"legacy direct path", `{"type":"image","path":"C:/owned/legacy.png"}`, "C:/owned/legacy.png"},
		{"non image", `{"type":"file","attachment":{"attachmentId":"not-image"}}`, ""},
		{"missing image discriminator", `{"attachment":{"attachmentId":"not-image"}}`, ""},
		{"nested name is not an identifier", `{"type":"image","attachment":{"name":"C:/not-an-id.png"}}`, ""},
		{"empty identifier", `{"type":"image","attachment":{"attachmentId":"  "}}`, ""},
		{"numeric identifier", `{"type":"image","attachment":{"attachmentId":42}}`, ""},
		{"string attachment", `{"type":"image","attachment":"owned-image"}`, ""},
		{"array attachment", `{"type":"image","attachment":[{"attachmentId":"owned-image"}]}`, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := imagePathFromBlock(json.RawMessage(tt.block)); got != tt.want {
				t.Fatalf("reference=%q want=%q", got, tt.want)
			}
		})
	}
	raw := json.RawMessage(`{"type":"user/message","seq":5,"data":{"id":"owned-user","role":"user","source":{"kind":"user"},"content":[{"type":"text","text":"Owned image attachment"},{"type":"image","attachment":{"attachmentId":"owned-image","mediaType":"image/png","bytes":123,"width":96,"height":64}}]}}`)
	message, ok := classifyPersistenceEvent(raw)
	if !ok || message.Body != "Owned image attachment\n[image] attachment:owned-image" {
		t.Fatalf("saved user image lost: %+v classified=%v", message, ok)
	}
	refs := ImageRefs(message.Body)
	if len(refs) != 1 || refs[0] != "attachment:owned-image" {
		t.Fatalf("renderer image references=%v", refs)
	}
}
