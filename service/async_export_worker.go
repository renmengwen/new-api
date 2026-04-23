package service

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
)

const (
	asyncExportWorkerTick  = 2 * time.Second
	asyncExportCleanupTick = 30 * time.Minute
)

var (
	asyncExportWorkerOnce    sync.Once
	asyncExportWorkerRunning atomic.Bool
)

func StartAsyncExportWorker() {
	asyncExportWorkerOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}

		gopool.Go(func() {
			processTicker := time.NewTicker(asyncExportWorkerTick)
			cleanupTicker := time.NewTicker(asyncExportCleanupTick)
			defer processTicker.Stop()
			defer cleanupTicker.Stop()

			runAsyncExportWorkerOnce()
			for {
				select {
				case <-processTicker.C:
					runAsyncExportWorkerOnce()
				case <-cleanupTicker.C:
					runAsyncExportCleanupOnce()
				}
			}
		})
	})
}

func runAsyncExportWorkerOnce() {
	if !asyncExportWorkerRunning.CompareAndSwap(false, true) {
		return
	}
	defer asyncExportWorkerRunning.Store(false)

	job, err := ClaimNextAsyncExportJob()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return
		}
		common.SysError("claim async export job failed: " + err.Error())
		return
	}

	executor := GetAsyncExportExecutor(job.JobType)
	if executor == nil {
		_ = FailAsyncExportJob(job.Id, fmt.Sprintf("async export executor not registered: %s", job.JobType))
		return
	}

	if err := executor(job); err != nil {
		_ = FailAsyncExportJob(job.Id, err.Error())
	}
}

func runAsyncExportCleanupOnce() {
	if _, err := CleanupExpiredAsyncExportJobs(common.GetTimestamp()); err != nil {
		common.SysError("cleanup async export jobs failed: " + err.Error())
	}
}
