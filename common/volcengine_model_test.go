package common

import "testing"

func TestIsVolcEngineResponsesModel(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  bool
	}{
		{
			name:  "translation model",
			model: "doubao-seed-translation-en-250101",
			want:  true,
		},
		{
			name:  "translation model with casing and spaces",
			model: "  DouBao-Seed-Translation-EN-250101  ",
			want:  true,
		},
		{
			name:  "thinking model",
			model: "doubao-seed-1-6-thinking-250715",
			want:  true,
		},
		{
			name:  "thinking model with casing and spaces",
			model: "  DouBao-Seed-1-6-ThInKiNg-250715  ",
			want:  true,
		},
		{
			name:  "ordinary chat model",
			model: "Doubao-pro-32k",
			want:  false,
		},
		{
			name:  "empty model",
			model: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsVolcEngineResponsesModel(tt.model); got != tt.want {
				t.Fatalf("IsVolcEngineResponsesModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}
