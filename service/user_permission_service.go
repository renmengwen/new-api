package service

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

type UserPermissionTargetView struct {
	Id          int    `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        int    `json:"role"`
	UserType    string `json:"user_type"`
	Status      int    `json:"status"`
	ProfileId   int    `json:"profile_id"`
	ProfileName string `json:"profile_name"`
	ProfileType string `json:"profile_type"`
}

type UserPermissionOverrideInput struct {
	ResourceKey string `json:"resource_key"`
	ActionKey   string `json:"action_key"`
	Effect      string `json:"effect"`
}

type UserMenuOverrideInput struct {
	SectionKey string `json:"section_key"`
	ModuleKey  string `json:"module_key"`
	Effect     string `json:"effect"`
}

type UserDataScopeOverrideInput struct {
	ResourceKey string `json:"resource_key"`
	ScopeType   string `json:"scope_type"`
	ScopeValue  []int  `json:"scope_value"`
}

type UserDataScopeOverrideView struct {
	ResourceKey string `json:"resource_key"`
	ScopeType   string `json:"scope_type"`
	ScopeValue  []int  `json:"scope_value"`
}

type UserPermissionOverrideUpdateRequest struct {
	ActionOverrides    []UserPermissionOverrideInput `json:"action_overrides"`
	MenuOverrides      []UserMenuOverrideInput       `json:"menu_overrides"`
	DataScopeOverrides []UserDataScopeOverrideInput  `json:"data_scope_overrides"`
}

type UserPermissionDetail struct {
	User                *model.User                    `json:"user"`
	Binding             *model.UserPermissionBinding   `json:"binding,omitempty"`
	Profile             *model.PermissionProfile       `json:"profile,omitempty"`
	ActionOverrides     []model.UserPermissionOverride `json:"action_overrides"`
	MenuOverrides       []model.UserMenuOverride       `json:"menu_overrides"`
	DataScopeOverrides  []UserDataScopeOverrideView    `json:"data_scope_overrides"`
	EffectiveActions    map[string]bool                `json:"effective_actions"`
	EffectiveSidebar    map[string]any                 `json:"effective_sidebar"`
	EffectiveDataScopes map[string]string              `json:"effective_data_scopes"`
}

func ListUserPermissionTargets(pageInfo *common.PageInfo, keyword string, userType string, operatorUserId int, operatorRole int) ([]UserPermissionTargetView, int64, error) {
	var items []UserPermissionTargetView
	var total int64
	operator, err := ResolveOperatorUser(operatorUserId, operatorRole)
	if err != nil {
		return nil, 0, err
	}

	query := model.DB.Model(&model.User{}).
		Select(
			"users.id, users.username, users.display_name, users.role, users.user_type, users.status, " +
				"permission_profiles.id as profile_id, permission_profiles.profile_name, permission_profiles.profile_type",
		).
		Joins("LEFT JOIN user_permission_bindings ON user_permission_bindings.user_id = users.id AND user_permission_bindings.status = ?", model.CommonStatusEnabled).
		Joins("LEFT JOIN permission_profiles ON permission_profiles.id = user_permission_bindings.profile_id")

	if keyword != "" {
		like := "%" + strings.TrimSpace(keyword) + "%"
		query = query.Where("users.username LIKE ? OR users.display_name LIKE ? OR users.email LIKE ?", like, like, like)
	}
	countQuery := model.DB.Model(&model.User{})
	if keyword != "" {
		like := "%" + strings.TrimSpace(keyword) + "%"
		countQuery = countQuery.Where("username LIKE ? OR display_name LIKE ? OR email LIKE ?", like, like, like)
	}

	if operator.Role == common.RoleRootUser || operator.GetUserType() == model.UserTypeRoot || operator.GetUserType() == model.UserTypeAdmin {
		if userType != "" {
			query = query.Where("users.user_type = ? OR (COALESCE(users.user_type, '') = '' AND ? = ? AND users.role < ?)", userType, userType, model.UserTypeEndUser, common.RoleAdminUser)
			countQuery = countQuery.Where("user_type = ? OR (COALESCE(user_type, '') = '' AND ? = ? AND role < ?)", userType, userType, model.UserTypeEndUser, common.RoleAdminUser)
		}
	} else {
		query = ApplyManagedEndUserScope(query, operator, ResourcePermissionManagement)
		countQuery = ApplyManagedEndUserScope(countQuery, operator, ResourcePermissionManagement)
		if userType != "" && userType != model.UserTypeEndUser {
			query = query.Where("1 = 0")
			countQuery = countQuery.Where("1 = 0")
		}
	}

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("users.id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Scan(&items).Error; err != nil {
		return nil, 0, err
	}
	for idx := range items {
		items[idx].UserType = normalizePermissionTargetUserType(items[idx].UserType, items[idx].Role)
	}
	return items, total, nil
}

func normalizePermissionTargetUserType(userType string, role int) string {
	switch role {
	case common.RoleRootUser:
		return model.UserTypeRoot
	case common.RoleAdminUser:
		if userType == model.UserTypeAgent {
			return model.UserTypeAgent
		}
		return model.UserTypeAdmin
	default:
		if userType != "" {
			return userType
		}
		return model.UserTypeEndUser
	}
}

func GetUserPermissionDetail(userId int, operatorUserId int, operatorRole int) (*UserPermissionDetail, error) {
	user, err := getPermissionManagementTargetUser(userId, operatorUserId, operatorRole)
	if err != nil {
		return nil, err
	}

	var binding model.UserPermissionBinding
	var bindingPtr *model.UserPermissionBinding
	var profilePtr *model.PermissionProfile
	if err := model.DB.Where("user_id = ? AND status = ?", userId, model.CommonStatusEnabled).Order("id desc").First(&binding).Error; err == nil {
		bindingCopy := binding
		bindingPtr = &bindingCopy
		var profile model.PermissionProfile
		if err := model.DB.First(&profile, binding.ProfileId).Error; err == nil {
			profileCopy := profile
			profilePtr = &profileCopy
		}
	}

	actionOverrides, err := loadUserPermissionOverrides(userId)
	if err != nil {
		return nil, err
	}
	menuOverrides, err := loadUserMenuOverrides(userId)
	if err != nil {
		return nil, err
	}
	dataScopeOverrides, err := loadUserDataScopeOverrides(userId)
	if err != nil {
		return nil, err
	}
	_, effectiveActions, effectiveSidebar, err := getEffectivePermissionState(userId)
	if err != nil {
		return nil, err
	}

	return &UserPermissionDetail{
		User:                user,
		Binding:             bindingPtr,
		Profile:             profilePtr,
		ActionOverrides:     actionOverrides,
		MenuOverrides:       menuOverrides,
		DataScopeOverrides:  buildUserDataScopeOverrideViews(dataScopeOverrides),
		EffectiveActions:    effectiveActions,
		EffectiveSidebar:    effectiveSidebar,
		EffectiveDataScopes: buildEffectiveDataScopes(user, dataScopeOverrides),
	}, nil
}

func UpdateUserPermissionOverrides(userId int, req UserPermissionOverrideUpdateRequest, operatorUserId int, operatorRole int) error {
	if _, err := getPermissionManagementTargetUser(userId, operatorUserId, operatorRole); err != nil {
		return err
	}
	now := common.GetTimestamp()
	tx := model.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Where("user_id = ?", userId).Delete(&model.UserPermissionOverride{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Where("user_id = ?", userId).Delete(&model.UserMenuOverride{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Where("user_id = ?", userId).Delete(&model.UserDataScopeOverride{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if len(req.ActionOverrides) > 0 {
		rows := make([]model.UserPermissionOverride, 0, len(req.ActionOverrides))
		for _, item := range req.ActionOverrides {
			rows = append(rows, model.UserPermissionOverride{
				UserId:      userId,
				ResourceKey: item.ResourceKey,
				ActionKey:   item.ActionKey,
				Effect:      item.Effect,
				CreatedAtTs: now,
				UpdatedAtTs: now,
			})
		}
		if err := tx.Create(&rows).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if len(req.MenuOverrides) > 0 {
		rows := make([]model.UserMenuOverride, 0, len(req.MenuOverrides))
		for _, item := range req.MenuOverrides {
			rows = append(rows, model.UserMenuOverride{
				UserId:      userId,
				SectionKey:  item.SectionKey,
				ModuleKey:   item.ModuleKey,
				Effect:      item.Effect,
				CreatedAtTs: now,
				UpdatedAtTs: now,
			})
		}
		if err := tx.Create(&rows).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if len(req.DataScopeOverrides) > 0 {
		rows := make([]model.UserDataScopeOverride, 0, len(req.DataScopeOverrides))
		for _, item := range req.DataScopeOverrides {
			if item.ResourceKey == "" || item.ScopeType == "" {
				continue
			}
			scopeValueJSON := ""
			if len(item.ScopeValue) > 0 {
				data, err := common.Marshal(item.ScopeValue)
				if err != nil {
					tx.Rollback()
					return err
				}
				scopeValueJSON = string(data)
			}
			rows = append(rows, model.UserDataScopeOverride{
				UserId:         userId,
				ResourceKey:    item.ResourceKey,
				ScopeType:      item.ScopeType,
				ScopeValueJSON: scopeValueJSON,
				CreatedAtTs:    now,
				UpdatedAtTs:    now,
			})
		}
		if len(rows) > 0 {
			if err := tx.Create(&rows).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit().Error
}

func buildUserDataScopeOverrideViews(rows []model.UserDataScopeOverride) []UserDataScopeOverrideView {
	items := make([]UserDataScopeOverrideView, 0, len(rows))
	for _, row := range rows {
		view := UserDataScopeOverrideView{
			ResourceKey: row.ResourceKey,
			ScopeType:   row.ScopeType,
			ScopeValue:  []int{},
		}
		if row.ScopeValueJSON != "" {
			_ = common.UnmarshalJsonStr(row.ScopeValueJSON, &view.ScopeValue)
		}
		items = append(items, view)
	}
	return items
}

func buildEffectiveDataScopes(user *model.User, overrides []model.UserDataScopeOverride) map[string]string {
	scopeMap := map[string]string{
		ResourceUserManagement:  defaultDataScopeForUserType(user.GetUserType()),
		ResourceQuotaManagement: defaultDataScopeForUserType(user.GetUserType()),
	}
	for _, item := range overrides {
		scopeMap[item.ResourceKey] = item.ScopeType
	}
	return scopeMap
}

func defaultDataScopeForUserType(userType string) string {
	switch userType {
	case model.UserTypeRoot, model.UserTypeAdmin:
		return model.ScopeTypeAll
	case model.UserTypeAgent:
		return model.ScopeTypeAgentOnly
	default:
		return model.ScopeTypeSelf
	}
}

func getPermissionManagementTargetUser(userId int, operatorUserId int, operatorRole int) (*model.User, error) {
	operator, err := ResolveOperatorUser(operatorUserId, operatorRole)
	if err != nil {
		return nil, err
	}
	if operator.Role == common.RoleRootUser || operator.GetUserType() == model.UserTypeRoot || operator.GetUserType() == model.UserTypeAdmin {
		return model.GetUserById(userId, false)
	}
	return GetManagedEndUserForResource(userId, operatorUserId, operatorRole, ResourcePermissionManagement)
}
