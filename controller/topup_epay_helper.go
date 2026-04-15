package controller

import (
	"fmt"

	"github.com/QuantumNous/new-api/model"
)

func processEpayTopUpSuccess(tradeNo string) error {
	if tradeNo == "" {
		return fmt.Errorf("trade no is empty")
	}
	return model.RechargeEpay(tradeNo)
}
