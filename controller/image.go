package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"

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

	dto := relay.TaskModel2Dto(task)
	c.JSON(http.StatusOK, gin.H{
		"id":         dto.TaskID,
		"task_id":    dto.TaskID,
		"object":     "image.generation.task",
		"created":    dto.SubmitTime,
		"model":      task.Properties.OriginModelName,
		"status":     dto.Status,
		"progress":   dto.Progress,
		"result_url": dto.ResultURL,
		"error":      dto.FailReason,
		"data":       dto.Data,
	})
}
