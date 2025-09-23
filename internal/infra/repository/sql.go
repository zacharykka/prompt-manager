package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/zacharykka/prompt-manager/internal/domain"
	"github.com/zacharykka/prompt-manager/internal/infra/database"
)

// NewSQLRepositories 构建基于 *sql.DB 的仓储集合。
func NewSQLRepositories(db *sql.DB, dialect database.Dialect) *domain.Repositories {
	userRepo := &userRepository{db: db, dialect: dialect}
	promptRepo := &promptRepository{db: db, dialect: dialect}
	promptVersionRepo := &promptVersionRepository{db: db, dialect: dialect}
	execLogRepo := &promptExecutionLogRepository{db: db, dialect: dialect}

	return &domain.Repositories{
		Users:              userRepo,
		Prompts:            promptRepo,
		PromptVersions:     promptVersionRepo,
		PromptExecutionLog: execLogRepo,
	}
}

// ---- 用户仓储 ----

type userRepository struct {
	db      *sql.DB
	dialect database.Dialect
}

type userRow struct {
	id             string
	email          string
	hashedPassword string
	role           string
	status         string
	lastLoginAt    sql.NullTime
	createdAt      time.Time
	updatedAt      time.Time
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`INSERT INTO users (id, email, hashed_password, role, status)
VALUES (%s, %s, %s, %s, %s)`, ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next())

	role := user.Role
	if role == "" {
		role = "viewer"
	}
	status := user.Status
	if status == "" {
		status = "active"
	}

	_, err := r.db.ExecContext(ctx, query, user.ID, user.Email, user.HashedPassword, role, status)
	return err
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`SELECT id, email, hashed_password, role, status, last_login_at, created_at, updated_at
FROM users WHERE email = %s`, ph.Next())

	var row userRow
	err := r.db.QueryRowContext(ctx, query, email).Scan(&row.id, &row.email, &row.hashedPassword, &row.role, &row.status, &row.lastLoginAt, &row.createdAt, &row.updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	user := &domain.User{
		ID:             row.id,
		Email:          row.email,
		HashedPassword: row.hashedPassword,
		Role:           row.role,
		Status:         row.status,
		CreatedAt:      row.createdAt,
		UpdatedAt:      row.updatedAt,
	}
	if row.lastLoginAt.Valid {
		user.LastLoginAt = &row.lastLoginAt.Time
	}
	return user, nil
}

func (r *userRepository) UpdateLastLogin(ctx context.Context, userID string) error {
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`UPDATE users SET last_login_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = %s`, ph.Next())

	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ---- Prompt 仓储 ----

type promptRepository struct {
	db      *sql.DB
	dialect database.Dialect
}

type promptRow struct {
	id              string
	name            string
	description     sql.NullString
	tags            sql.NullString
	activeVersionID sql.NullString
	activeBody      sql.NullString
	createdBy       sql.NullString
	createdAt       time.Time
	updatedAt       time.Time
}

func (r *promptRepository) Create(ctx context.Context, prompt *domain.Prompt) error {
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`INSERT INTO prompts (id, name, description, tags, active_version_id, created_by)
VALUES (%s, %s, %s, %s, %s, %s)`, ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next())

	desc := sql.NullString{}
	if prompt.Description != nil {
		desc = sql.NullString{String: *prompt.Description, Valid: true}
	}
	tags := sql.NullString{}
	if len(prompt.Tags) > 0 {
		tags = sql.NullString{String: string(prompt.Tags), Valid: true}
	}
	active := sql.NullString{}
	if prompt.ActiveVersionID != nil {
		active = sql.NullString{String: *prompt.ActiveVersionID, Valid: true}
	}
	createdBy := sql.NullString{}
	if prompt.CreatedBy != nil {
		createdBy = sql.NullString{String: *prompt.CreatedBy, Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query, prompt.ID, prompt.Name, desc, tags, active, createdBy)
	return err
}

