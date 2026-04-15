package model

type AdminAuditLog struct {
	Id               int    `json:"id"`
	OperatorUserId   int    `json:"operator_user_id" gorm:"not null;index:idx_aal_operator_user_id_created_at,priority:1"`
	OperatorUserType string `json:"operator_user_type" gorm:"type:varchar(32);not null;default:''"`
	ActionModule     string `json:"action_module" gorm:"type:varchar(64);not null;index:idx_aal_action_module_created_at,priority:1"`
	ActionType       string `json:"action_type" gorm:"type:varchar(64);not null"`
	ActionDesc       string `json:"action_desc" gorm:"type:text"`
	TargetType       string `json:"target_type" gorm:"type:varchar(32);not null;default:'';index:idx_aal_target_type_target_id,priority:1"`
	TargetId         int    `json:"target_id" gorm:"not null;default:0;index:idx_aal_target_type_target_id,priority:2"`
	BeforeJSON       string `json:"before_json" gorm:"type:text"`
	AfterJSON        string `json:"after_json" gorm:"type:text"`
	Ip               string `json:"ip" gorm:"type:varchar(64);default:''"`
	CreatedAtTs      int64  `json:"created_at" gorm:"column:created_at;bigint;not null;default:0;index:idx_aal_operator_user_id_created_at,priority:2;index:idx_aal_action_module_created_at,priority:2;index:idx_aal_created_at"`
}
