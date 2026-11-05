package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// UnmarshalJSON admits the published projection union and the earlier Host's
// kind-less sequenced shape. A cached migration hint never acquires a sequence.
func (v *SessionProjectionsValue) UnmarshalJSON(raw []byte) error {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		*v = SessionProjectionsValue{}
		return nil
	}
	var wire struct {
		Kind    string          `json:"kind"`
		AsOfSeq *float64        `json:"asOfSeq"`
		Values  json.RawMessage `json:"values"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil {
		return err
	}
	values := bytes.TrimSpace(wire.Values)
	if len(values) == 0 || values[0] != '{' {
		return errors.New("Session projections require an object of values")
	}
	next := SessionProjectionsValue{Kind: wire.Kind, Values: wire.Values}
	switch wire.Kind {
	case "", "sequenced":
		if wire.AsOfSeq == nil {
			return errors.New("Sequenced Session projections require asOfSeq")
		}
		next.AsOfSeq = *wire.AsOfSeq
	case "migration-required":
		// asOfSeq, if supplied as an unknown field, is not authority in this arm.
	default:
		return fmt.Errorf("Unknown Session projection kind %q", wire.Kind)
	}
	*v = next
	return nil
}

func (v SessionProjectionsValue) MarshalJSON() ([]byte, error) {
	if v.Kind == "" && v.AsOfSeq == 0 && len(v.Values) == 0 {
		return []byte("null"), nil
	}
	if v.Kind == "migration-required" {
		return json.Marshal(struct {
			Kind   string          `json:"kind"`
			Values json.RawMessage `json:"values"`
		}{v.Kind, v.Values})
	}
	type sequenced SessionProjectionsValue
	return json.Marshal(sequenced(v))
}

// Sequenced is also false for the zero value left by a null RPC response.
func (v SessionProjectionsValue) Sequenced() bool {
	return (v.Kind == "" || v.Kind == "sequenced") && len(v.Values) > 0
}

// RequireSequenced is used when values authorize editing or confirm a mutation.
func (v SessionProjectionsValue) RequireSequenced() error {
	if v.Sequenced() {
		return nil
	}
	if v.Kind == "migration-required" {
		return errors.New("Current Session state is unavailable until migration is complete")
	}
	return errors.New("The Host did not return current Session projections")
}
