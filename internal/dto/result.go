package dto

type Result struct {
	TaskID    uint   `json:"task_id"`
	CreatedBy uint   `json:"created_by"`
	Result    any    `json:"result"`
}

