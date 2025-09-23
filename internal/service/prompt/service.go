package prompt

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	domain "github.com/zacharykka/prompt-manager/internal/domain"
)

// Service 提供 Prompt 领域相关操作。
type Service struct {
	repos *domain.Repositories
}

// NewService 创建 Prompt 服务实例。
func NewService(repos *domain.Repositories) *Service {
	return &Service{repos: repos}
}

// CreatePromptInput 定义创建 Prompt 所需的字段。
type CreatePromptInput struct {
	Name        string
	Description *string
	Tags        []string
	CreatedBy   string
}

// UpdatePromptInput 定义更新 Prompt 所需的可选字段。
type UpdatePromptInput struct {
	PromptID    string
	Name        *string
	Description *string
	Tags        *[]string
}

// CreatePrompt 创建新的 Prompt 记录。
func (s *Service) CreatePrompt(ctx context.Context, input CreatePromptInput) (*domain.Prompt, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrNameRequired
	}

	var tagsJSON json.RawMessage
	if len(input.Tags) > 0 {
		data, err := json.Marshal(input.Tags)
		if err != nil {
			return nil, err
		}
		tagsJSON = data
	}

	existing, err := s.repos.Prompts.GetByName(ctx, name, true)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	createdBy := optionalString(input.CreatedBy)
	description := optionalTrimmedString(input.Description)

	var created *domain.Prompt

	if existing != nil && existing.Status == "deleted" {
		restoreParams := domain.PromptRestoreParams{
			Description: description,
			CreatedBy:   createdBy,
		}
		if len(tagsJSON) > 0 {
			tagsStr := string(tagsJSON)
			restoreParams.Tags = &tagsStr
		}
		restoreParams.Body = nil

		if err := s.repos.Prompts.Restore(ctx, existing.ID, restoreParams); err != nil {
			return nil, err
		}

		restored, err := s.repos.Prompts.GetByID(ctx, existing.ID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, ErrPromptNotFound
			}
			return nil, err
		}
		if len(tagsJSON) > 0 {
			restored.Tags = tagsJSON
		} else {
			restored.Tags = nil
		}
		restored.Description = description
		restored.CreatedBy = createdBy
		created = restored
	} else if existing != nil {
		return nil, ErrPromptAlreadyExists
	} else {
		prompt := &domain.Prompt{
			ID:        uuid.NewString(),
			Name:      name,
			Tags:      tagsJSON,
			CreatedBy: createdBy,
		}
		prompt.Description = description

		if err := s.repos.Prompts.Create(ctx, prompt); err != nil {
			if isUniqueViolation(err) {
				return nil, ErrPromptAlreadyExists
			}
			return nil, err
		}

		created, err = s.repos.Prompts.GetByID(ctx, prompt.ID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, ErrPromptNotFound
			}
			return nil, err
		}
	}

	if len(tagsJSON) > 0 {
		created.Tags = tagsJSON
	} else {
		created.Tags = nil
	}
	created.Description = description
	created.CreatedBy = createdBy

	if created == nil {
		return nil, ErrPromptNotFound
	}

	return created, nil
}

// ListPrompts 返回 Prompt 列表。
// ListPromptsOptions 控制 Prompt 列表查询行为。
type ListPromptsOptions struct {
	Limit          int
	Offset         int
	Search         string
	IncludeDeleted bool
}

// ListPrompts 返回 Prompt 列表及总数。
func (s *Service) ListPrompts(ctx context.Context, opts ListPromptsOptions) ([]*domain.Prompt, int64, error) {
	repoOpts := domain.PromptListOptions{
		Limit:          opts.Limit,
		Offset:         opts.Offset,
		Search:         strings.TrimSpace(opts.Search),
		IncludeDeleted: opts.IncludeDeleted,
	}

	prompts, err := s.repos.Prompts.List(ctx, repoOpts)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repos.Prompts.Count(ctx, repoOpts)
	if err != nil {
		return nil, 0, err
	}

	return prompts, total, nil
}

// UpdatePrompt 更新 Prompt 元数据。
func (s *Service) UpdatePrompt(ctx context.Context, input UpdatePromptInput) (*domain.Prompt, error) {
	updates := domain.PromptUpdateParams{}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, ErrNameRequired
		}
		updates.HasName = true
		updates.Name = &name
	}

	if input.Description != nil {
		updates.HasDescription = true
		updates.Description = optionalTrimmedString(input.Description)
	}

	if input.Tags != nil {
		updates.HasTags = true
		if *input.Tags != nil {
			data, err := json.Marshal(*input.Tags)
			if err != nil {
				return nil, err
			}
			tagsStr := string(data)
			updates.Tags = &tagsStr
		}
	}

	if !updates.HasName && !updates.HasDescription && !updates.HasTags {
		return nil, ErrNoFieldsToUpdate
	}

	if err := s.repos.Prompts.Update(ctx, input.PromptID, updates); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrPromptNotFound
		}
		if isUniqueViolation(err) {
			return nil, ErrPromptAlreadyExists
		}
		return nil, err
	}

	return s.GetPrompt(ctx, input.PromptID)
}

