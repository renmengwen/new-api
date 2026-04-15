package dto

type AdminAuditExportRequest struct {
	ActionModule   string `json:"action_module"`
	OperatorUserID int    `json:"operator_user_id"`
	Limit          int    `json:"limit"`
}

type AdminQuotaLedgerExportRequest struct {
	UserID         int    `json:"user_id"`
	OperatorUserID int    `json:"operator_user_id"`
	EntryType      string `json:"entry_type"`
	Limit          int    `json:"limit"`
}

type UsageLogExportRequest struct {
	Type           int      `json:"type"`
	StartTimestamp int64    `json:"start_timestamp"`
	EndTimestamp   int64    `json:"end_timestamp"`
	Username       string   `json:"username"`
	TokenName      string   `json:"token_name"`
	ModelName      string   `json:"model_name"`
	Channel        string   `json:"channel"`
	Group          string   `json:"group"`
	RequestID      string   `json:"request_id"`
	ColumnKeys     []string `json:"column_keys"`
	Limit          int      `json:"limit"`
}
