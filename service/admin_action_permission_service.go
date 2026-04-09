package service

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

const (
	ResourcePermissionManagement = "permission_management"
	ResourceAgentManagement      = "agent_management"
	ResourceUserManagement       = "user_management"
	ResourceQuotaManagement      = "quota_management"
	ResourceAuditManagement      = "audit_management"
)

const (
	ActionRead         = "read"
	ActionCreate       = "create"
	ActionUpdateStatus = "update_status"
	ActionBindProfile  = "bind_profile"
	ActionReadSummary  = "read_summary"
	ActionAdjust       = "adjust"
	ActionAdjustBatch  = "adjust_batch"
	ActionLedgerRead   = "ledger_read"
)

var adminActionCatalog = map[string][]string{
	ResourcePermissionManagement: {ActionRead, ActionBindProfile},
	ResourceAgentManagement:      {ActionRead, ActionCreate, ActionUpdateStatus},
	ResourceUserManagement:       {ActionRead, ActionUpdateStatus},
	ResourceQuotaManagement:      {ActionReadSummary, ActionAdjust, ActionAdjustBatch, ActionLedgerRead},
	ResourceAuditManagement:      {ActionRead},
}

func RequirePermissionAction(operatorUserId int, operatorRole int, resourceKey string, actionKey string) error {
	operator, err := ResolveOperatorUser(operatorUserId, operatorRole)
	if err != nil {
		return err
	}

	if operator.Role == common.RoleRootUser || operator.GetUserType() == model.UserTypeRoot {
		return nil
	}
	if err := MustBeAdminOrAgentUserType(operator.GetUserType()); err != nil {
		return errors.New("permission denied")
	}

	profile, actionMap, err := getActivePermissionActionMap(operator.Id)
	if err != nil {
		return err
	}
	if profile == nil {
		return errors.New("permission profile not assigned")
	}
	if !actionMap[permissionActionKey(resourceKey, actionKey)] {
		return errors.New("permission denied")
	}
	return nil
}

func BuildUserPermissions(userId int, userRole int) map[string]any {
	permissions := buildLegacySidebarPermissions(userRole)
	actions := map[string]bool{}

	operator, err := ResolveOperatorUser(userId, userRole)
	if err != nil {
		permissions["actions"] = actions
		return permissions
	}

	if operator.Role == common.RoleRootUser || operator.GetUserType() == model.UserTypeRoot {
		for resource, actionKeys := range adminActionCatalog {
			for _, actionKey := range actionKeys {
				actions[permissionActionKey(resource, actionKey)] = true
			}
		}
		permissions["actions"] = actions
		permissions["profile_type"] = model.UserTypeRoot
		return permissions
	}

	profile, actionMap, err := getActivePermissionActionMap(operator.Id)
	if err != nil {
		permissions["actions"] = actions
		return permissions
	}
	for key, allowed := range actionMap {
		if allowed {
			actions[key] = true
		}
	}
	permissions["actions"] = actions
	if profile != nil {
		permissions["profile_id"] = profile.Id
		permissions["profile_name"] = profile.ProfileName
		permissions["profile_type"] = profile.ProfileType
	}
	return permissions
}

func getActivePermissionActionMap(userId int) (*model.PermissionProfile, map[string]bool, error) {
	var binding model.UserPermissionBinding
	err := model.DB.Where("user_id = ? AND status = ?", userId, model.CommonStatusEnabled).
		Order("id desc").
		First(&binding).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, map[string]bool{}, nil
	}
	if err != nil {
		return nil, nil, err
	}

	var profile model.PermissionProfile
	if err := model.DB.First(&profile, binding.ProfileId).Error; err != nil {
		return nil, nil, err
	}

	var items []model.PermissionProfileItem
	if err := model.DB.Where("profile_id = ? AND allowed = ?", profile.Id, true).Find(&items).Error; err != nil {
		return nil, nil, err
	}

	actionMap := make(map[string]bool, len(items))
	for _, item := range items {
		actionMap[permissionActionKey(item.ResourceKey, item.ActionKey)] = true
	}
	return &profile, actionMap, nil
}

func permissionActionKey(resourceKey string, actionKey string) string {
	return resourceKey + "." + actionKey
}

func buildLegacySidebarPermissions(userRole int) map[string]any {
	permissions := map[string]any{}
	if userRole == common.RoleRootUser {
		permissions["sidebar_settings"] = false
		permissions["sidebar_modules"] = map[string]any{}
		return permissions
	}
	if userRole == common.RoleAdminUser {
		permissions["sidebar_settings"] = true
		permissions["sidebar_modules"] = map[string]any{
			"admin": map[string]any{
				"setting": false,
			},
		}
		return permissions
	}

	permissions["sidebar_settings"] = true
	permissions["sidebar_modules"] = map[string]any{
		"admin": false,
	}
	return permissions
}
