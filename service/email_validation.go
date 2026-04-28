package service

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

func normalizeRequiredEmail(email string) (string, error) {
	normalized := strings.TrimSpace(email)
	if err := common.Validate.Var(normalized, "required,email,max=50"); err != nil {
		return "", errors.New("无效的邮箱地址")
	}
	return normalized, nil
}

func normalizeRequiredUniqueEmail(email string, excludedUserId int) (string, error) {
	normalized, err := normalizeRequiredEmail(email)
	if err != nil {
		return "", err
	}
	if model.IsEmailAlreadyTakenByOtherUser(normalized, excludedUserId) {
		return "", errors.New("邮箱已存在，请更换后重试")
	}
	return normalized, nil
}
