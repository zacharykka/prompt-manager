package prompt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
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

type JSONFieldChange struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	Left  string `json:"left,omitempty"`
	Right string `json:"right,omitempty"`
}

type FieldDiff struct {
	Changes []JSONFieldChange `json:"changes"`
}

type VersionSummary struct {
	ID            string    `json:"id"`
	VersionNumber int       `json:"version_number"`
	CreatedBy     *string   `json:"created_by,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	Status        string    `json:"status"`
}

type PromptVersionDiff struct {
	PromptID  string         `json:"prompt_id"`
	Base      VersionSummary `json:"base"`
	Target    VersionSummary `json:"target"`
	Body      []DiffSegment  `json:"body"`
	Variables *FieldDiff     `json:"variables_schema,omitempty"`
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
	leftMap := map[string]interface{}{}
	if len(leftRaw) > 0 {
		_ = json.Unmarshal(leftRaw, &leftMap)
	}
	rightMap := map[string]interface{}{}
	if len(rightRaw) > 0 {
		_ = json.Unmarshal(rightRaw, &rightMap)
	}

	keys := make(map[string]struct{})
	for key := range leftMap {
		keys[key] = struct{}{}
	}
	for key := range rightMap {
		keys[key] = struct{}{}
	}

	if len(keys) == 0 {
		return nil
	}

	changes := make([]JSONFieldChange, 0)
	for key := range keys {
		leftVal, leftOK := leftMap[key]
		rightVal, rightOK := rightMap[key]
		switch {
		case !leftOK && rightOK:
			changes = append(changes, JSONFieldChange{
				Key:   key,
				Type:  "added",
				Right: stringifyJSONValue(rightVal),
			})
		case leftOK && !rightOK:
			changes = append(changes, JSONFieldChange{
				Key:  key,
				Type: "removed",
				Left: stringifyJSONValue(leftVal),
			})
		default:
			leftString := stringifyJSONValue(leftVal)
			rightString := stringifyJSONValue(rightVal)
			if leftString == rightString {
				continue
			}
			changes = append(changes, JSONFieldChange{
				Key:   key,
				Type:  "modified",
				Left:  leftString,
				Right: rightString,
			})
		}
	}

	if len(changes) == 0 {
		return nil
	}

	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Key < changes[j].Key
	})

	return &FieldDiff{Changes: changes}
}

func stringifyJSONValue(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case float64, bool, int, int64, json.Number:
		return fmt.Sprintf("%v", v)
	default:
		bytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(bytes)
	}
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
