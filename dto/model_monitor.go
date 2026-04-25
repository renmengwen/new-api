package dto

type ModelMonitorSettingsUpdateRequest struct {
	Enabled               bool                                    `json:"enabled"`
	IntervalMinutes       int                                     `json:"interval_minutes"`
	BatchSize             int                                     `json:"batch_size"`
	DefaultTimeoutSeconds int                                     `json:"default_timeout_seconds"`
	FailureThreshold      int                                     `json:"failure_threshold"`
	ExcludedModelPatterns []string                                `json:"excluded_model_patterns"`
	ModelOverrides        map[string]ModelMonitorModelOverrideDTO `json:"model_overrides"`
}

type ModelMonitorModelOverrideDTO struct {
	Enabled        *bool `json:"enabled,omitempty"`
	TimeoutSeconds int   `json:"timeout_seconds,omitempty"`
}

type ModelMonitorResponse struct {
	Settings ModelMonitorSettingsUpdateRequest `json:"settings"`
	Summary  ModelMonitorSummary               `json:"summary"`
	Items    []ModelMonitorItem                `json:"items"`
}

type ModelMonitorSummary struct {
	TotalModels       int   `json:"total_models"`
	HealthyModels     int   `json:"healthy_models"`
	PartialModels     int   `json:"partial_models"`
	UnavailableModels int   `json:"unavailable_models"`
	SkippedModels     int   `json:"skipped_models"`
	TotalChannels     int   `json:"total_channels"`
	FailedChannels    int   `json:"failed_channels"`
	EnabledModels     int   `json:"enabled_models"`
	DisabledModels    int   `json:"disabled_models"`
	SuccessCount      int   `json:"success_count"`
	FailedCount       int   `json:"failed_count"`
	TimeoutCount      int   `json:"timeout_count"`
	SkippedCount      int   `json:"skipped_count"`
	LastTestedAt      int64 `json:"last_tested_at"`
}

type ModelMonitorItem struct {
	ModelName           string                    `json:"model_name"`
	Enabled             bool                      `json:"enabled"`
	TimeoutSeconds      int                       `json:"timeout_seconds"`
	Status              string                    `json:"status"`
	TestedAt            int64                     `json:"tested_at"`
	ConsecutiveFailures int                       `json:"consecutive_failures"`
	ChannelCount        int                       `json:"channel_count"`
	SuccessCount        int                       `json:"success_count"`
	FailedCount         int                       `json:"failed_count"`
	SkippedCount        int                       `json:"skipped_count"`
	Channels            []ModelMonitorChannelItem `json:"channels"`
}

type ModelMonitorChannelItem struct {
	ChannelId           int    `json:"channel_id"`
	ChannelName         string `json:"channel_name"`
	ChannelType         int    `json:"channel_type"`
	Status              string `json:"status"`
	ResponseTimeMs      int    `json:"response_time_ms"`
	ErrorMessage        string `json:"error_message"`
	TestedAt            int64  `json:"tested_at"`
	ConsecutiveFailures int    `json:"consecutive_failures"`
}
