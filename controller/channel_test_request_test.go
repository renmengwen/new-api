package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
)

func TestDefaultChannelTestImageSize(t *testing.T) {
	t.Parallel()

	volcengineChannel := &model.Channel{Type: constant.ChannelTypeVolcEngine}
	if got := defaultChannelTestImageSize("doubao-seedream-5-0-260128", volcengineChannel); got != "2048x2048" {
		t.Fatalf("defaultChannelTestImageSize() = %s, want 2048x2048", got)
	}

	openAIChannel := &model.Channel{Type: constant.ChannelTypeOpenAI}
	if got := defaultChannelTestImageSize("gpt-image-1", openAIChannel); got != "1024x1024" {
		t.Fatalf("defaultChannelTestImageSize() = %s, want 1024x1024", got)
	}
}
