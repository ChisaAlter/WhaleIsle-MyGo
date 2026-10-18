package projectflow

import (
	"encoding/json"
	"fmt"
)

type ProcessRecord struct {
	Type  string `json:"type"`
	Event struct {
		Type string          `json:"type"`
		Seq  float64         `json:"seq"`
		Data json.RawMessage `json:"data"`
	} `json:"event"`
}
type ProcessSnapshot struct {
	Live      bool            `json:"-"`
	Truncated bool            `json:"-"`
	Error     string          `json:"-"`
	Type      string          `json:"type"`
	Cursor    float64         `json:"cursor"`
	Records   []ProcessRecord `json:"records"`
	HasMore   bool            `json:"hasMore"`
}

func ParseProcessSnapshot(raw []byte) (ProcessSnapshot, error) {
	var out ProcessSnapshot
	err := json.Unmarshal(raw, &out)
	if err == nil && (out.Type != "snapshot" || out.Records == nil) {
		err = fmt.Errorf("invalid member history opening snapshot")
	}
	return out, err
}