func (r *promptRepository) GetByID(ctx context.Context, promptID string) (*domain.Prompt, error) {
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`SELECT p.id, p.name, p.description, p.tags, p.active_version_id, pv.body, p.created_by, p.created_at, p.updated_at
FROM prompts p
LEFT JOIN prompt_versions pv ON pv.id = p.active_version_id
WHERE p.id = %s`, ph.Next())

	var row promptRow
	err := r.db.QueryRowContext(ctx, query, promptID).Scan(&row.id, &row.name, &row.description, &row.tags, &row.activeVersionID, &row.activeBody, &row.createdBy, &row.createdAt, &row.updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	prompt := &domain.Prompt{
		ID:        row.id,
		Name:      row.name,
		CreatedAt: row.createdAt,
		UpdatedAt: row.updatedAt,
	}
	if row.description.Valid {
		prompt.Description = &row.description.String
	}
	if row.tags.Valid {
		prompt.Tags = json.RawMessage(row.tags.String)
	}
	if row.activeVersionID.Valid {
		prompt.ActiveVersionID = &row.activeVersionID.String
	}
	if row.activeBody.Valid {
		prompt.ActiveVersionBody = &row.activeBody.String
	}
	if row.createdBy.Valid {
		prompt.CreatedBy = &row.createdBy.String
	}
	return prompt, nil
}

func (r *promptRepository) List(ctx context.Context, opts domain.PromptListOptions) ([]*domain.Prompt, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}
	search := strings.TrimSpace(strings.ToLower(opts.Search))

	ph := database.NewPlaceholderBuilder(r.dialect)
	var builder strings.Builder
	var args []interface{}

	builder.WriteString(`SELECT p.id, p.name, p.description, p.tags, p.active_version_id, pv.body, p.created_by, p.created_at, p.updated_at FROM prompts p`)
	builder.WriteString(` LEFT JOIN prompt_versions pv ON pv.id = p.active_version_id`)

	if search != "" {
		builder.WriteString(" WHERE LOWER(name) LIKE ")
		builder.WriteString(ph.Next())
		args = append(args, fmt.Sprintf("%%%s%%", search))
	}

	builder.WriteString(" ORDER BY updated_at DESC LIMIT ")
	builder.WriteString(ph.Next())
	builder.WriteString(" OFFSET ")
	builder.WriteString(ph.Next())

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, builder.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prompts []*domain.Prompt
	for rows.Next() {
		var row promptRow
		if err := rows.Scan(&row.id, &row.name, &row.description, &row.tags, &row.activeVersionID, &row.activeBody, &row.createdBy, &row.createdAt, &row.updatedAt); err != nil {
			return nil, err
		}
		prompt := &domain.Prompt{
			ID:        row.id,
			Name:      row.name,
			CreatedAt: row.createdAt,
			UpdatedAt: row.updatedAt,
		}
		if row.description.Valid {
			prompt.Description = &row.description.String
		}
		if row.tags.Valid {
			prompt.Tags = json.RawMessage(row.tags.String)
		}
		if row.activeVersionID.Valid {
			prompt.ActiveVersionID = &row.activeVersionID.String
		}
		if row.activeBody.Valid {
			prompt.ActiveVersionBody = &row.activeBody.String
		}
		if row.createdBy.Valid {
			prompt.CreatedBy = &row.createdBy.String
		}
		prompts = append(prompts, prompt)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return prompts, nil
}

