package claude

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

const (
	opus47ModelPrefix     = "claude-opus-4-7"
	opus47ThinkingDisplay = "summarized"
	taskBudgetBetaHeader  = "task-budgets-2026-03-13"
)

func IsOpus47Model(model string) bool {
	return strings.HasPrefix(model, opus47ModelPrefix)
}

func NormalizeOpus47Request(request *dto.ClaudeRequest, effort string) error {
	if request == nil || !IsOpus47Model(request.Model) {
		return nil
	}

	request.Temperature = nil
	request.TopP = nil
	request.TopK = nil

	effort = normalizeOpus47Effort(request, effort)
	if effort != "" && effort != GetOutputConfigEffort(request.OutputConfig) {
		outputConfig, err := MergeOutputConfigEffort(request.OutputConfig, effort)
		if err != nil {
			return err
		}
		request.OutputConfig = outputConfig
	}

	if request.Thinking == nil {
		if effort == "" {
			return nil
		}
		request.Thinking = &dto.Thinking{}
	}

	request.Thinking.Type = "adaptive"
	request.Thinking.BudgetTokens = nil
	if request.Thinking.Display == nil {
		request.Thinking.Display = common.GetPointer(opus47ThinkingDisplay)
	}
	return nil
}

func normalizeOpus47Effort(request *dto.ClaudeRequest, effort string) string {
	effort = strings.TrimSpace(effort)
	if effort != "" {
		return effort
	}

	effort = GetOutputConfigEffort(request.OutputConfig)
	if effort != "" {
		return effort
	}

	if request.Thinking != nil && request.Thinking.BudgetTokens != nil {
		return opus47EffortFromBudgetTokens(*request.Thinking.BudgetTokens)
	}
	return ""
}

func opus47EffortFromBudgetTokens(budgetTokens int) string {
	switch {
	case budgetTokens <= 0:
		return ""
	case budgetTokens <= 1280:
		return "low"
	case budgetTokens <= 2048:
		return "medium"
	case budgetTokens <= 4096:
		return "high"
	default:
		return "xhigh"
	}
}

func MergeOutputConfigEffort(raw json.RawMessage, effort string) (json.RawMessage, error) {
	effort = strings.TrimSpace(effort)
	if effort == "" {
		return raw, nil
	}

	outputConfig, err := outputConfigMap(raw)
	if err != nil {
		return nil, err
	}
	outputConfig["effort"] = effort

	data, err := common.Marshal(outputConfig)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func GetOutputConfigEffort(raw json.RawMessage) string {
	outputConfig, err := outputConfigMap(raw)
	if err != nil {
		return ""
	}
	effort, _ := outputConfig["effort"].(string)
	return strings.TrimSpace(effort)
}

func OutputConfigHasTaskBudget(raw json.RawMessage) bool {
	outputConfig, err := outputConfigMap(raw)
	if err != nil {
		return false
	}
	_, ok := outputConfig["task_budget"]
	return ok
}

func RequestBodyHasTaskBudget(raw []byte) bool {
	var request struct {
		OutputConfig json.RawMessage `json:"output_config"`
	}
	if err := common.Unmarshal(raw, &request); err != nil {
		return false
	}
	return OutputConfigHasTaskBudget(request.OutputConfig)
}

func outputConfigMap(raw json.RawMessage) (map[string]any, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return map[string]any{}, nil
	}

	var outputConfig map[string]any
	if err := common.Unmarshal(trimmed, &outputConfig); err != nil {
		return nil, err
	}
	if outputConfig == nil {
		outputConfig = map[string]any{}
	}
	return outputConfig, nil
}
