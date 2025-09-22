package prompt

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

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

	prompt := &domain.Prompt{
		ID:        uuid.NewString(),
		Name:      name,
		Tags:      tagsJSON,
		CreatedBy: optionalString(input.CreatedBy),
	}
	prompt.Description = optionalTrimmedString(input.Description)

	if err := s.repos.Prompts.Create(ctx, prompt); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrPromptAlreadyExists
		}
		return nil, err
	}

	created, err := s.repos.Prompts.GetByID(ctx, prompt.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrPromptNotFound
		}
		return nil, err
	}
	return created, nil
}

// ListPrompts 返回 Prompt 列表。
func (s *Service) ListPrompts(ctx context.Context, limit, offset int) ([]*domain.Prompt, error) {
	prompts, err := s.repos.Prompts.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	return prompts, nil
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

	if input.Activate {
		if err := s.repos.Prompts.UpdateActiveVersion(ctx, prompt.ID, &version.ID); err != nil {
			return nil, err
		}
	}

	created, err := s.repos.PromptVersions.GetByID(ctx, version.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrVersionNotFound
		}
		return nil, err
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

	if _, err := s.repos.PromptVersions.GetByID(ctx, versionID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return ErrVersionNotFound
		}
		return err
	}

	return s.repos.Prompts.UpdateActiveVersion(ctx, promptID, &versionID)
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
