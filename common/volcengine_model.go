package common

import "strings"

var volcEngineResponsesModelPrefixes = []string{
	"doubao-seed-translation-",
	"doubao-seed-1-6-thinking-",
}

func IsVolcEngineResponsesModel(model string) bool {
	target := strings.ToLower(strings.TrimSpace(model))
	if target == "" {
		return false
	}

	for _, prefix := range volcEngineResponsesModelPrefixes {
		if strings.HasPrefix(target, prefix) {
			return true
		}
	}
	return false
}
