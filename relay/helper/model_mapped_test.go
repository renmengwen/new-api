package helper

import "testing"

func TestResolveMappedModelName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		originModel  string
		modelMapping string
		wantModel    string
		wantMapped   bool
		wantErr      string
	}{
		{
			name:         "empty mapping keeps original model",
			originModel:  "alias",
			modelMapping: "",
			wantModel:    "alias",
			wantMapped:   false,
		},
		{
			name:         "direct mapping resolves upstream model",
			originModel:  "alias",
			modelMapping: `{"alias":"doubao-seed-translation-250915"}`,
			wantModel:    "doubao-seed-translation-250915",
			wantMapped:   true,
		},
		{
			name:         "chain mapping resolves final upstream model",
			originModel:  "alias",
			modelMapping: `{"alias":"middle","middle":"doubao-seed-1-6-thinking-250715"}`,
			wantModel:    "doubao-seed-1-6-thinking-250715",
			wantMapped:   true,
		},
		{
			name:         "self mapping keeps original model unmapped",
			originModel:  "alias",
			modelMapping: `{"alias":"alias"}`,
			wantModel:    "alias",
			wantMapped:   false,
		},
		{
			name:         "cyclic mapping returns error",
			originModel:  "alias",
			modelMapping: `{"alias":"middle","middle":"alias"}`,
			wantErr:      "model_mapping_contains_cycle",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotModel, gotMapped, err := ResolveMappedModelName(tt.originModel, tt.modelMapping)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("ResolveMappedModelName() error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveMappedModelName() error = %v", err)
			}
			if gotModel != tt.wantModel {
				t.Fatalf("ResolveMappedModelName() model = %q, want %q", gotModel, tt.wantModel)
			}
			if gotMapped != tt.wantMapped {
				t.Fatalf("ResolveMappedModelName() mapped = %t, want %t", gotMapped, tt.wantMapped)
			}
		})
	}
}
