package main

import (
	"encoding/json"
	"fmt"
)

// Inventory is the machine-readable persistence schema inventory
// (docs/persistence-schema.json, formatVersion 1|2).
type Inventory struct {
	FormatVersion int             `json:"formatVersion"`
	Roots         []PersistenceRoot `json:"roots"`
	Types         []json.RawMessage `json:"types"` // retained for parse checks; unused by emit
}

// PersistenceRoot is one independently tracked persistent record vocabulary.
type PersistenceRoot struct {
	Key     string           `json:"key"`
	Kind    string           `json:"kind"` // header | envelope | event
	Event   string           `json:"event,omitempty"`
	Surface bool             `json:"surface,omitempty"`
	Digest  string           `json:"digest"`
	Schema  CanonicalSchema  `json:"schema"`
}

// CanonicalSchema is a self-contained minimized type graph.
type CanonicalSchema struct {
	Root  int          `json:"root"`
	Nodes []SchemaNode `json:"nodes"`
}

// SchemaNode is one resolved persisted type; numeric refs address nodes in the
// enclosing graph.
type SchemaNode struct {
	Kind string `json:"kind"`

	// primitive
	Type string `json:"type,omitempty"` // null | boolean | number | string | never

	// literal
	Value json.RawMessage `json:"value,omitempty"`

	// opaque
	Reason string `json:"reason,omitempty"` // any | unknown

	// array
	Element *int `json:"element,omitempty"`

	// tuple
	Elements []SchemaTupleElement `json:"elements,omitempty"`

	// object
	Properties []SchemaProperty `json:"properties,omitempty"`
	Indices    []SchemaIndex    `json:"indices,omitempty"`

	// union
	Types []int `json:"types,omitempty"`
}

// SchemaProperty is one JSON object property.
type SchemaProperty struct {
	Name     string `json:"name"`
	Type     int    `json:"type"`
	Optional bool   `json:"optional"`
}

// SchemaIndex is one object index signature (key/value type refs).
type SchemaIndex struct {
	Key   int `json:"key"`
	Value int `json:"value"`
}

// SchemaTupleElement is one positional tuple slot.
type SchemaTupleElement struct {
	Type     int  `json:"type"`
	Optional bool `json:"optional"`
	Rest     bool `json:"rest"`
}

func parseInventory(data []byte) (*Inventory, error) {
	var inv Inventory
	if err := json.Unmarshal(data, &inv); err != nil {
		return nil, fmt.Errorf("decode persistence schema: %w", err)
	}
	if inv.FormatVersion != 1 && inv.FormatVersion != 2 {
		return nil, fmt.Errorf("unsupported formatVersion %d (want 1 or 2)", inv.FormatVersion)
	}
	if len(inv.Roots) == 0 {
		return nil, fmt.Errorf("persistence schema has no roots")
	}
	for i, r := range inv.Roots {
		if r.Key == "" {
			return nil, fmt.Errorf("roots[%d]: missing key", i)
		}
		switch r.Kind {
		case "header", "envelope", "event":
		default:
			return nil, fmt.Errorf("roots[%d] %q: unknown kind %q", i, r.Key, r.Kind)
		}
		if r.Kind == "event" && r.Event == "" {
			return nil, fmt.Errorf("roots[%d] %q: event root missing event field", i, r.Key)
		}
		if len(r.Schema.Nodes) == 0 {
			return nil, fmt.Errorf("roots[%d] %q: empty schema.nodes", i, r.Key)
		}
		if r.Schema.Root < 0 || r.Schema.Root >= len(r.Schema.Nodes) {
			return nil, fmt.Errorf("roots[%d] %q: root index %d out of range", i, r.Key, r.Schema.Root)
		}
		for j, n := range r.Schema.Nodes {
			if n.Kind == "" {
				return nil, fmt.Errorf("roots[%d] %q: nodes[%d] missing kind", i, r.Key, j)
			}
		}
	}
	return &inv, nil
}
