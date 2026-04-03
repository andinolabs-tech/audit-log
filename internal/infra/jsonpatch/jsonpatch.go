package jsonpatch

import (
	"encoding/json"

	"github.com/wI2L/jsondiff"
)

// DiffMaps returns RFC 6902 JSON Patch operations from before to after as
// map[string]any with key "operations" holding the patch array (JSON types).
func DiffMaps(before, after map[string]any) (map[string]any, error) {
	patch, err := jsondiff.Compare(before, after)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(patch)
	if err != nil {
		return nil, err
	}
	var ops []any
	if err := json.Unmarshal(raw, &ops); err != nil {
		return nil, err
	}
	if ops == nil {
		ops = []any{}
	}
	return map[string]any{"operations": ops}, nil
}