func (r *promptRepository) UpdateActiveVersion(ctx context.Context, promptID string, versionID *string) error {
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`UPDATE prompts SET active_version_id = %s, updated_at = CURRENT_TIMESTAMP WHERE id = %s`, ph.Next(), ph.Next())

	active := sql.NullString{}
	if versionID != nil {
		active = sql.NullString{String: *versionID, Valid: true}
	}

	result, err := r.db.ExecContext(ctx, query, active, promptID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *promptRepository) Count(ctx context.Context, opts domain.PromptListOptions) (int64, error) {
	search := strings.TrimSpace(strings.ToLower(opts.Search))
	ph := database.NewPlaceholderBuilder(r.dialect)
	var builder strings.Builder
	var args []interface{}

	builder.WriteString("SELECT COUNT(1) FROM prompts")
	if search != "" {
		builder.WriteString(" WHERE LOWER(name) LIKE ")
		builder.WriteString(ph.Next())
		args = append(args, fmt.Sprintf("%%%s%%", search))
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, builder.String(), args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// ---- Prompt Version 仓储 ----

type promptVersionRepository struct {
	db      *sql.DB
	dialect database.Dialect
}

type promptVersionRow struct {
	id              string
	promptID        string
	versionNumber   int
	body            string
	variablesSchema sql.NullString
	status          string
	metadata        sql.NullString
	createdBy       sql.NullString
	createdAt       time.Time
}

func (r *promptVersionRepository) Create(ctx context.Context, version *domain.PromptVersion) error {
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`INSERT INTO prompt_versions (id, prompt_id, version_number, body, variables_schema, status, metadata, created_by)
VALUES (%s, %s, %s, %s, %s, %s, %s, %s)`, ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next())

	variables := sql.NullString{}
	if len(version.VariablesSchema) > 0 {
		variables = sql.NullString{String: string(version.VariablesSchema), Valid: true}
	}
	metadata := sql.NullString{}
	if len(version.Metadata) > 0 {
		metadata = sql.NullString{String: string(version.Metadata), Valid: true}
	}
	createdBy := sql.NullString{}
	if version.CreatedBy != nil {
		createdBy = sql.NullString{String: *version.CreatedBy, Valid: true}
	}

	status := version.Status
	if status == "" {
		status = "draft"
	}

	_, err := r.db.ExecContext(ctx, query, version.ID, version.PromptID, version.VersionNumber, version.Body, variables, status, metadata, createdBy)
	return err
}

func (r *promptVersionRepository) GetByID(ctx context.Context, versionID string) (*domain.PromptVersion, error) {
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`SELECT id, prompt_id, version_number, body, variables_schema, status, metadata, created_by, created_at
FROM prompt_versions WHERE id = %s`, ph.Next())

	var row promptVersionRow
	err := r.db.QueryRowContext(ctx, query, versionID).Scan(&row.id, &row.promptID, &row.versionNumber, &row.body, &row.variablesSchema, &row.status, &row.metadata, &row.createdBy, &row.createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	version := &domain.PromptVersion{
		ID:            row.id,
		PromptID:      row.promptID,
		VersionNumber: row.versionNumber,
		Body:          row.body,
		Status:        row.status,
		CreatedAt:     row.createdAt,
	}
	if row.variablesSchema.Valid {
		version.VariablesSchema = json.RawMessage(row.variablesSchema.String)
	}
	if row.metadata.Valid {
		version.Metadata = json.RawMessage(row.metadata.String)
	}
	if row.createdBy.Valid {
		version.CreatedBy = &row.createdBy.String
	}
	return version, nil
}

func (r *promptVersionRepository) ListByPrompt(ctx context.Context, promptID string, limit, offset int) ([]*domain.PromptVersion, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`SELECT id, prompt_id, version_number, body, variables_schema, status, metadata, created_by, created_at
FROM prompt_versions WHERE prompt_id = %s ORDER BY version_number DESC LIMIT %s OFFSET %s`, ph.Next(), ph.Next(), ph.Next())

	rows, err := r.db.QueryContext(ctx, query, promptID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []*domain.PromptVersion
	for rows.Next() {
		var row promptVersionRow
		if err := rows.Scan(&row.id, &row.promptID, &row.versionNumber, &row.body, &row.variablesSchema, &row.status, &row.metadata, &row.createdBy, &row.createdAt); err != nil {
			return nil, err
		}
		version := &domain.PromptVersion{
			ID:            row.id,
			PromptID:      row.promptID,
			VersionNumber: row.versionNumber,
			Body:          row.body,
			Status:        row.status,
			CreatedAt:     row.createdAt,
		}
		if row.variablesSchema.Valid {
			version.VariablesSchema = json.RawMessage(row.variablesSchema.String)
		}
		if row.metadata.Valid {
			version.Metadata = json.RawMessage(row.metadata.String)
		}
		if row.createdBy.Valid {
			version.CreatedBy = &row.createdBy.String
		}
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return versions, nil
}

func (r *promptVersionRepository) GetLatestVersionNumber(ctx context.Context, promptID string) (int, error) {
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`SELECT COALESCE(MAX(version_number), 0) FROM prompt_versions WHERE prompt_id = %s`, ph.Next())

	var latest sql.NullInt64
	if err := r.db.QueryRowContext(ctx, query, promptID).Scan(&latest); err != nil {
		return 0, err
	}
	if latest.Valid {
		return int(latest.Int64), nil
	}
	return 0, nil
}

// ---- 执行日志仓储 ----

type promptExecutionLogRepository struct {
	db      *sql.DB
	dialect database.Dialect
}

type executionLogRow struct {
	id               string
	promptID         string
	promptVersionID  string
	userID           sql.NullString
	status           string
	durationMs       sql.NullInt64
	requestPayload   sql.NullString
	responseMetadata sql.NullString
	createdAt        time.Time
}

type executionAggregateRow struct {
	dayStr       string
	totalCalls   int
	successCalls int
	averageMs    sql.NullFloat64
}

func (r *promptExecutionLogRepository) Create(ctx context.Context, log *domain.PromptExecutionLog) error {
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`INSERT INTO prompt_execution_logs (id, prompt_id, prompt_version_id, user_id, status, duration_ms, request_payload, response_metadata)
VALUES (%s, %s, %s, %s, %s, %s, %s, %s)`, ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next())

	userID := sql.NullString{}
	if log.UserID != nil {
		userID = sql.NullString{String: *log.UserID, Valid: true}
	}
	duration := sql.NullInt64{}
	if log.DurationMs != 0 {
		duration = sql.NullInt64{Int64: log.DurationMs, Valid: true}
	}
	request := sql.NullString{}
	if len(log.RequestPayload) > 0 {
		request = sql.NullString{String: string(log.RequestPayload), Valid: true}
	}
	response := sql.NullString{}
	if len(log.ResponseMetadata) > 0 {
		response = sql.NullString{String: string(log.ResponseMetadata), Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query, log.ID, log.PromptID, log.PromptVersionID, userID, log.Status, duration, request, response)
	return err
}

func (r *promptExecutionLogRepository) ListRecent(ctx context.Context, promptID string, limit int) ([]*domain.PromptExecutionLog, error) {
	if limit <= 0 {
		limit = 20
	}
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`SELECT id, prompt_id, prompt_version_id, user_id, status, duration_ms, request_payload, response_metadata, created_at
FROM prompt_execution_logs WHERE prompt_id = %s ORDER BY created_at DESC LIMIT %s`, ph.Next(), ph.Next())

	rows, err := r.db.QueryContext(ctx, query, promptID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*domain.PromptExecutionLog
	for rows.Next() {
		var row executionLogRow
		if err := rows.Scan(&row.id, &row.promptID, &row.promptVersionID, &row.userID, &row.status, &row.durationMs, &row.requestPayload, &row.responseMetadata, &row.createdAt); err != nil {
			return nil, err
		}
		log := &domain.PromptExecutionLog{
			ID:              row.id,
			PromptID:        row.promptID,
			PromptVersionID: row.promptVersionID,
			Status:          row.status,
			CreatedAt:       row.createdAt,
		}
		if row.userID.Valid {
			log.UserID = &row.userID.String
		}
		if row.durationMs.Valid {
			log.DurationMs = row.durationMs.Int64
		}
		if row.requestPayload.Valid {
			log.RequestPayload = json.RawMessage(row.requestPayload.String)
		}
		if row.responseMetadata.Valid {
			log.ResponseMetadata = json.RawMessage(row.responseMetadata.String)
		}
		logs = append(logs, log)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return logs, nil
}

func (r *promptExecutionLogRepository) AggregateUsage(ctx context.Context, promptID string, from time.Time) ([]*domain.PromptExecutionAggregate, error) {
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`SELECT DATE(created_at) as day,
        COUNT(*) as total_calls,
        SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_calls,
        AVG(duration_ms) as average_ms
      FROM prompt_execution_logs
      WHERE prompt_id = %s AND created_at >= %s
      GROUP BY DATE(created_at)
      ORDER BY DATE(created_at) DESC`, ph.Next(), ph.Next())

	rows, err := r.db.QueryContext(ctx, query, promptID, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*domain.PromptExecutionAggregate
	for rows.Next() {
		var row executionAggregateRow
		if err := rows.Scan(&row.dayStr, &row.totalCalls, &row.successCalls, &row.averageMs); err != nil {
			return nil, err
		}
		aggregate := &domain.PromptExecutionAggregate{
			TotalCalls:   row.totalCalls,
			SuccessCalls: row.successCalls,
		}
		if row.dayStr != "" {
			if parsed, err := time.Parse("2006-01-02", row.dayStr); err == nil {
				aggregate.Day = parsed
			}
		}
		if row.averageMs.Valid {
			aggregate.AverageMillis = row.averageMs.Float64
		}
		stats = append(stats, aggregate)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return stats, nil
}
