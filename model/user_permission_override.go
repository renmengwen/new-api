package model

type UserPermissionOverride struct {
	Id          int    `json:"id"`
	UserId      int    `json:"user_id" gorm:"not null;index:idx_upo_user_id;uniqueIndex:uk_upo_user_resource_action,priority:1"`
	ResourceKey string `json:"resource_key" gorm:"type:varchar(64);not null;uniqueIndex:uk_upo_user_resource_action,priority:2"`
	ActionKey   string `json:"action_key" gorm:"type:varchar(64);not null;uniqueIndex:uk_upo_user_resource_action,priority:3"`
	Effect      string `json:"effect" gorm:"type:varchar(16);not null"`
	CreatedAtTs int64  `json:"created_at" gorm:"column:created_at;bigint;not null;default:0"`
	UpdatedAtTs int64  `json:"updated_at" gorm:"column:updated_at;bigint;not null;default:0"`
}

type UserMenuOverride struct {
	Id          int    `json:"id"`
	UserId      int    `json:"user_id" gorm:"not null;index:idx_umo_user_id;uniqueIndex:uk_umo_user_section_module,priority:1"`
	SectionKey  string `json:"section_key" gorm:"type:varchar(64);not null;uniqueIndex:uk_umo_user_section_module,priority:2"`
	ModuleKey   string `json:"module_key" gorm:"type:varchar(64);not null;uniqueIndex:uk_umo_user_section_module,priority:3"`
	Effect      string `json:"effect" gorm:"type:varchar(16);not null"`
	CreatedAtTs int64  `json:"created_at" gorm:"column:created_at;bigint;not null;default:0"`
	UpdatedAtTs int64  `json:"updated_at" gorm:"column:updated_at;bigint;not null;default:0"`
}

type UserDataScopeOverride struct {
	Id             int    `json:"id"`
	UserId         int    `json:"user_id" gorm:"not null;index:idx_udso_user_id;uniqueIndex:uk_udso_user_resource,priority:1"`
	ResourceKey    string `json:"resource_key" gorm:"type:varchar(64);not null;uniqueIndex:uk_udso_user_resource,priority:2"`
	ScopeType      string `json:"scope_type" gorm:"type:varchar(32);not null"`
	ScopeValueJSON string `json:"scope_value_json" gorm:"type:text"`
	CreatedAtTs    int64  `json:"created_at" gorm:"column:created_at;bigint;not null;default:0"`
	UpdatedAtTs    int64  `json:"updated_at" gorm:"column:updated_at;bigint;not null;default:0"`
}
