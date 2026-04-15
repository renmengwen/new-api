package dto

type SeedanceVideoTaskCreateResponse struct {
	ID string `json:"id"`
}

type SeedanceVideoTaskDeleteResponse struct{}

type SeedanceVideoTask struct {
	ID              string                    `json:"id"`
	Model           string                    `json:"model,omitempty"`
	Status          string                    `json:"status,omitempty"`
	Content         *SeedanceVideoTaskContent `json:"content,omitempty"`
	Seed            int                       `json:"seed,omitempty"`
	Resolution      string                    `json:"resolution,omitempty"`
	Duration        int                       `json:"duration,omitempty"`
	Ratio           string                    `json:"ratio,omitempty"`
	FramesPerSecond int                       `json:"framespersecond,omitempty"`
	ServiceTier     string                    `json:"service_tier,omitempty"`
	Usage           *SeedanceVideoTaskUsage   `json:"usage,omitempty"`
	CreatedAt       int64                     `json:"created_at,omitempty"`
	UpdatedAt       int64                     `json:"updated_at,omitempty"`
}

type SeedanceVideoTaskContent struct {
	VideoURL string `json:"video_url,omitempty"`
}

type SeedanceVideoTaskUsage struct {
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

type SeedanceVideoTaskListResponse struct {
	Total int64                `json:"total"`
	Items []*SeedanceVideoTask `json:"items"`
}
