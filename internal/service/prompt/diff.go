package prompt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/sergi/go-diff/diffmatchpatch"
	domain "github.com/zacharykka/prompt-manager/internal/domain"
)

type DiffPromptVersionOptions struct {
	TargetVersionID   *string
	CompareToActive   bool
	CompareToPrevious bool
}

type DiffSegment struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type FieldDiff struct {
	Left    string `json:"left"`
	Right   string `json:"right"`
	Changed bool   `json:"changed"`
}

type VersionSummary struct {
	ID            string    `json:"id"`
	VersionNumber int       `json:"versionNumber"`
	CreatedBy     *string   `json:"createdBy,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	Status        string    `json:"status"`
}

type PromptVersionDiff struct {
	PromptID  string         `json:"promptId"`
	Base      VersionSummary `json:"base"`
	Target    VersionSummary `json:"target"`
	Body      []DiffSegment  `json:"body"`
	Variables *FieldDiff     `json:"variablesSchema,omitempty"`
	Metadata  *FieldDiff     `json:"metadata,omitempty"`
}

func (s *Service) DiffPromptVersion(ctx context.Context, promptID, baseVersionID string, opts DiffPromptVersionOptions) (*PromptVersionDiff, error) {
	base, err := s.repos.PromptVersions.GetByID(ctx, baseVersionID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrVersionNotFound
		}
		return nil, err
	}
	if base.PromptID != promptID {
		return nil, ErrVersionNotFound
	}

	target, err := s.resolveDiffTarget(ctx, promptID, base, opts)
	if err != nil {
		return nil, err
	}

	diff := &PromptVersionDiff{
		PromptID: promptID,
		Base:     summarizeVersion(base),
		Target:   summarizeVersion(target),
		Body:     buildBodyDiff(target.Body, base.Body),
	}

	if fieldDiff := buildFieldDiff(target.VariablesSchema, base.VariablesSchema); fieldDiff != nil {
		diff.Variables = fieldDiff
	}
	if fieldDiff := buildFieldDiff(target.Metadata, base.Metadata); fieldDiff != nil {
		diff.Metadata = fieldDiff
	}

	return diff, nil
}

func (s *Service) resolveDiffTarget(ctx context.Context, promptID string, base *domain.PromptVersion, opts DiffPromptVersionOptions) (*domain.PromptVersion, error) {
	if opts.TargetVersionID != nil {
		version, err := s.repos.PromptVersions.GetByID(ctx, *opts.TargetVersionID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, ErrVersionNotFound
			}
			return nil, err
		}
		if version.PromptID != promptID {
			return nil, ErrVersionNotFound
		}
		return version, nil
	}

	if opts.CompareToActive {
		prompt, err := s.repos.Prompts.GetByID(ctx, promptID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, ErrPromptNotFound
			}
			return nil, err
		}
		if prompt.ActiveVersionID == nil {
			return nil, ErrVersionNotFound
		}
		version, err := s.repos.PromptVersions.GetByID(ctx, *prompt.ActiveVersionID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, ErrVersionNotFound
			}
			return nil, err
		}
		return version, nil
	}

	if !opts.CompareToPrevious {
		opts.CompareToPrevious = true
	}

	if opts.CompareToPrevious {
		previous, err := s.repos.PromptVersions.GetPreviousVersion(ctx, promptID, base.VersionNumber)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil, ErrVersionNotFound
			}
			return nil, err
		}
		return previous, nil
	}

	return nil, ErrVersionNotFound
}

func buildBodyDiff(left, right string) []DiffSegment {
	dmp := diffmatchpatch.New()
	patches := dmp.DiffMain(left, right, false)
	dmp.DiffCleanupSemantic(patches)

	segments := make([]DiffSegment, 0, len(patches))
	for _, piece := range patches {
		segType := "equal"
		switch piece.Type {
		case diffmatchpatch.DiffDelete:
			segType = "delete"
		case diffmatchpatch.DiffInsert:
			segType = "insert"
		}
		if piece.Text == "" {
			continue
		}
		segments = append(segments, DiffSegment{Type: segType, Text: piece.Text})
	}
	return segments
}

func buildFieldDiff(leftRaw, rightRaw json.RawMessage) *FieldDiff {
	left := normalizeJSON(leftRaw)
	right := normalizeJSON(rightRaw)
	if left == right {
		return nil
	}
	return &FieldDiff{Left: left, Right: right, Changed: true}
}

func normalizeJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		return strings.TrimSpace(string(raw))
	}
	return buf.String()
}

func summarizeVersion(version *domain.PromptVersion) VersionSummary {
	return VersionSummary{
		ID:            version.ID,
		VersionNumber: version.VersionNumber,
		CreatedBy:     version.CreatedBy,
		CreatedAt:     version.CreatedAt,
		Status:        version.Status,
	}
}
