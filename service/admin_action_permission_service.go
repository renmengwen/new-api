package service

import (
	"errors"
	"strings"

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
	ActionUpdate       = "update"
	ActionUpdateStatus = "update_status"
	ActionBindProfile  = "bind_profile"
	ActionReadSummary  = "read_summary"
	ActionAdjust       = "adjust"
	ActionAdjustBatch  = "adjust_batch"
	ActionLedgerRead   = "ledger_read"
)

var adminActionCatalog = map[string][]string{
	ResourcePermissionManagement: {ActionRead, ActionBindProfile},
	ResourceAgentManagement:      {ActionRead, ActionCreate, ActionUpdate, ActionUpdateStatus},
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

	_, actionMap, _, err := getEffectivePermissionState(operator.Id)
	if err != nil {
		return err
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

	profile, actionMap, sidebarModules, err := getEffectivePermissionState(operator.Id)
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
	permissions["sidebar_modules"] = sidebarModules
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

func getEffectivePermissionState(userId int) (*model.PermissionProfile, map[string]bool, map[string]any, error) {
	user, err := model.GetUserById(userId, true)
	if err != nil {
		return nil, nil, nil, err
	}

	baseSidebar := buildLegacySidebarPermissions(user.Role)
	sidebarModules, _ := baseSidebar["sidebar_modules"].(map[string]any)
	if sidebarModules == nil {
		sidebarModules = map[string]any{}
	}

	profile, actionMap, err := getActivePermissionActionMap(userId)
	if err != nil {
		return nil, nil, nil, err
	}

	actionOverrides, err := loadUserPermissionOverrides(userId)
	if err != nil {
		return nil, nil, nil, err
	}
	menuOverrides, err := loadUserMenuOverrides(userId)
	if err != nil {
		return nil, nil, nil, err
	}

	return profile, mergeActionOverrides(actionMap, actionOverrides), mergeMenuOverrides(sidebarModules, menuOverrides), nil
}

func permissionActionKey(resourceKey string, actionKey string) string {
	return resourceKey + "." + actionKey
}

func loadUserPermissionOverrides(userId int) ([]model.UserPermissionOverride, error) {
	var overrides []model.UserPermissionOverride
	err := model.DB.Where("user_id = ?", userId).Find(&overrides).Error
	if isMissingPermissionOverrideTableError(err) {
		return []model.UserPermissionOverride{}, nil
	}
	return overrides, err
}

func loadUserMenuOverrides(userId int) ([]model.UserMenuOverride, error) {
	var overrides []model.UserMenuOverride
	err := model.DB.Where("user_id = ?", userId).Find(&overrides).Error
	if isMissingPermissionOverrideTableError(err) {
		return []model.UserMenuOverride{}, nil
	}
	return overrides, err
}

func loadUserDataScopeOverrides(userId int) ([]model.UserDataScopeOverride, error) {
	var overrides []model.UserDataScopeOverride
	err := model.DB.Where("user_id = ?", userId).Find(&overrides).Error
	if isMissingPermissionOverrideTableError(err) {
		return []model.UserDataScopeOverride{}, nil
	}
	return overrides, err
}

func isMissingPermissionOverrideTableError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "no such table") ||
		strings.Contains(errMsg, "doesn't exist") ||
		strings.Contains(errMsg, "does not exist") ||
		strings.Contains(errMsg, "undefined table")
}

func mergeActionOverrides(actionMap map[string]bool, overrides []model.UserPermissionOverride) map[string]bool {
	merged := make(map[string]bool, len(actionMap)+len(overrides))
	for key, allowed := range actionMap {
		merged[key] = allowed
	}

	for _, override := range overrides {
		key := permissionActionKey(override.ResourceKey, override.ActionKey)
		switch override.Effect {
		case model.PermissionEffectAllow:
			merged[key] = true
		case model.PermissionEffectDeny:
			merged[key] = false
		}
	}
	return merged
}

func mergeMenuOverrides(sidebarModules map[string]any, overrides []model.UserMenuOverride) map[string]any {
	merged := cloneSidebarModules(sidebarModules)
	for _, override := range overrides {
		section := ensureSidebarSection(merged, override.SectionKey)
		switch override.Effect {
		case model.MenuEffectShow:
			section["enabled"] = true
			section[override.ModuleKey] = true
		case model.MenuEffectHide:
			section[override.ModuleKey] = false
		}
	}
	return merged
}

func cloneSidebarModules(sidebarModules map[string]any) map[string]any {
	clone := make(map[string]any, len(sidebarModules))
	for key, value := range sidebarModules {
		if section, ok := value.(map[string]any); ok {
			sectionClone := make(map[string]any, len(section))
			for sectionKey, sectionValue := range section {
				sectionClone[sectionKey] = sectionValue
			}
			clone[key] = sectionClone
			continue
		}
		clone[key] = value
	}
	return clone
}

func ensureSidebarSection(sidebarModules map[string]any, sectionKey string) map[string]any {
	if section, ok := sidebarModules[sectionKey].(map[string]any); ok {
		return section
	}
	section := map[string]any{"enabled": true}
	sidebarModules[sectionKey] = section
	return section
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
