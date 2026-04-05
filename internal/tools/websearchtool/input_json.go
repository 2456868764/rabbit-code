package websearchtool

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// DecodeInputStrictJSON parses input like WebSearchTool z.strictObject (reject unknown JSON keys).
func DecodeInputStrictJSON(data []byte) (Input, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var in Input
	if err := dec.Decode(&in); err != nil {
		return Input{}, fmt.Errorf("websearchtool: invalid json: %w", err)
	}
	if dec.More() {
		return Input{}, fmt.Errorf("websearchtool: invalid json: trailing data")
	}
	return in, nil
}
