package domain

import "context"

// UserRepository 定义用户存取接口。
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	UpdateLastLogin(ctx context.Context, userID string) error
}

// PromptRepository 定义 Prompt 模板存取接口。
type PromptRepository interface {
	Create(ctx context.Context, prompt *Prompt) error
	GetByID(ctx context.Context, promptID string) (*Prompt, error)
	List(ctx context.Context, limit, offset int) ([]*Prompt, error)
	UpdateActiveVersion(ctx context.Context, promptID string, versionID *string) error
}

// PromptVersionRepository 定义 Prompt 版本存取接口。
type PromptVersionRepository interface {
	Create(ctx context.Context, version *PromptVersion) error
	GetByID(ctx context.Context, versionID string) (*PromptVersion, error)
	ListByPrompt(ctx context.Context, promptID string, limit, offset int) ([]*PromptVersion, error)
	GetLatestVersionNumber(ctx context.Context, promptID string) (int, error)
}

// PromptExecutionLogRepository 定义 Prompt 执行日志接口。
type PromptExecutionLogRepository interface {
	Create(ctx context.Context, log *PromptExecutionLog) error
	ListRecent(ctx context.Context, promptID string, limit int) ([]*PromptExecutionLog, error)
}

// Repositories 聚合全部仓储接口，便于依赖注入。
type Repositories struct {
	Users              UserRepository
	Prompts            PromptRepository
	PromptVersions     PromptVersionRepository
	PromptExecutionLog PromptExecutionLogRepository
}
