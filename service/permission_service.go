package service

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

type PermissionUserView struct {
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

func ListPermissionProfiles(pageInfo *common.PageInfo, profileType string) ([]model.PermissionProfile, int64, error) {
	var profiles []model.PermissionProfile
	var total int64

	query := model.DB.Model(&model.PermissionProfile{})
	if profileType != "" {
		query = query.Where("profile_type = ?", profileType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&profiles).Error; err != nil {
		return nil, 0, err
	}
	return profiles, total, nil
}

func ListPermissionUsers(pageInfo *common.PageInfo, keyword string, userType string) ([]PermissionUserView, int64, error) {
	var items []PermissionUserView
	var total int64

	query := model.DB.Model(&model.User{}).
		Select(
			"users.id, users.username, users.display_name, users.role, users.user_type, users.status, "+
				"permission_profiles.id as profile_id, permission_profiles.profile_name, permission_profiles.profile_type",
		).
		Joins("LEFT JOIN user_permission_bindings ON user_permission_bindings.user_id = users.id AND user_permission_bindings.status = ?", model.CommonStatusEnabled).
		Joins("LEFT JOIN permission_profiles ON permission_profiles.id = user_permission_bindings.profile_id")

	if keyword != "" {
		like := "%" + strings.TrimSpace(keyword) + "%"
		query = query.Where("users.username LIKE ? OR users.display_name LIKE ? OR users.email LIKE ?", like, like, like)
	}

	if userType != "" {
		query = query.Where("users.user_type = ?", userType)
	} else {
		query = query.Where("users.role >= ? OR users.user_type IN ?", common.RoleAdminUser, []string{model.UserTypeRoot, model.UserTypeAdmin, model.UserTypeAgent})
	}

	countQuery := model.DB.Model(&model.User{})
	if keyword != "" {
		like := "%" + strings.TrimSpace(keyword) + "%"
		countQuery = countQuery.Where("username LIKE ? OR display_name LIKE ? OR email LIKE ?", like, like, like)
	}
	if userType != "" {
		countQuery = countQuery.Where("user_type = ?", userType)
	} else {
		countQuery = countQuery.Where("role >= ? OR user_type IN ?", common.RoleAdminUser, []string{model.UserTypeRoot, model.UserTypeAdmin, model.UserTypeAgent})
	}

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("users.id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Scan(&items).Error; err != nil {
		return nil, 0, err
	}

	for idx := range items {
		if items[idx].UserType == "" {
			items[idx].UserType = deriveUserTypeFromRole(items[idx].Role)
		}
	}

	return items, total, nil
}

func UpdateUserPermissionBinding(userId int, profileId int, operatorUserId int, operatorUserType string, ip string) (*model.UserPermissionBinding, error) {
	user, err := model.GetUserById(userId, true)
	if err != nil {
		return nil, err
	}
	if user.Id == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	targetUserType := user.GetUserType()
	if err := MustBeAdminOrAgentUserType(targetUserType); err != nil {
		return nil, err
	}

	var currentBinding model.UserPermissionBinding
	hasCurrentBinding := false
	err = model.DB.Where("user_id = ? AND status = ?", userId, model.CommonStatusEnabled).
		Order("id desc").
		First(&currentBinding).Error
	if err == nil {
		hasCurrentBinding = true
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	beforeAuditPayload := map[string]any{"profile_id": 0}
	if hasCurrentBinding {
		beforeAuditPayload["profile_id"] = currentBinding.ProfileId
		var currentProfile model.PermissionProfile
		if err := model.DB.First(&currentProfile, currentBinding.ProfileId).Error; err == nil {
			beforeAuditPayload["profile_name"] = currentProfile.ProfileName
			beforeAuditPayload["profile_type"] = currentProfile.ProfileType
		}
	}

	var profile model.PermissionProfile
	if profileId != 0 {
		if err := model.DB.First(&profile, profileId).Error; err != nil {
			return nil, err
		}
		expectedProfileType := permissionProfileTypeForUserType(targetUserType)
		if expectedProfileType == "" {
			return nil, errors.New("unsupported user_type for permission management")
		}
		if profile.ProfileType != expectedProfileType {
			return nil, errors.New("profile type does not match target user_type")
		}
	}

	tx := model.DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Model(&model.UserPermissionBinding{}).
		Where("user_id = ? AND status = ?", userId, model.CommonStatusEnabled).
		Update("status", model.CommonStatusDisabled).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if profileId == 0 {
		beforeJSON, _ := common.Marshal(beforeAuditPayload)
		afterJSON, _ := common.Marshal(map[string]any{"profile_id": 0})
		if err := CreateAdminAuditLogTx(tx, AuditLogInput{
			OperatorUserId:   operatorUserId,
			OperatorUserType: operatorUserType,
			ActionModule:     "permission",
			ActionType:       "clear_profile",
			ActionDesc:       "clear user permission profile",
			TargetType:       "user",
			TargetId:         userId,
			BeforeJSON:       string(beforeJSON),
			AfterJSON:        string(afterJSON),
			IP:               ip,
		}); err != nil {
			tx.Rollback()
			return nil, err
		}
		if err := tx.Commit().Error; err != nil {
			return nil, err
		}
		return &model.UserPermissionBinding{UserId: userId, ProfileId: 0, Status: model.CommonStatusDisabled}, nil
	}

	binding := &model.UserPermissionBinding{
		UserId:        userId,
		ProfileId:     profileId,
		Status:        model.CommonStatusEnabled,
		EffectiveFrom: common.GetTimestamp(),
		CreatedAtTs:   common.GetTimestamp(),
	}
	if err := tx.Create(binding).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	beforeJSON, _ := common.Marshal(beforeAuditPayload)
	afterJSON, _ := common.Marshal(map[string]any{"profile_id": profileId, "profile_name": profile.ProfileName, "profile_type": profile.ProfileType})
	if err := CreateAdminAuditLogTx(tx, AuditLogInput{
		OperatorUserId:   operatorUserId,
		OperatorUserType: operatorUserType,
		ActionModule:     "permission",
		ActionType:       "bind_profile",
		ActionDesc:       "bind permission profile",
		TargetType:       "user",
		TargetId:         userId,
		BeforeJSON:       string(beforeJSON),
		AfterJSON:        string(afterJSON),
		IP:               ip,
	}); err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return binding, nil
}

func deriveUserTypeFromRole(role int) string {
	return UserTypeFromRole(role)
}

func MustBeAdminOrAgentUserType(userType string) error {
	switch userType {
	case model.UserTypeRoot, model.UserTypeAdmin, model.UserTypeAgent, model.UserTypeEndUser:
		return nil
	default:
		return errors.New("unsupported user_type for permission management")
	}
}

func permissionProfileTypeForUserType(userType string) string {
	switch userType {
	case model.UserTypeRoot, model.UserTypeAdmin:
		return model.UserTypeAdmin
	case model.UserTypeAgent:
		return model.UserTypeAgent
	case model.UserTypeEndUser:
		return model.UserTypeEndUser
	default:
		return ""
	}
}
