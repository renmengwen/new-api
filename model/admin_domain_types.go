package model

const (
	UserTypeRoot    = "root"
	UserTypeAdmin   = "admin"
	UserTypeAgent   = "agent"
	UserTypeEndUser = "end_user"
)

const (
	CommonStatusDisabled = 0
	CommonStatusEnabled  = 1
)

const (
	ScopeTypeAll       = "all"
	ScopeTypeSelf      = "self"
	ScopeTypeAgentOnly = "agent_only"
	ScopeTypeAssigned  = "assigned"
)

const (
	TransferTypeAdminAdjust   = "admin_adjust"
	TransferTypeAgentRecharge = "agent_recharge"
	TransferTypeAgentReclaim  = "agent_reclaim"
)

const (
	QuotaOwnerTypeUser   = "user"
	QuotaOwnerTypeSystem = "system"
)

const (
	LedgerEntryOpening    = "opening"
	LedgerEntryAdjust     = "adjust"
	LedgerEntryRecharge   = "recharge"
	LedgerEntryReclaim    = "reclaim"
	LedgerEntryConsume    = "consume"
	LedgerEntryRefund     = "refund"
	LedgerEntryCommission = "commission"
	LedgerEntryReward     = "reward"
)

const (
	LedgerDirectionIn  = "in"
	LedgerDirectionOut = "out"
)

const (
	PermissionEffectAllow = "allow"
	PermissionEffectDeny  = "deny"
	MenuEffectShow        = "show"
	MenuEffectHide        = "hide"
)
