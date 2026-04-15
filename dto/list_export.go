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
