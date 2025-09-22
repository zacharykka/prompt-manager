package domain

import (
	"encoding/json"
	"time"
)

// User 表示系统中的登录用户。
type User struct {
	ID             string     `json:"id"`
	Email          string     `json:"email"`
	HashedPassword string     `json:"-"`
	Role           string     `json:"role"`
	Status         string     `json:"status"`
	LastLoginAt    *time.Time `json:"last_login_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Prompt 定义 Prompt 模板的元数据。
type Prompt struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Description     *string         `json:"description,omitempty"`
	Tags            json.RawMessage `json:"tags,omitempty"`
	ActiveVersionID *string         `json:"active_version_id,omitempty"`
	CreatedBy       *string         `json:"created_by,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// PromptVersion 记录 Prompt 的具体模板内容与变量信息。
type PromptVersion struct {
	ID              string          `json:"id"`
	PromptID        string          `json:"prompt_id"`
	VersionNumber   int             `json:"version_number"`
	Body            string          `json:"body"`
	VariablesSchema json.RawMessage `json:"variables_schema,omitempty"`
	Status          string          `json:"status"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
	CreatedBy       *string         `json:"created_by,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// PromptExecutionLog 记录 Prompt 运行时日志。
type PromptExecutionLog struct {
	ID               string          `json:"id"`
	PromptID         string          `json:"prompt_id"`
	PromptVersionID  string          `json:"prompt_version_id"`
	UserID           *string         `json:"user_id,omitempty"`
	Status           string          `json:"status"`
	DurationMs       int64           `json:"duration_ms"`
	RequestPayload   json.RawMessage `json:"request_payload,omitempty"`
	ResponseMetadata json.RawMessage `json:"response_metadata,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
}

// PromptExecutionAggregate 描述某一时间区间的统计信息。
type PromptExecutionAggregate struct {
	Day           time.Time `json:"day"`
	TotalCalls    int       `json:"total_calls"`
	SuccessCalls  int       `json:"success_calls"`
	AverageMillis float64   `json:"average_ms"`
}
