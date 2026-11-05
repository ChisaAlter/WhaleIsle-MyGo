package api

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPublishedResultProjectionUnion(t *testing.T) {
	for _, tc := range []struct {
		name, raw, kind string
		seq             float64
		sequenced       bool
	}{
		{"legacy-zero", `{"asOfSeq":0,"values":{}}`, "", 0, true},
		{"legacy", `{"asOfSeq":7,"values":{}}`, "", 7, true},
		{"published", `{"kind":"sequenced","asOfSeq":8,"values":{}}`, "sequenced", 8, true},
		{"cached", `{"kind":"migration-required","values":{"title":"cached"}}`, "migration-required", 0, false},
		{"cached-extra-seq", `{"kind":"migration-required","asOfSeq":99,"values":{}}`, "migration-required", 0, false},
		{"absent", `null`, "", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var v SessionProjectionsValue
			if err := json.Unmarshal([]byte(tc.raw), &v); err != nil {
				t.Fatal(err)
			}
			if v.Kind != tc.kind || v.AsOfSeq != tc.seq || v.Sequenced() != tc.sequenced || (v.RequireSequenced() == nil) != tc.sequenced {
				t.Fatalf("union arm lost: %+v", v)
			}
			encoded, err := json.Marshal(v)
			if err != nil {
				t.Fatal(err)
			}
			var roundtrip SessionProjectionsValue
			if err := json.Unmarshal(encoded, &roundtrip); err != nil || roundtrip.Kind != v.Kind || roundtrip.AsOfSeq != v.AsOfSeq || roundtrip.Sequenced() != v.Sequenced() {
				t.Fatalf("union roundtrip lost: %s %+v %v", encoded, roundtrip, err)
			}
			if tc.raw == "null" && string(encoded) != "null" {
				t.Fatalf("null projection fabricated a baseline: %s", encoded)
			}
			if v.Kind == "migration-required" {
				raw, err := json.Marshal(v)
				if err != nil || strings.Contains(string(raw), "asOfSeq") {
					t.Fatalf("cached hint fabricated sequencing authority: %s %v", raw, err)
				}
			}
		})
	}
	for _, raw := range []string{`{"kind":"sequenced","values":{}}`, `{"values":{}}`, `{"kind":"other","values":{}}`, `{"asOfSeq":1,"values":null}`} {
		var v SessionProjectionsValue
		if json.Unmarshal([]byte(raw), &v) == nil {
			t.Fatalf("non-contract projection accepted: %s", raw)
		}
	}
}

func TestPublishedResultBonusOpaqueMetadata(t *testing.T) {
	raw := `{"accountId":"owned","bonuses":[{"orderId":"a","title":"","campaign":{"id":"owned"},"amount":12.5,"currency":null,"grantedAt":[1,2],"expiresAt":"","message":"Server-authored message"},{"orderId":"b","expiresAt":"","message":"Metadata omitted"}]}`
	var batch GetUnnotifiedBonusesResult
	if err := json.Unmarshal([]byte(raw), &batch); err != nil {
		t.Fatal(err)
	}
	a, b := batch.Bonuses[0], batch.Bonuses[1]
	if a.Title == nil || *a.Title != "" || string(a.Amount) != "12.5" || string(a.Campaign) != `{"id":"owned"}` || string(a.Currency) != "null" || string(a.GrantedAt) != "[1,2]" || b.Title != nil || b.Amount != nil {
		t.Fatalf("opaque/absent/empty metadata conflated: %+v %+v", a, b)
	}
}
