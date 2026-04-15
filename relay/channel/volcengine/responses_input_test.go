package volcengine

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestNormalizeResponsesInputWrapsStringContent(t *testing.T) {
	input := json.RawMessage(`[{"role":"user","content":"hello"}]`)

	normalized, err := normalizeResponsesInputForVolcEngine(input)
	require.NoError(t, err)
	require.JSONEq(t, `[{"role":"user","content":[{"type":"input_text","text":"hello"}]}]`, string(normalized))
}

func TestNormalizeResponsesInputKeepsTypedContent(t *testing.T) {
	input := json.RawMessage(`[{"role":"user","content":[{"type":"input_text","text":"hello"}]}]`)

	normalized, err := normalizeResponsesInputForVolcEngine(input)
	require.NoError(t, err)
	require.Equal(t, input, normalized)
}

func TestConvertOpenAIResponsesRequestNormalizesInputForVolcEngine(t *testing.T) {
	a := &Adaptor{}
	req := dto.OpenAIResponsesRequest{
		Model: "doubao-seed",
		Input: json.RawMessage(`[{"role":"user","content":"hello"}]`),
	}

	got, err := a.ConvertOpenAIResponsesRequest(nil, &relaycommon.RelayInfo{}, req)
	require.NoError(t, err)

	out, ok := got.(dto.OpenAIResponsesRequest)
	require.True(t, ok)
	require.JSONEq(t, `[{"role":"user","content":[{"type":"input_text","text":"hello"}]}]`, string(out.Input))
}
