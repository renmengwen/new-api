package service

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

type CreateAgentRequest struct {
	Username                  string   `json:"username"`
	Password                  string   `json:"password"`
	AgentName                 string   `json:"agent_name"`
	CompanyName               string   `json:"company_name"`
	ContactPhone              string   `json:"contact_phone"`
	Group                     string   `json:"group"`
	AllowedTokenGroupsEnabled bool     `json:"allowed_token_groups_enabled"`
	AllowedTokenGroups        []string `json:"allowed_token_groups"`
	Remark                    string   `json:"remark"`
}

type UpdateAgentRequest struct {
	DisplayName               string   `json:"display_name"`
	AgentName                 string   `json:"agent_name"`
	CompanyName               string   `json:"company_name"`
	ContactPhone              string   `json:"contact_phone"`
	Group                     string   `json:"group"`
	AllowedTokenGroupsEnabled bool     `json:"allowed_token_groups_enabled"`
	AllowedTokenGroups        []string `json:"allowed_token_groups"`
	Remark                    string   `json:"remark"`
}

type AgentListItem struct {
	Id           int    `json:"id"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	Status       int    `json:"status"`
	UserType     string `json:"user_type"`
	AgentName    string `json:"agent_name"`
	CompanyName  string `json:"company_name"`
	ContactPhone string `json:"contact_phone"`
}

func CreateAgent(req CreateAgentRequest) (*model.User, error) {
	return CreateAgentWithOperator(req, 0, "", "")
}

func CreateAgentWithOperator(req CreateAgentRequest, operatorUserId int, operatorUserType string, ip string) (*model.User, error) {
	user := &model.User{
		Username:    strings.TrimSpace(req.Username),
		Password:    req.Password,
		DisplayName: firstNonEmpty(strings.TrimSpace(req.AgentName), strings.TrimSpace(req.Username)),
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAgent,
		Group:       firstNonEmpty(strings.TrimSpace(req.Group), "default"),
		Phone:       strings.TrimSpace(req.ContactPhone),
	}

	inviterId := 0
	if operatorUserId > 0 && (operatorUserType == model.UserTypeRoot || operatorUserType == model.UserTypeAdmin) {
		inviterId = operatorUserId
		user.InviterId = inviterId
	}
	assignableGroups := currentTokenGroupDefinitions()
	if operatorUserId > 0 {
		if operator, err := model.GetUserById(operatorUserId, false); err == nil {
			assignableGroups = ResolveAssignableTokenGroups(operator)
		}
	}
	normalizedAllowedTokenGroups, err := ValidateAllowedTokenGroupsAssignment(
		user.Group,
		req.AllowedTokenGroupsEnabled,
		req.AllowedTokenGroups,
		assignableGroups,
	)
	if err != nil {
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

	if err := user.InsertWithTx(tx, inviterId, "agent_create"); err != nil {
		tx.Rollback()
		return nil, err
	}

	now := common.GetTimestamp()
	profile := &model.AgentProfile{
		UserId:       user.Id,
		AgentName:    firstNonEmpty(strings.TrimSpace(req.AgentName), user.DisplayName),
		CompanyName:  strings.TrimSpace(req.CompanyName),
		ContactPhone: strings.TrimSpace(req.ContactPhone),
		Remark:       strings.TrimSpace(req.Remark),
		Status:       model.CommonStatusEnabled,
		CreatedAtTs:  now,
		UpdatedAtTs:  now,
	}
	if err := tx.Create(profile).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	policy := &model.AgentQuotaPolicy{
		AgentUserId:           user.Id,
		AllowRechargeUser:     true,
		AllowReclaimQuota:     true,
		MaxSingleAdjustAmount: 0,
		Status:                model.CommonStatusEnabled,
		UpdatedAtTs:           now,
	}
	if err := tx.Create(policy).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	setting := user.GetSetting()
	setting.SidebarModules = buildAgentSidebarModules()
	setting.AllowedTokenGroupsEnabled = req.AllowedTokenGroupsEnabled
	setting.AllowedTokenGroups = normalizedAllowedTokenGroups
	user.SetSetting(setting)
	if err := tx.Model(&model.User{}).Where("id = ?", user.Id).Update("setting", user.Setting).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	afterJSON, _ := common.Marshal(map[string]any{
		"username":                     user.Username,
		"user_type":                    user.UserType,
		"agent_name":                   profile.AgentName,
		"group":                        user.Group,
		"allowed_token_groups_enabled": req.AllowedTokenGroupsEnabled,
		"allowed_token_groups":         normalizedAllowedTokenGroups,
	})
	if err := CreateAdminAuditLogTx(tx, AuditLogInput{
		OperatorUserId:   operatorUserId,
		OperatorUserType: operatorUserType,
		ActionModule:     "agent",
		ActionType:       "create",
		ActionDesc:       "create agent",
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
	return user, nil
}

func ListAgents(pageInfo *common.PageInfo, keyword string) ([]AgentListItem, int64, error) {
	var items []AgentListItem
	var total int64

	query := model.DB.Model(&model.User{}).
		Select("users.id, users.username, users.display_name, users.status, users.user_type, agent_profiles.agent_name, agent_profiles.company_name, agent_profiles.contact_phone").
		Joins("INNER JOIN agent_profiles ON agent_profiles.user_id = users.id").
		Where("users.user_type = ?", model.UserTypeAgent)

	countQuery := model.DB.Model(&model.User{}).
		Joins("INNER JOIN agent_profiles ON agent_profiles.user_id = users.id").
		Where("users.user_type = ?", model.UserTypeAgent)

	if keyword != "" {
		like := "%" + strings.TrimSpace(keyword) + "%"
		query = query.Where("users.username LIKE ? OR users.display_name LIKE ? OR agent_profiles.agent_name LIKE ?", like, like, like)
		countQuery = countQuery.Where("users.username LIKE ? OR users.display_name LIKE ? OR agent_profiles.agent_name LIKE ?", like, like, like)
	}

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("users.id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Scan(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func GetAgentDetail(userId int) (map[string]any, error) {
	user, err := model.GetUserById(userId, false)
	if err != nil {
		return nil, err
	}
	if user.GetUserType() != model.UserTypeAgent {
		return nil, errors.New("agent user not found")
	}

	var profile model.AgentProfile
	if err := model.DB.Where("user_id = ?", userId).First(&profile).Error; err != nil {
		return nil, err
	}

	account, err := model.GetQuotaAccountByOwner(model.QuotaOwnerTypeUser, userId)
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
		"agent_name":                   profile.AgentName,
		"company_name":                 profile.CompanyName,
		"contact_phone":                profile.ContactPhone,
		"group":                        user.Group,
		"allowed_token_groups_enabled": userSetting.AllowedTokenGroupsEnabled,
		"allowed_token_groups":         userSetting.AllowedTokenGroups,
		"remark":                       profile.Remark,
		"quota_summary": map[string]any{
			"balance":        account.Balance,
			"frozen_balance": account.FrozenBalance,
			"status":         account.Status,
		},
	}, nil
}

func UpdateAgentWithOperator(userId int, req UpdateAgentRequest, operatorUserId int, operatorUserType string, ip string) error {
	now := common.GetTimestamp()

	var user model.User
	if err := model.DB.First(&user, userId).Error; err != nil {
		return err
	}
	if user.GetUserType() != model.UserTypeAgent {
		return errors.New("agent user not found")
	}

	var profile model.AgentProfile
	if err := model.DB.Where("user_id = ?", userId).First(&profile).Error; err != nil {
		return err
	}

	nextDisplayName := firstNonEmpty(strings.TrimSpace(req.DisplayName), strings.TrimSpace(req.AgentName), user.DisplayName)
	nextAgentName := firstNonEmpty(strings.TrimSpace(req.AgentName), nextDisplayName, profile.AgentName)
	nextCompanyName := strings.TrimSpace(req.CompanyName)
	nextContactPhone := strings.TrimSpace(req.ContactPhone)
	nextGroup := firstNonEmpty(strings.TrimSpace(req.Group), user.Group, "default")
	nextRemark := strings.TrimSpace(req.Remark)
	assignableGroups := currentTokenGroupDefinitions()
	if operatorUserId > 0 {
		if operator, err := model.GetUserById(operatorUserId, false); err == nil {
			assignableGroups = ResolveAssignableTokenGroups(operator)
		}
	}
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
		"display_name":                 user.DisplayName,
		"agent_name":                   profile.AgentName,
		"company_name":                 profile.CompanyName,
		"contact_phone":                profile.ContactPhone,
		"group":                        user.Group,
		"allowed_token_groups_enabled": user.GetSetting().AllowedTokenGroupsEnabled,
		"allowed_token_groups":         user.GetSetting().AllowedTokenGroups,
		"remark":                       profile.Remark,
	})
	afterJSON, _ := common.Marshal(map[string]any{
		"display_name":                 nextDisplayName,
		"agent_name":                   nextAgentName,
		"company_name":                 nextCompanyName,
		"contact_phone":                nextContactPhone,
		"group":                        nextGroup,
		"allowed_token_groups_enabled": req.AllowedTokenGroupsEnabled,
		"allowed_token_groups":         normalizedAllowedTokenGroups,
		"remark":                       nextRemark,
	})

	tx := model.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Model(&model.User{}).Where("id = ?", userId).Updates(map[string]any{
		"display_name": nextDisplayName,
		"phone":        nextContactPhone,
		"group":        nextGroup,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}
	currentSetting := user.GetSetting()
	currentSetting.AllowedTokenGroupsEnabled = req.AllowedTokenGroupsEnabled
	currentSetting.AllowedTokenGroups = normalizedAllowedTokenGroups
	user.SetSetting(currentSetting)
	if err := tx.Model(&model.User{}).Where("id = ?", userId).Update("setting", user.Setting).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&model.AgentProfile{}).Where("user_id = ?", userId).Updates(map[string]any{
		"agent_name":    nextAgentName,
		"company_name":  nextCompanyName,
		"contact_phone": nextContactPhone,
		"remark":        nextRemark,
		"updated_at":    now,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := CreateAdminAuditLogTx(tx, AuditLogInput{
		OperatorUserId:   operatorUserId,
		OperatorUserType: operatorUserType,
		ActionModule:     "agent",
		ActionType:       "update",
		ActionDesc:       "update agent profile",
		TargetType:       "user",
		TargetId:         userId,
		BeforeJSON:       string(beforeJSON),
		AfterJSON:        string(afterJSON),
		IP:               ip,
	}); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func UpdateAgentStatus(userId int, status int) error {
	return UpdateAgentStatusWithOperator(userId, status, 0, "", "")
}

func UpdateAgentStatusWithOperator(userId int, status int, operatorUserId int, operatorUserType string, ip string) error {
	now := common.GetTimestamp()
	var user model.User
	if err := model.DB.First(&user, userId).Error; err != nil {
		return err
	}
	if user.GetUserType() != model.UserTypeAgent {
		return errors.New("agent user not found")
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

	if err := tx.Model(&model.User{}).Where("id = ?", userId).Updates(map[string]any{
		"status": status,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&model.AgentProfile{}).Where("user_id = ?", userId).Updates(map[string]any{
		"status":     status,
		"updated_at": now,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	beforeJSON, _ := common.Marshal(map[string]any{"status": user.Status})
	afterJSON, _ := common.Marshal(map[string]any{"status": status})
	actionType := "disable"
	if status == common.UserStatusEnabled {
		actionType = "enable"
	}
	if err := CreateAdminAuditLogTx(tx, AuditLogInput{
		OperatorUserId:   operatorUserId,
		OperatorUserType: operatorUserType,
		ActionModule:     "agent",
		ActionType:       actionType,
		ActionDesc:       "update agent status",
		TargetType:       "user",
		TargetId:         userId,
		BeforeJSON:       string(beforeJSON),
		AfterJSON:        string(afterJSON),
		IP:               ip,
	}); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func buildAgentSidebarModules() string {
	return `{"chat":{"enabled":true,"playground":true,"chat":true},"console":{"enabled":true,"detail":true,"token":true,"log":true,"midjourney":true,"task":true},"personal":{"enabled":true,"topup":true,"personal":true},"admin":{"enabled":true,"user":true}}`
}
