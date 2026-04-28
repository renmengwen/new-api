package controller

import (
	"context"
	"encoding/base64"
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

	payload := gin.H{
		"task_id":     task.TaskID,
		"platform":    "image",
		"action":      constant.TaskTypeImageGeneration,
		"status":      string(task.Status),
		"progress":    task.Progress,
		"result_url":  imageGenerationResultURL(c, task),
		"fail_reason": task.FailReason,
	}
	if imageGenerationWantsB64JSON(task) {
		if b64JSON := imageGenerationB64JSON(task); b64JSON != "" {
			payload["b64_json"] = b64JSON
		}
	}
	c.JSON(http.StatusOK, payload)
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

func imageGenerationWantsB64JSON(task *model.Task) bool {
	return task != nil && strings.EqualFold(strings.TrimSpace(task.Properties.ResponseFormat), "b64_json")
}

func imageGenerationB64JSON(task *model.Task) string {
	if task == nil || task.Status != model.TaskStatusSuccess {
		return ""
	}
	value := strings.TrimSpace(task.GetResultURL())
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "data:") {
		parts := strings.SplitN(value, ",", 2)
		if len(parts) != 2 {
			return ""
		}
		value = parts[1]
	}
	imageBytes, _, err := decodeImageBase64(value)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(imageBytes)
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
		if err := writeImageDataURL(c, imageURL); err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to decode image data URL for task %s: %s", taskID, err.Error()))
			imageGenerationProxyError(c, http.StatusBadGateway, "server_error", "Failed to fetch image content")
		}
		return
	}
	if handled, err := writeImageBase64Content(c, imageURL); handled {
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("Failed to decode image base64 for task %s: %s", taskID, err.Error()))
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

func writeImageDataURL(c *gin.Context, dataURL string) error {
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid data url")
	}

	header := parts[0]
	payload := parts[1]
	if !strings.HasPrefix(header, "data:") || !strings.Contains(header, ";base64") {
		return fmt.Errorf("unsupported data url")
	}

	mimeType := strings.TrimPrefix(header, "data:")
	if idx := strings.Index(mimeType, ";"); idx >= 0 {
		mimeType = mimeType[:idx]
	}
	imageBytes, detectedMimeType, err := decodeImageBase64(payload)
	if err != nil {
		return err
	}
	if strings.TrimSpace(mimeType) == "" {
		mimeType = detectedMimeType
	}
	if !strings.HasPrefix(mimeType, "image/") {
		return fmt.Errorf("unsupported image content type %s", mimeType)
	}
	return writeImageBytes(c, mimeType, imageBytes)
}

func writeImageBase64Content(c *gin.Context, value string) (bool, error) {
	imageBytes, mimeType, err := decodeImageBase64(value)
	if err != nil {
		return false, nil
	}
	return true, writeImageBytes(c, mimeType, imageBytes)
}

func decodeImageBase64(value string) ([]byte, string, error) {
	payload := compactBase64String(strings.TrimSpace(value))
	if payload == "" {
		return nil, "", fmt.Errorf("empty base64 image")
	}

	var lastErr error
	for _, encoding := range []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	} {
		imageBytes, err := encoding.DecodeString(payload)
		if err != nil {
			lastErr = err
			continue
		}
		mimeType := http.DetectContentType(imageBytes)
		if !strings.HasPrefix(mimeType, "image/") {
			return nil, "", fmt.Errorf("decoded base64 content is %s, not image", mimeType)
		}
		return imageBytes, mimeType, nil
	}
	if lastErr != nil {
		return nil, "", lastErr
	}
	return nil, "", fmt.Errorf("invalid base64 image")
}

func compactBase64String(value string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\n', '\r', '\t':
			return -1
		default:
			return r
		}
	}, value)
}

func writeImageBytes(c *gin.Context, mimeType string, imageBytes []byte) error {
	c.Writer.Header().Set("Content-Type", mimeType)
	c.Writer.Header().Set("Cache-Control", "public, max-age=86400")
	c.Writer.WriteHeader(http.StatusOK)
	_, err := c.Writer.Write(imageBytes)
	return err
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
