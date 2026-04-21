package service

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

type AdminUserListItem struct {
	Id                  int    `json:"id"`
	Username            string `json:"username"`
	DisplayName         string `json:"display_name"`
	Status              int    `json:"status"`
	UserType            string `json:"user_type"`
	ParentAgentId       int    `json:"parent_agent_id"`
	ParentAgentUsername string `json:"parent_agent_username"`
	Group               string `json:"group"`
	Quota               int    `json:"quota"`
	UsedQuota           int    `json:"used_quota"`
	RequestCount        int    `json:"request_count"`
	InviterId           int    `json:"inviter_id"`
	InviterUsername     string `json:"inviter_username"`
	AffCount            int    `json:"aff_count"`
	AffHistory          int    `json:"aff_history_quota"`
	Remark              string `json:"remark"`
	LastActiveAt        int64  `json:"last_active_at"`
}

type CreateAdminUserRequest struct {
	Username                  string   `json:"username"`
	Password                  string   `json:"password"`
	DisplayName               string   `json:"display_name"`
	Email                     string   `json:"email"`
	Group                     string   `json:"group"`
	AllowedTokenGroupsEnabled bool     `json:"allowed_token_groups_enabled"`
	AllowedTokenGroups        []string `json:"allowed_token_groups"`
	Remark                    string   `json:"remark"`
}

type UpdateAdminUserRequest struct {
	Username                  string   `json:"username"`
	Password                  string   `json:"password"`
	DisplayName               string   `json:"display_name"`
	Email                     string   `json:"email"`
	Group                     string   `json:"group"`
	AllowedTokenGroupsEnabled bool     `json:"allowed_token_groups_enabled"`
	AllowedTokenGroups        []string `json:"allowed_token_groups"`
	Remark                    string   `json:"remark"`
	Quota                     int      `json:"quota"`
}

func adminUserGroupSelectExpr() string {
	if common.UsingPostgreSQL {
		return `users."group" AS "group"`
	}
	return "users.`group` AS `group`"
}

