package domain

import (
	"context"
	"time"
)

// UserRepository 定义用户存取接口。
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, userID string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	UpdateLastLogin(ctx context.Context, userID string) error
}

// UserIdentityRepository 负责外部身份与本地用户的映射。
type UserIdentityRepository interface {
	Create(ctx context.Context, identity *UserIdentity) error
	GetByProviderAndExternalID(ctx context.Context, provider, externalID string) (*UserIdentity, error)
}

// PromptRepository 定义 Prompt 模板存取接口。
type PromptRepository interface {
	Create(ctx context.Context, prompt *Prompt) error
	GetByID(ctx context.Context, promptID string) (*Prompt, error)
	GetByIDIncludeDeleted(ctx context.Context, promptID string) (*Prompt, error)
	GetByName(ctx context.Context, name string, includeDeleted bool) (*Prompt, error)
	List(ctx context.Context, opts PromptListOptions) ([]*Prompt, error)
	Count(ctx context.Context, opts PromptListOptions) (int64, error)
	UpdateActiveVersion(ctx context.Context, promptID string, versionID *string, body *string) error
	Update(ctx context.Context, promptID string, params PromptUpdateParams) error
	Delete(ctx context.Context, promptID string) error
	Restore(ctx context.Context, promptID string, params PromptRestoreParams) error
}

// PromptVersionRepository 定义 Prompt 版本存取接口。
type PromptVersionRepository interface {
	Create(ctx context.Context, version *PromptVersion) error
	GetByID(ctx context.Context, versionID string) (*PromptVersion, error)
	ListByPrompt(ctx context.Context, promptID string, limit, offset int) ([]*PromptVersion, error)
	// ListByPromptAndStatus 基于状态过滤版本列表（如 draft/published/archived）。
	ListByPromptAndStatus(ctx context.Context, promptID string, status string, limit, offset int) ([]*PromptVersion, error)
	// CountByPrompt 统计指定 Prompt 的版本总数。
	CountByPrompt(ctx context.Context, promptID string) (int64, error)
	// CountByPromptAndStatus 统计指定 Prompt 在某状态下的版本总数。
	CountByPromptAndStatus(ctx context.Context, promptID string, status string) (int64, error)
	GetLatestVersionNumber(ctx context.Context, promptID string) (int, error)
	GetPreviousVersion(ctx context.Context, promptID string, versionNumber int) (*PromptVersion, error)
}

// PromptExecutionLogRepository 定义 Prompt 执行日志接口。
type PromptExecutionLogRepository interface {
	Create(ctx context.Context, log *PromptExecutionLog) error
	ListRecent(ctx context.Context, promptID string, limit int) ([]*PromptExecutionLog, error)
	AggregateUsage(ctx context.Context, promptID string, from time.Time) ([]*PromptExecutionAggregate, error)
}

// PromptAuditLogRepository 定义 Prompt 审计日志存取接口。
type PromptAuditLogRepository interface {
	Create(ctx context.Context, log *PromptAuditLog) error
	ListByPrompt(ctx context.Context, promptID string, limit int) ([]*PromptAuditLog, error)
}

// Repositories 聚合全部仓储接口，便于依赖注入。
type Repositories struct {
	Users              UserRepository
	UserIdentities     UserIdentityRepository
	Prompts            PromptRepository
	PromptVersions     PromptVersionRepository
	PromptExecutionLog PromptExecutionLogRepository
	PromptAuditLog     PromptAuditLogRepository
}

// PromptListOptions 定义 Prompt 列表过滤与分页参数。
type PromptListOptions struct {
	Limit          int
	Offset         int
	Search         string
	IncludeDeleted bool
}

// PromptUpdateParams 描述 Prompt 更新操作的可选字段。
type PromptUpdateParams struct {
	Name           *string
	Description    *string
	Tags           *string
	HasName        bool
	HasDescription bool
	HasTags        bool
}

// PromptRestoreParams 描述 Prompt 恢复时需要更新的字段。
type PromptRestoreParams struct {
	Description    *string
	Tags           *string
	CreatedBy      *string
	Body           *string
	HasDescription bool
	HasTags        bool
	HasCreatedBy   bool
	HasBody        bool
}
