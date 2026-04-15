package model

import "gorm.io/gorm"

type PermissionProfile struct {
	Id          int            `json:"id"`
	ProfileName string         `json:"profile_name" gorm:"type:varchar(128);not null;uniqueIndex:uk_permission_profiles_name_type,priority:1"`
	ProfileType string         `json:"profile_type" gorm:"type:varchar(32);not null;uniqueIndex:uk_permission_profiles_name_type,priority:2;index:idx_permission_profiles_type_status,priority:1"`
	IsBuiltin   bool           `json:"is_builtin" gorm:"not null;default:false"`
	Status      int            `json:"status" gorm:"not null;default:1;index:idx_permission_profiles_type_status,priority:2"`
	Description string         `json:"description" gorm:"type:text"`
	CreatedAtTs int64          `json:"created_at" gorm:"column:created_at;bigint;not null;default:0"`
	UpdatedAtTs int64          `json:"updated_at" gorm:"column:updated_at;bigint;not null;default:0"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

type PermissionProfileItem struct {
	Id             int    `json:"id"`
	ProfileId      int    `json:"profile_id" gorm:"not null;uniqueIndex:uk_ppi_profile_resource_action,priority:1;index:idx_ppi_profile_id"`
	ResourceKey    string `json:"resource_key" gorm:"type:varchar(64);not null;uniqueIndex:uk_ppi_profile_resource_action,priority:2"`
	ActionKey      string `json:"action_key" gorm:"type:varchar(64);not null;uniqueIndex:uk_ppi_profile_resource_action,priority:3"`
	Allowed        bool   `json:"allowed" gorm:"not null;default:true"`
	ScopeType      string `json:"scope_type" gorm:"type:varchar(32);not null;default:'all'"`
	ExtraScopeJSON string `json:"extra_scope_json" gorm:"type:text"`
	CreatedAtTs    int64  `json:"created_at" gorm:"column:created_at;bigint;not null;default:0"`
}

type UserPermissionBinding struct {
	Id            int   `json:"id"`
	UserId        int   `json:"user_id" gorm:"not null;index:idx_upb_user_id_status,priority:1"`
	ProfileId     int   `json:"profile_id" gorm:"not null;index:idx_upb_profile_id"`
	EffectiveFrom int64 `json:"effective_from" gorm:"bigint;not null;default:0"`
	EffectiveTo   int64 `json:"effective_to" gorm:"bigint;not null;default:0"`
	Status        int   `json:"status" gorm:"not null;default:1;index:idx_upb_user_id_status,priority:2"`
	CreatedAtTs   int64 `json:"created_at" gorm:"column:created_at;bigint;not null;default:0"`
}

type UserDataScope struct {
	Id          int    `json:"id"`
	UserId      int    `json:"user_id" gorm:"not null;index:idx_uds_user_id"`
	ScopeType   string `json:"scope_type" gorm:"type:varchar(32);not null"`
	TargetType  string `json:"target_type" gorm:"type:varchar(32);not null;index:idx_uds_target_type_target_id,priority:1"`
	TargetId    int    `json:"target_id" gorm:"not null;index:idx_uds_target_type_target_id,priority:2"`
	CreatedAtTs int64  `json:"created_at" gorm:"column:created_at;bigint;not null;default:0"`
}
