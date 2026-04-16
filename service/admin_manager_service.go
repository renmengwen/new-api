package service

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

type CreateAdminManagerRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	Group       string `json:"group"`
	Remark      string `json:"remark"`
}

type UpdateAdminManagerRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	Remark      string `json:"remark"`
}

type AdminManagerListItem struct {
	Id           int    `json:"id"`
	Username     string `json:"username"`
	DisplayName  string `json:"display_name"`
	Status       int    `json:"status"`
	UserType     string `json:"user_type"`
	LastActiveAt int64  `json:"last_active_at"`
	Remark       string `json:"remark"`
}

func ListAdminManagers(pageInfo *common.PageInfo, keyword string) ([]AdminManagerListItem, int64, error) {
	var items []AdminManagerListItem
	var total int64

	query := model.DB.Model(&model.User{}).
		Select("users.id, users.username, users.display_name, users.status, users.user_type, users.last_active_at, users.remark").
		Where("users.role = ?", common.RoleAdminUser)

	countQuery := model.DB.Model(&model.User{}).
		Where("users.role = ?", common.RoleAdminUser)

	if keyword != "" {
		like := "%" + strings.TrimSpace(keyword) + "%"
		query = query.Where("users.username LIKE ? OR users.display_name LIKE ? OR users.remark LIKE ?", like, like, like)
		countQuery = countQuery.Where("users.username LIKE ? OR users.display_name LIKE ? OR users.remark LIKE ?", like, like, like)
	}

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("users.id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Scan(&items).Error; err != nil {
		return nil, 0, err
	}

	for idx := range items {
		if items[idx].UserType == "" {
			items[idx].UserType = model.UserTypeAdmin
		}
	}
	return items, total, nil
}

func CreateAdminManagerWithOperator(req CreateAdminManagerRequest, operatorUserId int, operatorUserType string, ip string) (*model.User, error) {
	user := &model.User{
		Username:    strings.TrimSpace(req.Username),
		Password:    req.Password,
		DisplayName: firstNonEmpty(strings.TrimSpace(req.DisplayName), strings.TrimSpace(req.Username)),
		Role:        common.RoleAdminUser,
		Status:      common.UserStatusEnabled,
		UserType:    model.UserTypeAdmin,
		Group:       firstNonEmpty(strings.TrimSpace(req.Group), "default"),
		Remark:      strings.TrimSpace(req.Remark),
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

	if err := user.InsertWithTx(tx, 0, "admin_manager_create"); err != nil {
		tx.Rollback()
		return nil, err
	}

	afterJSON, _ := common.Marshal(map[string]any{
		"username":     user.Username,
		"display_name": user.DisplayName,
		"user_type":    user.GetUserType(),
		"remark":       user.Remark,
	})
	if err := CreateAdminAuditLogTx(tx, AuditLogInput{
		OperatorUserId:   operatorUserId,
		OperatorUserType: operatorUserType,
		ActionModule:     ResourceAdminManagement,
		ActionType:       ActionCreate,
		ActionDesc:       "create admin manager",
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

func GetAdminManagerDetail(userId int) (map[string]any, error) {
	user, err := model.GetUserById(userId, false)
	if err != nil {
		return nil, err
	}
	if user.Role != common.RoleAdminUser || user.GetUserType() != model.UserTypeAdmin {
		return nil, gorm.ErrRecordNotFound
	}

	return map[string]any{
		"id":             user.Id,
		"username":       user.Username,
		"display_name":   user.DisplayName,
		"status":         user.Status,
		"user_type":      user.GetUserType(),
		"last_active_at": user.LastActiveAt,
		"remark":         user.Remark,
	}, nil
}

func UpdateAdminManagerWithOperator(userId int, req UpdateAdminManagerRequest, operatorUserId int, operatorUserType string, ip string) error {
	var user model.User
	if err := model.DB.First(&user, userId).Error; err != nil {
		return err
	}
	if user.Role != common.RoleAdminUser || user.GetUserType() != model.UserTypeAdmin {
		return gorm.ErrRecordNotFound
	}

	nextUsername := firstNonEmpty(strings.TrimSpace(req.Username), user.Username)
	nextDisplayName := firstNonEmpty(strings.TrimSpace(req.DisplayName), user.DisplayName)
	nextRemark := strings.TrimSpace(req.Remark)

	beforeJSON, _ := common.Marshal(map[string]any{
		"username":     user.Username,
		"display_name": user.DisplayName,
		"remark":       user.Remark,
	})
	afterJSON, _ := common.Marshal(map[string]any{
		"username":     nextUsername,
		"display_name": nextDisplayName,
		"remark":       nextRemark,
	})

	user.Username = nextUsername
	user.DisplayName = nextDisplayName
	user.Remark = nextRemark
	updatePassword := strings.TrimSpace(req.Password) != ""
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
		"remark":       user.Remark,
	}
	if updatePassword {
		updates["password"] = user.Password
	}

	if err := tx.Model(&model.User{}).Where("id = ?", userId).Updates(updates).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := CreateAdminAuditLogTx(tx, AuditLogInput{
		OperatorUserId:   operatorUserId,
		OperatorUserType: operatorUserType,
		ActionModule:     ResourceAdminManagement,
		ActionType:       ActionUpdate,
		ActionDesc:       "update admin manager",
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

func UpdateAdminManagerStatusWithOperator(userId int, status int, operatorUserId int, operatorUserType string, ip string) error {
	var user model.User
	if err := model.DB.First(&user, userId).Error; err != nil {
		return err
	}
	if user.Role != common.RoleAdminUser || user.GetUserType() != model.UserTypeAdmin {
		return gorm.ErrRecordNotFound
	}

	beforeJSON, _ := common.Marshal(map[string]any{"status": user.Status})
	afterJSON, _ := common.Marshal(map[string]any{"status": status})
	actionType := ActionUpdateStatus
	if status == common.UserStatusEnabled {
		actionType = "enable"
	} else if status == common.UserStatusDisabled {
		actionType = "disable"
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

	if err := tx.Model(&model.User{}).Where("id = ?", userId).Update("status", status).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := CreateAdminAuditLogTx(tx, AuditLogInput{
		OperatorUserId:   operatorUserId,
		OperatorUserType: operatorUserType,
		ActionModule:     ResourceAdminManagement,
		ActionType:       actionType,
		ActionDesc:       "update admin manager status",
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
