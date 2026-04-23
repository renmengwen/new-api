package controller

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func GetAsyncExportJob(c *gin.Context) {
	jobID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		common.ApiErrorMsg(c, "invalid export job id")
		return
	}

	job, err := service.GetAuthorizedAsyncExportJob(jobID, c.GetInt("id"), c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, service.BuildAsyncExportJobResponse(job))
}

func DownloadAsyncExportJobFile(c *gin.Context) {
	jobID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		common.ApiErrorMsg(c, "invalid export job id")
		return
	}

	job, err := service.GetAuthorizedAsyncExportJob(jobID, c.GetInt("id"), c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if job.Status != model.AsyncExportStatusSucceeded {
		common.ApiErrorMsg(c, "export file is not ready")
		return
	}
	c.FileAttachment(job.FilePath, job.FileName)
}

func respondAsyncExportJobCreated(c *gin.Context, decision service.SmartExportDecision, job *model.AsyncExportJob) {
	common.ApiSuccess(c, gin.H{
		"mode":     service.SmartExportModeAsync,
		"decision": decision,
		"job":      service.BuildAsyncExportJobResponse(job),
	})
}
