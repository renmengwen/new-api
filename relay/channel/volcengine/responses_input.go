package volcengine

import (
	"encoding/json"

	"github.com/QuantumNous/new-api/common"
)

func normalizeResponsesInputForVolcEngine(input json.RawMessage) (json.RawMessage, error) {
	if len(input) == 0 || common.GetJsonType(input) != "array" {
		return input, nil
	}

	var items []map[string]json.RawMessage
	if err := common.Unmarshal(input, &items); err != nil {
		return nil, err
	}

	changed := false
	for i := range items {
		content, ok := items[i]["content"]
		if !ok || len(content) == 0 || common.GetJsonType(content) != "string" {
			continue
		}

		var text string
		if err := common.Unmarshal(content, &text); err != nil {
			return nil, err
		}

		typedContent, err := common.Marshal([]map[string]any{
			{
				"type": "input_text",
				"text": text,
			},
		})
		if err != nil {
			return nil, err
		}

		items[i]["content"] = typedContent
		changed = true
	}

	if !changed {
		return input, nil
	}

	normalized, err := common.Marshal(items)
	if err != nil {
		return nil, err
	}
	return normalized, nil
}
