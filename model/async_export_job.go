package model

const (
	AsyncExportStatusQueued    = "queued"
	AsyncExportStatusRunning   = "running"
	AsyncExportStatusSucceeded = "succeeded"
	AsyncExportStatusFailed    = "failed"
	AsyncExportStatusExpired   = "expired"
)

type AsyncExportJob struct {
	Id              int64  `json:"id"`
	JobType         string `json:"job_type" gorm:"type:varchar(64);index"`
	Status          string `json:"status" gorm:"type:varchar(32);index"`
	RequesterUserId int    `json:"requester_user_id" gorm:"index"`
	RequesterRole   int    `json:"requester_role"`
	FileName        string `json:"file_name" gorm:"type:varchar(255)"`
	FilePath        string `json:"file_path" gorm:"type:text"`
	FileSize        int64  `json:"file_size"`
	RowCount        int64  `json:"row_count"`
	ErrorMessage    string `json:"error_message" gorm:"type:text"`
	PayloadJSON     string `json:"payload_json" gorm:"type:text"`
	ResultJSON      string `json:"result_json" gorm:"type:text"`
	CreatedAtTs     int64  `json:"created_at" gorm:"column:created_at;bigint;index"`
	StartedAtTs     int64  `json:"started_at" gorm:"column:started_at;bigint"`
	CompletedAtTs   int64  `json:"completed_at" gorm:"column:completed_at;bigint"`
	ExpiresAtTs     int64  `json:"expires_at" gorm:"column:expires_at;bigint;index"`
}
