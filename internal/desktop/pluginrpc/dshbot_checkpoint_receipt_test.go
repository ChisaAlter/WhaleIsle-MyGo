package pluginrpc

import (
	"encoding/json"
	"testing"
)

func TestCreatedReceiptExactCheckpointDefault(t *testing.T) {
	base := schemaReceiptFixture()
	delete(base.Profile, "futureField")
	item, err := decodeDshbotBotItem(schemaReceiptRaw(base.Profile))
	if err != nil {
		t.Fatal(err)
	}
	view := CloneDshbotBotItem(item)
	view.Profile["holds"] = json.RawMessage(`{}`)
	view.Profile["holdCheckpoint"] = json.RawMessage(`{"version":1,"holds":{},"processedEventIds":[],"processedThroughSeq":-1}`)
	if !sameDshbotCreatedItemReceipt(item, view) {
		t.Fatal("exact schema defaults rejected")
	}
	if _, ok := item.Profile["holdCheckpoint"]; ok {
		t.Fatal("receipt comparison mutated source")
	}
	if sameDshbotBotItem(item, view) {
		t.Fatal("CAS equality was broadened")
	}
	for _, bad := range []string{
		`null`, `{}`, `{"version":2,"holds":{},"processedEventIds":[],"processedThroughSeq":-1}`,
		`{"version":1,"holds":{"member":{}},"processedEventIds":[],"processedThroughSeq":-1}`,
		`{"version":1,"holds":{},"processedEventIds":["event"],"processedThroughSeq":-1}`,
		`{"version":1,"holds":{},"processedEventIds":[],"processedThroughSeq":0}`,
		`{"version":1,"holds":{},"processedEventIds":[],"processedThroughSeq":-1,"future":true}`,
	} {
		v := CloneDshbotBotItem(view)
		v.Profile["holdCheckpoint"] = json.RawMessage(bad)
		if sameDshbotCreatedItemReceipt(item, v) {
			t.Fatalf("non-default checkpoint accepted: %s", bad)
		}
	}
	if sameDshbotCreatedItemReceipt(view, item) {
		t.Fatal("reverse omission accepted")
	}
	v := CloneDshbotBotItem(view)
	v.SessionID = "foreign"
	if sameDshbotCreatedItemReceipt(item, v) {
		t.Fatal("foreign identity accepted")
	}
	declared := CloneDshbotBotItem(item)
	declared.Profile["holdCheckpoint"] = json.RawMessage(`null`)
	if sameDshbotCreatedItemReceipt(declared, view) {
		t.Fatal("explicit conflicting checkpoint accepted")
	}
}
