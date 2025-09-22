package domain

import "context"

// TenantRepository 定义租户相关的存取接口。
type TenantRepository interface {
	Create(ctx context.Context, tenant *Tenant) error
	GetByID(ctx context.Context, tenantID string) (*Tenant, error)
	List(ctx context.Context, limit, offset int) ([]*Tenant, error)
}

// UserRepository 定义用户存取接口。
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, tenantID, email string) (*User, error)
	UpdateLastLogin(ctx context.Context, tenantID, userID string) error
}

// PromptRepository 定义 Prompt 模板存取接口。
type PromptRepository interface {
	Create(ctx context.Context, prompt *Prompt) error
	GetByID(ctx context.Context, tenantID, promptID string) (*Prompt, error)
	ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*Prompt, error)
	UpdateActiveVersion(ctx context.Context, tenantID, promptID string, versionID *string) error
}

// PromptVersionRepository 定义 Prompt 版本存取接口。
type PromptVersionRepository interface {
	Create(ctx context.Context, version *PromptVersion) error
	GetByID(ctx context.Context, tenantID, versionID string) (*PromptVersion, error)
	ListByPrompt(ctx context.Context, tenantID, promptID string, limit, offset int) ([]*PromptVersion, error)
	GetLatestVersionNumber(ctx context.Context, tenantID, promptID string) (int, error)
}

// PromptExecutionLogRepository 定义 Prompt 执行日志接口。
type PromptExecutionLogRepository interface {
	Create(ctx context.Context, log *PromptExecutionLog) error
	ListRecent(ctx context.Context, tenantID, promptID string, limit int) ([]*PromptExecutionLog, error)
}

// Repositories 聚合全部仓储接口，便于依赖注入。
type Repositories struct {
	Tenants            TenantRepository
	Users              UserRepository
	Prompts            PromptRepository
	PromptVersions     PromptVersionRepository
	PromptExecutionLog PromptExecutionLogRepository
}
