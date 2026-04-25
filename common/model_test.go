package common

import "testing"

func TestIsImageGenerationModelRecognizesGPTImage2Aliases(t *testing.T) {
	tests := []string{
		"gpt-image-2",
		"gpt-image-2-plus",
	}

	for _, model := range tests {
		if !IsImageGenerationModel(model) {
			t.Fatalf("expected %q to be recognized as an image generation model", model)
		}
	}
}
