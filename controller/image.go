package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

func GetImage(c *gin.Context) {

}

func GetImageGenerationTask(c *gin.Context) {
	taskID := c.Param("task_id")
	task, exist, err := model.GetByTaskId(c.GetInt("id"), taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": err.Error(),
				"type":    "server_error",
				"code":    "get_task_failed",
			},
		})
		return
	}
	if !exist {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"message": "task_not_exist",
				"type":    "invalid_request_error",
				"code":    "task_not_exist",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"task_id":     task.TaskID,
		"platform":    "image",
		"action":      constant.TaskTypeImageGeneration,
		"status":      string(task.Status),
		"progress":    task.Progress,
		"result_url":  imageGenerationResultURL(c, task),
		"fail_reason": task.FailReason,
	})
}

func imageGenerationResultURL(c *gin.Context, task *model.Task) string {
	if task == nil || task.Status != model.TaskStatusSuccess || strings.TrimSpace(task.GetResultURL()) == "" {
		return ""
	}
	return imageGenerationContentURL(c, task.TaskID)
}

func imageGenerationContentURL(c *gin.Context, taskID string) string {
	path := "/v1/images/generations/" + url.PathEscape(taskID) + "/content"
	if c != nil && c.Request != nil {
		host := strings.TrimSpace(c.GetHeader("X-Forwarded-Host"))
		if host == "" {
			host = strings.TrimSpace(c.Request.Host)
		}
		if host != "" {
			scheme := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto"))
			if scheme == "" {
				scheme = strings.TrimSpace(c.GetHeader("X-Forwarded-Protocol"))
			}
			if scheme == "" {
				if c.Request.TLS != nil {
					scheme = "https"
				} else {
					scheme = "http"
				}
			}
			return scheme + "://" + host + path
		}
	}
	serverAddress := strings.TrimRight(strings.TrimSpace(system_setting.ServerAddress), "/")
	if serverAddress != "" {
		return serverAddress + path
	}
	return path
}

func GetImageGenerationContent(c *gin.Context) {
	taskID := c.Param("task_id")
	task, exist, err := model.GetByTaskId(c.GetInt("id"), taskID)
	if err != nil {
		imageGenerationProxyError(c, http.StatusInternalServerError, "server_error", "Failed to query task")
		return
	}
	if !exist || task == nil {
		imageGenerationProxyError(c, http.StatusNotFound, "invalid_request_error", "Task not found")
		return
	}
	if task.Status != model.TaskStatusSuccess {
		imageGenerationProxyError(c, http.StatusBadRequest, "invalid_request_error", fmt.Sprintf("Task is not completed yet, current status: %s", task.Status))
		return
	}

	imageURL := strings.TrimSpace(task.GetResultURL())
	if imageURL == "" {
		imageGenerationProxyError(c, http.StatusBadGateway, "server_error", "Failed to fetch image content")
		return
	}
	if strings.HasPrefix(imageURL, "data:") {
		if err := writeVideoDataURL(c, imageURL); err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to decode image data URL for task %s: %s", taskID, err.Error()))
			imageGenerationProxyError(c, http.StatusBadGateway, "server_error", "Failed to fetch image content")
		}
		return
	}

	fetchSetting := system_setting.GetFetchSetting()
	if err := common.ValidateURLWithFetchSetting(imageURL, fetchSetting.EnableSSRFProtection, fetchSetting.AllowPrivateIp, fetchSetting.DomainFilterMode, fetchSetting.IpFilterMode, fetchSetting.DomainList, fetchSetting.IpList, fetchSetting.AllowedPorts, fetchSetting.ApplyIPFilterForDomain); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Image URL blocked for task %s: %v", taskID, err))
		imageGenerationProxyError(c, http.StatusForbidden, "server_error", fmt.Sprintf("request blocked: %v", err))
		return
	}

	client := http.DefaultClient
	if channel, err := model.CacheGetChannel(task.ChannelId); err == nil && channel != nil {
		proxyClient, err := service.GetHttpClientWithProxy(channel.GetSetting().Proxy)
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to create proxy client for image task %s: %s", taskID, err.Error()))
			imageGenerationProxyError(c, http.StatusInternalServerError, "server_error", "Failed to create proxy client")
			return
		}
		if proxyClient != nil {
			client = proxyClient
		}
	}

	ctx, cancel := contextWithImageTimeout(c)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		imageGenerationProxyError(c, http.StatusInternalServerError, "server_error", "Failed to create proxy request")
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to fetch image from %s: %s", common.MaskSensitiveInfo(imageURL), err.Error()))
		imageGenerationProxyError(c, http.StatusBadGateway, "server_error", "Failed to fetch image content")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		imageGenerationProxyError(c, http.StatusBadGateway, "server_error", fmt.Sprintf("Upstream service returned status %d", resp.StatusCode))
		return
	}

	for key, values := range resp.Header {
		for _, value := range values {
			c.Writer.Header().Add(key, value)
		}
	}
	c.Writer.Header().Set("Cache-Control", "public, max-age=86400")
	c.Writer.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(c.Writer, resp.Body); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to stream image content: %s", err.Error()))
	}
}

func contextWithImageTimeout(c *gin.Context) (context.Context, context.CancelFunc) {
	if c == nil || c.Request == nil {
		return context.WithTimeout(context.Background(), 60*time.Second)
	}
	return context.WithTimeout(c.Request.Context(), 60*time.Second)
}

func imageGenerationProxyError(c *gin.Context, status int, errType, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"message": message,
			"type":    errType,
		},
	})
}
