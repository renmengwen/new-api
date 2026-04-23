package dto

type AsyncExportJobResponse struct {
	Id           int64  `json:"id"`
	JobType      string `json:"job_type"`
	Status       string `json:"status"`
	FileName     string `json:"file_name"`
	FileSize     int64  `json:"file_size"`
	RowCount     int64  `json:"row_count"`
	ErrorMessage string `json:"error_message"`
	CreatedAt    int64  `json:"created_at"`
	CompletedAt  int64  `json:"completed_at"`
	ExpiresAt    int64  `json:"expires_at"`
	StatusURL    string `json:"status_url"`
	DownloadURL  string `json:"download_url"`
}