func ListAdminUsers(pageInfo *common.PageInfo, keyword string, operatorUserId int, operatorRole int) ([]AdminUserListItem, int64, error) {
	operator, err := ResolveOperatorUser(operatorUserId, operatorRole)
	if err != nil {
		return nil, 0, err
	}

	var items []AdminUserListItem
	var total int64

	query := model.DB.Model(&model.User{}).
		Joins("LEFT JOIN users AS inviter_users ON inviter_users.id = users.inviter_id").
		Joins("LEFT JOIN users AS parent_agents ON parent_agents.id = users.parent_agent_id").
		Select(
			"users.id, users.username, users.display_name, users.status, users.user_type, users.parent_agent_id, " +
				adminUserGroupSelectExpr() +
				", parent_agents.username AS parent_agent_username" +
				", users.quota, users.used_quota, users.request_count, users.inviter_id, inviter_users.username AS inviter_username, users.aff_count, users.aff_history, users.remark, users.last_active_at",
		)
	query = ApplyManagedEndUserScope(query, operator)
	if keyword != "" {
		like := "%" + strings.TrimSpace(keyword) + "%"
		query = query.Where("users.username LIKE ? OR users.display_name LIKE ? OR users.email LIKE ?", like, like, like)
	}

	countQuery := model.DB.Model(&model.User{})
	countQuery = ApplyManagedEndUserScope(countQuery, operator)
	if keyword != "" {
		like := "%" + strings.TrimSpace(keyword) + "%"
		countQuery = countQuery.Where("users.username LIKE ? OR users.display_name LIKE ? OR users.email LIKE ?", like, like, like)
	}

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("users.id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Scan(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func CreateAdminUserWithOperator(req CreateAdminUserRequest, operatorUserId int, operatorRole int, ip string) (*model.User, error) {
	operator, err := ResolveOperatorUser(operatorUserId, operatorRole)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username:      strings.TrimSpace(req.Username),
		Password:      req.Password,
		DisplayName:   firstNonEmpty(strings.TrimSpace(req.DisplayName), strings.TrimSpace(req.Username)),
		Role:          common.RoleCommonUser,
		Status:        common.UserStatusEnabled,
		UserType:      model.UserTypeEndUser,
		ParentAgentId: 0,
		Email:         strings.TrimSpace(req.Email),
		Group:         firstNonEmpty(strings.TrimSpace(req.Group), "default"),
		Remark:        strings.TrimSpace(req.Remark),
	}

	if operator.GetUserType() == model.UserTypeAgent {
		user.ParentAgentId = operator.Id
	}
	assignableGroups := ResolveAssignableTokenGroups(operator)
	normalizedAllowedTokenGroups, err := ValidateAllowedTokenGroupsAssignment(
		user.Group,
		req.AllowedTokenGroupsEnabled,
		req.AllowedTokenGroups,
		assignableGroups,
	)
	if err != nil {
		return nil, err
	}

	if err := common.Validate.Struct(user); err != nil {
		return nil, err
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

	if err := user.InsertWithTx(tx, 0, "admin_user_create"); err != nil {
		tx.Rollback()
		return nil, err
	}

	now := common.GetTimestamp()
	if operator.GetUserType() == model.UserTypeAgent {
		if err := tx.Model(&model.User{}).Where("id = ?", user.Id).Update("parent_agent_id", operator.Id).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		user.ParentAgentId = operator.Id

		relation := &model.AgentUserRelation{
			AgentUserId: operator.Id,
			EndUserId:   user.Id,
			BindSource:  "admin_create",
			BindAt:      now,
			Status:      model.CommonStatusEnabled,
			CreatedAtTs: now,
		}
		if err := tx.Create(relation).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}
	currentSetting := user.GetSetting()
	currentSetting.AllowedTokenGroupsEnabled = req.AllowedTokenGroupsEnabled
	currentSetting.AllowedTokenGroups = normalizedAllowedTokenGroups
	user.SetSetting(currentSetting)
	if err := tx.Model(&model.User{}).Where("id = ?", user.Id).Update("setting", user.Setting).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	afterJSON, _ := common.Marshal(map[string]any{
		"username":                     user.Username,
		"display_name":                 user.DisplayName,
		"user_type":                    user.GetUserType(),
		"parent_agent_id":              user.ParentAgentId,
		"group":                        user.Group,
		"allowed_token_groups_enabled": req.AllowedTokenGroupsEnabled,
		"allowed_token_groups":         normalizedAllowedTokenGroups,
	})
	if err := CreateAdminAuditLogTx(tx, AuditLogInput{
		OperatorUserId:   operator.Id,
		OperatorUserType: operator.GetUserType(),
		ActionModule:     ResourceUserManagement,
		ActionType:       ActionCreate,
		ActionDesc:       "create managed user",
		TargetType:       "user",
		TargetId:         user.Id,
		AfterJSON:        string(afterJSON),
		IP:               ip,
	}); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	user.FinalizeOAuthUserCreation(0)
	return user, nil
}

func GetAdminUserDetail(targetUserId int, operatorUserId int, operatorRole int) (map[string]any, error) {
	operator, err := ResolveOperatorUser(operatorUserId, operatorRole)
	if err != nil {
		return nil, err
	}

	var user model.User
	query := model.DB.Model(&model.User{}).Where("users.id = ?", targetUserId)
	query = ApplyManagedEndUserScope(query, operator)
	if err := query.First(&user).Error; err != nil {
		return nil, err
	}

	quotaSummary, err := GetUserQuotaSummary(user.Id)
	if err != nil {
		return nil, err
	}
	userSetting := user.GetSetting()

	return map[string]any{
		"id":                           user.Id,
		"username":                     user.Username,
		"display_name":                 user.DisplayName,
		"status":                       user.Status,
		"user_type":                    user.GetUserType(),
		"parent_agent_id":              user.ParentAgentId,
		"group":                        user.Group,
		"phone":                        user.Phone,
		"email":                        user.Email,
		"quota":                        user.Quota,
		"used_quota":                   user.UsedQuota,
		"request_count":                user.RequestCount,
		"inviter_id":                   user.InviterId,
		"aff_count":                    user.AffCount,
		"aff_history_quota":            user.AffHistoryQuota,
		"remark":                       user.Remark,
		"last_active_at":               user.LastActiveAt,
		"allowed_token_groups_enabled": userSetting.AllowedTokenGroupsEnabled,
		"allowed_token_groups":         userSetting.AllowedTokenGroups,
		"quota_summary":                quotaSummary,
	}, nil
}

func UpdateAdminUserWithOperator(targetUserId int, req UpdateAdminUserRequest, operatorUserId int, operatorRole int, ip string) error {
	operator, err := ResolveOperatorUser(operatorUserId, operatorRole)
	if err != nil {
		return err
	}

	user, err := GetManagedEndUser(targetUserId, operator.Id, operator.Role)
	if err != nil {
		return err
	}

	nextUsername := firstNonEmpty(strings.TrimSpace(req.Username), user.Username)
	nextDisplayName := firstNonEmpty(strings.TrimSpace(req.DisplayName), user.DisplayName, nextUsername)
	nextGroup := firstNonEmpty(strings.TrimSpace(req.Group), user.Group, "default")
	nextRemark := strings.TrimSpace(req.Remark)
	nextEmail := strings.TrimSpace(req.Email)
	originalQuota := user.Quota
	targetQuota := req.Quota
	quotaChanged := targetQuota != originalQuota
	assignableGroups := ResolveAssignableTokenGroups(operator)
	normalizedAllowedTokenGroups, err := ValidateAllowedTokenGroupsAssignment(
		nextGroup,
		req.AllowedTokenGroupsEnabled,
		req.AllowedTokenGroups,
		assignableGroups,
	)
	if err != nil {
		return err
	}

	beforeJSON, _ := common.Marshal(map[string]any{
		"username":                     user.Username,
		"display_name":                 user.DisplayName,
		"group":                        user.Group,
		"quota":                        user.Quota,
		"remark":                       user.Remark,
		"email":                        user.Email,
		"allowed_token_groups_enabled": user.GetSetting().AllowedTokenGroupsEnabled,
		"allowed_token_groups":         user.GetSetting().AllowedTokenGroups,
	})
	afterJSON, _ := common.Marshal(map[string]any{
		"username":                     nextUsername,
		"display_name":                 nextDisplayName,
		"group":                        nextGroup,
		"quota":                        req.Quota,
		"remark":                       nextRemark,
		"email":                        nextEmail,
		"allowed_token_groups_enabled": req.AllowedTokenGroupsEnabled,
		"allowed_token_groups":         normalizedAllowedTokenGroups,
	})

	user.Username = nextUsername
	user.DisplayName = nextDisplayName
	user.Group = nextGroup
	user.Remark = nextRemark
	user.Email = nextEmail
	user.Quota = targetQuota

	updatePassword := strings.TrimSpace(req.Password) != ""
	validationUser := *user
	if updatePassword {
		validationUser.Password = req.Password
	} else {
		validationUser.Password = "$I_LOVE_U"
	}

	if err := common.Validate.Struct(&validationUser); err != nil {
		return err
	}
	user.Quota = originalQuota
	if updatePassword {
		user.Password = req.Password
	}

	tx := model.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if updatePassword {
		hashedPassword, err := common.Password2Hash(user.Password)
		if err != nil {
			tx.Rollback()
			return err
		}
		user.Password = hashedPassword
	}

	updates := map[string]any{
		"username":     user.Username,
		"display_name": user.DisplayName,
		"group":        user.Group,
		"remark":       user.Remark,
		"email":        user.Email,
	}
	currentSetting := user.GetSetting()
	currentSetting.AllowedTokenGroupsEnabled = req.AllowedTokenGroupsEnabled
	currentSetting.AllowedTokenGroups = normalizedAllowedTokenGroups
	user.SetSetting(currentSetting)
	updates["setting"] = user.Setting
	if updatePassword {
		updates["password"] = user.Password
	}

	if err := tx.Model(&model.User{}).Where("id = ?", user.Id).Updates(updates).Error; err != nil {
		tx.Rollback()
		return err
	}

	now := common.GetTimestamp()
	var quotaResult *quotaApplyResult
	if quotaChanged {
		quotaResult, err = applyQuotaAdjustmentTx(tx, operator, user, AdjustUserQuotaRequest{
			OperatorUserId:   operator.Id,
			OperatorRole:     operator.Role,
			OperatorUserType: operator.GetUserType(),
			TargetUserId:     user.Id,
			Delta:            targetQuota - originalQuota,
			Reason:           "managed_user_update",
			Remark:           nextRemark,
			IP:               ip,
		}, "admin_user_update", user.Id, now)
		if err != nil {
			tx.Rollback()
			return err
		}
		user.Quota = targetQuota
	}

	if err := CreateAdminAuditLogTx(tx, AuditLogInput{
		OperatorUserId:   operator.Id,
		OperatorUserType: operator.GetUserType(),
		ActionModule:     ResourceUserManagement,
		ActionType:       ActionUpdate,
		ActionDesc:       "update managed user",
		TargetType:       "user",
		TargetId:         user.Id,
		BeforeJSON:       string(beforeJSON),
		AfterJSON:        string(afterJSON),
		IP:               ip,
	}); err != nil {
		tx.Rollback()
		return err
	}

	if quotaChanged && quotaResult != nil {
		quotaBeforeJSON, _ := common.Marshal(quotaResult.BeforeAudit)
		quotaAfterJSON, _ := common.Marshal(quotaResult.AfterAudit)
		if err := CreateAdminAuditLogTx(tx, AuditLogInput{
			OperatorUserId:   operator.Id,
			OperatorUserType: operator.GetUserType(),
			ActionModule:     "quota",
			ActionType:       "adjust",
			ActionDesc:       "update managed user quota",
			TargetType:       "user",
			TargetId:         user.Id,
			BeforeJSON:       string(quotaBeforeJSON),
			AfterJSON:        string(quotaAfterJSON),
			IP:               ip,
		}); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func UpdateAdminUserStatus(targetUserId int, status int, operatorUserId int, operatorRole int, ip string) error {
	operator, err := ResolveOperatorUser(operatorUserId, operatorRole)
	if err != nil {
		return err
	}

	var user model.User
	query := model.DB.Model(&model.User{}).Where("users.id = ?", targetUserId)
	query = ApplyManagedEndUserScope(query, operator)
	if err := query.First(&user).Error; err != nil {
		return err
	}

	if user.Status == status {
		return nil
	}

	beforeJSON, _ := common.Marshal(map[string]any{"status": user.Status})
	afterJSON, _ := common.Marshal(map[string]any{"status": status})
	actionType := "disable"
	if status == common.UserStatusEnabled {
		actionType = "enable"
	}

	tx := model.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Model(&model.User{}).Where("id = ?", user.Id).Update("status", status).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := CreateAdminAuditLogTx(tx, AuditLogInput{
		OperatorUserId:   operator.Id,
		OperatorUserType: operator.GetUserType(),
		ActionModule:     "user_management",
		ActionType:       actionType,
		ActionDesc:       "update managed user status",
		TargetType:       "user",
		TargetId:         user.Id,
		BeforeJSON:       string(beforeJSON),
		AfterJSON:        string(afterJSON),
		IP:               ip,
	}); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func DeleteAdminUserWithOperator(targetUserId int, operatorUserId int, operatorRole int, ip string) error {
	operator, err := ResolveOperatorUser(operatorUserId, operatorRole)
	if err != nil {
		return err
	}

	user, err := GetManagedEndUser(targetUserId, operator.Id, operator.Role)
	if err != nil {
		return err
	}

	beforeJSON, _ := common.Marshal(map[string]any{
		"username":     user.Username,
		"display_name": user.DisplayName,
		"status":       user.Status,
	})

	if err := user.Delete(); err != nil {
		return err
	}

	if err := CreateAdminAuditLog(AuditLogInput{
		OperatorUserId:   operator.Id,
		OperatorUserType: operator.GetUserType(),
		ActionModule:     ResourceUserManagement,
		ActionType:       ActionDelete,
		ActionDesc:       "delete managed user",
		TargetType:       "user",
		TargetId:         user.Id,
		BeforeJSON:       string(beforeJSON),
		IP:               ip,
	}); err != nil {
		return err
	}

	return nil
}

func IsManagedEndUserNotFound(err error) bool {
	return err != nil && err == gorm.ErrRecordNotFound
}
