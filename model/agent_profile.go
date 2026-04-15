package model

type AgentProfile struct {
	Id           int    `json:"id"`
	UserId       int    `json:"user_id" gorm:"not null;uniqueIndex:uk_agent_profiles_user_id"`
	AgentName    string `json:"agent_name" gorm:"type:varchar(128);not null;index:idx_agent_profiles_agent_name"`
	CompanyName  string `json:"company_name" gorm:"type:varchar(128);default:''"`
	ContactPhone string `json:"contact_phone" gorm:"type:varchar(32);default:''"`
	Remark       string `json:"remark" gorm:"type:text"`
	Status       int    `json:"status" gorm:"not null;default:1;index:idx_agent_profiles_status"`
	CreatedAtTs  int64  `json:"created_at" gorm:"column:created_at;bigint;not null;default:0"`
	UpdatedAtTs  int64  `json:"updated_at" gorm:"column:updated_at;bigint;not null;default:0"`
}

type AgentUserRelation struct {
	Id          int    `json:"id"`
	AgentUserId int    `json:"agent_user_id" gorm:"not null;uniqueIndex:uk_aur_agent_end_user_status,priority:1;index:idx_aur_agent_user_id_status,priority:1"`
	EndUserId   int    `json:"end_user_id" gorm:"not null;uniqueIndex:uk_aur_agent_end_user_status,priority:2;index:idx_aur_end_user_id"`
	BindSource  string `json:"bind_source" gorm:"type:varchar(32);not null;default:'manual'"`
	BindAt      int64  `json:"bind_at" gorm:"bigint;not null;default:0"`
	UnbindAt    int64  `json:"unbind_at" gorm:"bigint;not null;default:0"`
	Status      int    `json:"status" gorm:"not null;default:1;uniqueIndex:uk_aur_agent_end_user_status,priority:3;index:idx_aur_agent_user_id_status,priority:2"`
	CreatedAtTs int64  `json:"created_at" gorm:"column:created_at;bigint;not null;default:0"`
}

type AgentQuotaPolicy struct {
	Id                    int   `json:"id"`
	AgentUserId           int   `json:"agent_user_id" gorm:"not null;uniqueIndex:uk_aqp_agent_user_id"`
	AllowRechargeUser     bool  `json:"allow_recharge_user" gorm:"not null;default:true"`
	AllowReclaimQuota     bool  `json:"allow_reclaim_quota" gorm:"not null;default:true"`
	MaxSingleAdjustAmount int   `json:"max_single_adjust_amount" gorm:"not null;default:0"`
	Status                int   `json:"status" gorm:"not null;default:1"`
	UpdatedAtTs           int64 `json:"updated_at" gorm:"column:updated_at;bigint;not null;default:0"`
}
