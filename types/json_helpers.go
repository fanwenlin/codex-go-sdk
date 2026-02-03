package types

import (
	"encoding/json"
)

func decodeString(raw map[string]json.RawMessage, keys ...string) string {
	for _, key := range keys {
		if value, ok := raw[key]; ok {
			var s string
			if err := json.Unmarshal(value, &s); err == nil {
				return s
			}
		}
	}
	return ""
}

func decodeStringPtr(raw map[string]json.RawMessage, keys ...string) *string {
	for _, key := range keys {
		if value, ok := raw[key]; ok {
			var s string
			if err := json.Unmarshal(value, &s); err == nil {
				return &s
			}
		}
	}
	return nil
}

func decodeIntPtr(raw map[string]json.RawMessage, keys ...string) *int {
	for _, key := range keys {
		if value, ok := raw[key]; ok {
			var i int
			if err := json.Unmarshal(value, &i); err == nil {
				return &i
			}
		}
	}
	return nil
}

func decodeBoolPtr(raw map[string]json.RawMessage, keys ...string) *bool {
	for _, key := range keys {
		if value, ok := raw[key]; ok {
			var b bool
			if err := json.Unmarshal(value, &b); err == nil {
				return &b
			}
		}
	}
	return nil
}

func decodeAny(raw map[string]json.RawMessage, keys ...string) interface{} {
	for _, key := range keys {
		if value, ok := raw[key]; ok {
			var v interface{}
			if err := json.Unmarshal(value, &v); err == nil {
				return v
			}
		}
	}
	return nil
}
