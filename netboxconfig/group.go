package netboxconfig

import (
	"encoding/json"
	"slices"
)

// CollectGroups takes a map of strings to some json decodable object.
// It will sort the keys lexicographically and return only the values
// pointed to by the keys as a slice of json RawMessage structs ready
// for decoding.
//
// Groups exist not as a mapping but as a way to allow appending and
// overriding configuration values in Netbox's hierarchical config
// context system.
func CollectGroups(grouped json.RawMessage) ([]json.RawMessage, error) {
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(grouped, &decoded); err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(decoded))
	for k, _ := range decoded {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	out := []json.RawMessage{}
	for _, k := range keys {
		// Empty lists and nil values should be keys that were deleted (by
		// setting them to empty) when a higher priority config overrides a
		// lower priority one.
		if decoded[k] == nil || len(decoded[k]) == 0 {
			continue
		}
		out = append(out, decoded[k])
	}

	return out, nil
}
