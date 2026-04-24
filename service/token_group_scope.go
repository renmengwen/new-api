package service

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func NormalizeAllowedTokenGroups(groups []string) []string {
	normalized := make([]string, 0, len(groups))
	seen := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		if _, ok := seen[group]; ok {
			continue
		}
		seen[group] = struct{}{}
		normalized = append(normalized, group)
	}
	return normalized
}

func currentTokenGroupDefinitions() map[string]string {
	groupRatios := ratio_setting.GetGroupRatioCopy()
	usableGroups := setting.GetUserUsableGroupsCopy()
	definitions := make(map[string]string, len(groupRatios)+len(usableGroups)+1)
	for groupName := range groupRatios {
		definitions[groupName] = setting.GetUsableGroupDescription(groupName)
	}
	for groupName, desc := range usableGroups {
		if groupName == "auto" {
			continue
		}
		definitions[groupName] = desc
	}
	if len(setting.GetAutoGroups()) > 0 {
		definitions["auto"] = setting.GetUsableGroupDescription("auto")
	}
	return definitions
}

func ResolveUserTokenSelectableGroups(userGroup string, userSetting dto.UserSetting) map[string]string {
	if !userSetting.AllowedTokenGroupsEnabled {
		userGroup = strings.TrimSpace(userGroup)
		if userGroup == "" {
			return map[string]string{}
		}
		return map[string]string{
			userGroup: setting.GetUsableGroupDescription(userGroup),
		}
	}
	definitions := currentTokenGroupDefinitions()
	resolved := make(map[string]string)
	trimmedUserGroup := strings.TrimSpace(userGroup)
	for _, group := range NormalizeAllowedTokenGroups(userSetting.AllowedTokenGroups) {
		if desc, ok := definitions[group]; ok {
			resolved[group] = desc
			continue
		}
		if group == trimmedUserGroup {
			resolved[group] = setting.GetUsableGroupDescription(group)
		}
	}
	return resolved
}

func ResolveAssignableTokenGroups(user *model.User) map[string]string {
	if user == nil {
		return map[string]string{}
	}
	userType := user.GetUserType()
	if user.Role == common.RoleRootUser || userType == model.UserTypeRoot || userType == model.UserTypeAdmin {
		return currentTokenGroupDefinitions()
	}
	return ResolveUserTokenSelectableGroups(user.Group, user.GetSetting())
}

func ValidateAllowedTokenGroupsAssignment(primaryGroup string, enabled bool, groups []string, assignable map[string]string) ([]string, error) {
	normalized := NormalizeAllowedTokenGroups(groups)
	if !enabled {
		return normalized, nil
	}

	primaryGroup = strings.TrimSpace(primaryGroup)
	if primaryGroup == "" {
		return nil, fmt.Errorf("primary group is required")
	}
	if len(normalized) == 0 {
		return nil, fmt.Errorf("allowed token groups cannot be empty when enabled")
	}

	containsPrimary := false
	for _, group := range normalized {
		if group == primaryGroup {
			containsPrimary = true
			continue
		}
		if _, ok := assignable[group]; !ok {
			return nil, fmt.Errorf("group %s is not assignable", group)
		}
	}
	if !containsPrimary {
		return nil, fmt.Errorf("primary group %s must be included in allowed token groups", primaryGroup)
	}
	return normalized, nil
}
