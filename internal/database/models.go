package database

import (
	"time"
)

type User struct {
	ID        uint      `gorm:"type:bigint unsigned;primarykey;autoIncrement;not null" json:"id"`
	Username  string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"username"`
	Password  string    `gorm:"type:varchar(255);not null" json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Task struct {
	ID         uint      `gorm:"type:bigint unsigned;primarykey;autoIncrement;not null" json:"id"`
	WasmModule string    `gorm:"type:text;not null" json:"wasm_module"`
	Func       string    `gorm:"type:varchar(255);not null" json:"func"`
	Args       string    `gorm:"type:text" json:"args"`
	CreatedBy  uint      `gorm:"type:bigint unsigned;not null;index" json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	Creator User `gorm:"foreignKey:CreatedBy;references:ID;constraint:OnDelete:RESTRICT;OnUpdate:CASCADE" json:"creator,omitempty"`
}

type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusProcessing TaskStatus = "processing"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
)

type TaskAudit struct {
	ID          uint       `gorm:"type:bigint unsigned;primarykey;autoIncrement;not null" json:"id"`
	TaskID      uint       `gorm:"type:bigint unsigned;not null;uniqueIndex" json:"task_id"`
	Status      TaskStatus `gorm:"type:varchar(50);default:'pending';not null;index" json:"status"`
	ProcessedBy *uint      `gorm:"type:bigint unsigned;index:idx_task_processed_by" json:"processed_by,omitempty"`
	PublishedAt time.Time  `gorm:"type:datetime;not null" json:"published_at"`
	ConsumedAt  *time.Time `gorm:"type:datetime" json:"consumed_at,omitempty"`
	CompletedAt *time.Time `gorm:"type:datetime" json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	Task   Task `gorm:"foreignKey:TaskID;references:ID;constraint:OnDelete:CASCADE;OnUpdate:CASCADE" json:"task,omitempty"`
	Worker User `gorm:"foreignKey:ProcessedBy;references:ID;constraint:OnDelete:SET NULL;OnUpdate:CASCADE" json:"worker,omitempty"`
}

func (TaskAudit) TableName() string {
	return "task_audit"
}

type Result struct {
	ID          uint      `gorm:"type:bigint unsigned;primarykey;autoIncrement;not null" json:"id"`
	TaskID      uint      `gorm:"type:bigint unsigned;not null;index" json:"task_id"`
	CreatedBy   uint      `gorm:"type:bigint unsigned;not null;index" json:"created_by"`
	ProcessedBy uint      `gorm:"type:bigint unsigned;not null;index" json:"processed_by"`
	Result      string    `gorm:"type:text;not null" json:"result"`
	Consumed    bool      `gorm:"type:boolean;default:false;not null;index" json:"consumed"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Task      Task `gorm:"foreignKey:TaskID;references:ID;constraint:OnDelete:CASCADE;OnUpdate:CASCADE" json:"task,omitempty"`
	Creator   User `gorm:"foreignKey:CreatedBy;references:ID;constraint:OnDelete:RESTRICT;OnUpdate:CASCADE" json:"creator,omitempty"`
	Processor User `gorm:"foreignKey:ProcessedBy;references:ID;constraint:OnDelete:RESTRICT;OnUpdate:CASCADE" json:"processor,omitempty"`
}
