package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func BroadcastAnnouncementEmail(c *gin.Context) {
	var req service.AnnouncementEmailBroadcastRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的参数",
		})
		return
	}

	result, err := service.BroadcastAnnouncementEmail(req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
			"data":    result,
		})
		return
	}

	success := result.SentCount > 0 || result.FailedCount == 0
	message := ""
	if !success {
		message = "邮件发送失败"
	}
	c.JSON(http.StatusOK, gin.H{
		"success": success,
		"message": message,
		"data":    result,
	})
}
