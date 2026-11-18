package config

import (
	"encoding/json"
	"reflect"
	"strings"
)

func cloneRawMap(src map[string]json.RawMessage) map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func structJSONFields[T any]() map[string]bool {
	fields := map[string]bool{}
	typ := reflect.TypeFor[T]()
	for i := 0; i < typ.NumField(); i++ {
		name := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
		if name != "" && name != "-" {
			fields[name] = true
		}
	}
	return fields
}

// mergeStructObject encodes next and merges its modeled fields onto any JSON
// object in existingRaw, preserving unknown sibling keys and dropping modeled
// omitempty keys that are omitted in next.
func mergeStructObject[T any](existingRaw json.RawMessage, next T) (json.RawMessage, error) {
	encoded, err := json.Marshal(next)
	if err != nil {
		return nil, err
	}
	return mergeKnownFieldsObject(existingRaw, encoded, structJSONFields[T](), nil)
}

func mergeKnownFieldsObject(existingRaw, nextRaw json.RawMessage, knownFields map[string]bool, nested func(map[string]json.RawMessage, map[string]json.RawMessage) error) (json.RawMessage, error) {
	existing := decodeJSONObjectMap(existingRaw)
	var nextObj map[string]json.RawMessage
	if err := json.Unmarshal(nextRaw, &nextObj); err != nil {
		return nil, err
	}
	if nextObj == nil {
		nextObj = map[string]json.RawMessage{}
	}
	merged := make(map[string]json.RawMessage, len(existing)+len(nextObj))
	for k, v := range existing {
		if !knownFields[k] {
			merged[k] = v
		}
	}
	for k, v := range nextObj {
		merged[k] = v
	}
	if nested != nil {
		if err := nested(existing, merged); err != nil {
			return nil, err
		}
	}
	return json.Marshal(merged)
}

func decodeJSONObjectMap(raw json.RawMessage) map[string]json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	var out map[string]json.RawMessage
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		return nil
	}
	return out
}

func mergeObjectArrayByID(existingArrRaw, nextArrRaw json.RawMessage, knownFields map[string]bool, nested func(existingItem, nextItem map[string]json.RawMessage, mergedItem map[string]json.RawMessage) error) (json.RawMessage, error) {
	var nextItems []json.RawMessage
	if err := json.Unmarshal(nextArrRaw, &nextItems); err != nil {
		return nil, err
	}
	if nextItems == nil {
		return nextArrRaw, nil
	}
	var existingItems []json.RawMessage
	_ = json.Unmarshal(existingArrRaw, &existingItems)

	byID := make(map[string]json.RawMessage, len(existingItems))
	for _, rawItem := range existingItems {
		if id := jsonObjectID(rawItem); id != "" {
			byID[id] = rawItem
		}
	}
	out := make([]json.RawMessage, len(nextItems))
	for i, nextItem := range nextItems {
		id := jsonObjectID(nextItem)
		prevRaw, ok := byID[id]
		if id == "" || !ok {
			out[i] = nextItem
			continue
		}
		mergedItem, err := mergeKnownFieldsObject(prevRaw, nextItem, knownFields, func(existingObj, mergedObj map[string]json.RawMessage) error {
			if nested == nil {
				return nil
			}
			return nested(existingObj, decodeJSONObjectMap(nextItem), mergedObj)
		})
		if err != nil {
			return nil, err
		}
		out[i] = mergedItem
	}
	return json.Marshal(out)
}

func jsonObjectID(raw json.RawMessage) string {
	var head struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(raw, &head) != nil {
		return ""
	}
	return head.ID
}