// GetPrompt 根据 ID 获取 Prompt。
func (s *Service) GetPrompt(ctx context.Context, promptID string) (*domain.Prompt, error) {
	prompt, err := s.repos.Prompts.GetByID(ctx, promptID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrPromptNotFound
		}
		return nil, err
	}
	return prompt, nil
}

// CreatePromptVersionInput 定义创建 Prompt 版本所需字段。
type CreatePromptVersionInput struct {
	PromptID        string
	Body            string
	VariablesSchema interface{}
	Metadata        interface{}
	Status          string
	CreatedBy       string
	Activate        bool
}

// CreatePromptVersion 创建新的 Prompt 版本记录。
func (s *Service) CreatePromptVersion(ctx context.Context, input CreatePromptVersionInput) (*domain.PromptVersion, error) {
	prompt, err := s.GetPrompt(ctx, input.PromptID)
	if err != nil {
		return nil, err
	}

	body := strings.TrimSpace(input.Body)
	if body == "" {
		return nil, ErrBodyRequired
	}

	latest, err := s.repos.PromptVersions.GetLatestVersionNumber(ctx, prompt.ID)
	if err != nil {
		return nil, err
	}

	version := &domain.PromptVersion{
		ID:            uuid.NewString(),
		PromptID:      prompt.ID,
		VersionNumber: latest + 1,
		Body:          body,
		Status:        normalizedStatus(input.Status),
		CreatedBy:     optionalString(input.CreatedBy),
	}

	if input.VariablesSchema != nil {
		data, err := json.Marshal(input.VariablesSchema)
		if err != nil {
			return nil, err
		}
		version.VariablesSchema = data
	}
	if input.Metadata != nil {
		data, err := json.Marshal(input.Metadata)
		if err != nil {
			return nil, err
		}
		version.Metadata = data
	}

	if err := s.repos.PromptVersions.Create(ctx, version); err != nil {
		return nil, err
	}

	created, err := s.repos.PromptVersions.GetByID(ctx, version.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrVersionNotFound
		}
		return nil, err
	}

	if input.Activate {
		body := created.Body
		if err := s.repos.Prompts.UpdateActiveVersion(ctx, prompt.ID, &created.ID, &body); err != nil {
			return nil, err
		}
	}
	return created, nil
}

// ListPromptVersions 返回指定 Prompt 的版本列表。
func (s *Service) ListPromptVersions(ctx context.Context, promptID string, limit, offset int) ([]*domain.PromptVersion, error) {
	_, err := s.GetPrompt(ctx, promptID)
	if err != nil {
		return nil, err
	}

	versions, err := s.repos.PromptVersions.ListByPrompt(ctx, promptID, limit, offset)
	if err != nil {
		return nil, err
	}
	return versions, nil
}

// SetActiveVersion 将指定版本设为当前启用版本。
func (s *Service) SetActiveVersion(ctx context.Context, promptID, versionID string) error {
	_, err := s.GetPrompt(ctx, promptID)
	if err != nil {
		return err
	}

	version, err := s.repos.PromptVersions.GetByID(ctx, versionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return ErrVersionNotFound
		}
		return err
	}

	body := version.Body
	return s.repos.Prompts.UpdateActiveVersion(ctx, promptID, &versionID, &body)
}

// GetExecutionStats 返回最近若干天的执行统计。
func (s *Service) GetExecutionStats(ctx context.Context, promptID string, days int) ([]*domain.PromptExecutionAggregate, error) {
	if days <= 0 {
		days = 7
	}

	if _, err := s.GetPrompt(ctx, promptID); err != nil {
		return nil, err
	}

	from := time.Now().AddDate(0, 0, -days)
	stats, err := s.repos.PromptExecutionLog.AggregateUsage(ctx, promptID, from)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

// DeletePrompt 删除指定 Prompt（软删除），并记录审计日志。
func (s *Service) DeletePrompt(ctx context.Context, promptID, deletedBy string) error {
	if err := s.repos.Prompts.Delete(ctx, promptID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return ErrPromptNotFound
		}
		return err
	}

	if s.repos.PromptAuditLog != nil {
		actor := optionalString(deletedBy)
		payload, err := json.Marshal(map[string]string{
			"status": "deleted",
		})
		if err != nil {
			return err
		}
		audit := &domain.PromptAuditLog{
			ID:        uuid.NewString(),
			PromptID:  promptID,
			Action:    "prompt.deleted",
			Payload:   payload,
			CreatedBy: actor,
		}
		if err := s.repos.PromptAuditLog.Create(ctx, audit); err != nil {
			return err
		}
	}
	return nil
}

func optionalString(val string) *string {
	trimmed := strings.TrimSpace(val)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func optionalTrimmedString(val *string) *string {
	if val == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*val)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizedStatus(status string) string {
	value := strings.TrimSpace(strings.ToLower(status))
	switch value {
	case "published", "draft", "archived":
		return value
	default:
		return "draft"
	}
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// SQLite uses error strings for unique constraint failures; in PostgreSQL we can inspect error codes later.
	return strings.Contains(strings.ToLower(err.Error()), "unique")
}
