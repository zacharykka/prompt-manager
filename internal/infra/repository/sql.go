package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zacharykka/prompt-manager/internal/domain"
	"github.com/zacharykka/prompt-manager/internal/infra/database"
)

// NewSQLRepositories 构建基于 *sql.DB 的仓储集合。
func NewSQLRepositories(db *sql.DB, dialect database.Dialect) *domain.Repositories {
	tenantRepo := &tenantRepository{db: db, dialect: dialect}
	userRepo := &userRepository{db: db, dialect: dialect}
	promptRepo := &promptRepository{db: db, dialect: dialect}
	promptVersionRepo := &promptVersionRepository{db: db, dialect: dialect}
	execLogRepo := &promptExecutionLogRepository{db: db, dialect: dialect}

	return &domain.Repositories{
		Tenants:            tenantRepo,
		Users:              userRepo,
		Prompts:            promptRepo,
		PromptVersions:     promptVersionRepo,
		PromptExecutionLog: execLogRepo,
	}
}

type tenantRepository struct {
	db      *sql.DB
	dialect database.Dialect
}

type tenantRow struct {
	id          string
	name        string
	description sql.NullString
	status      string
	createdAt   time.Time
	updatedAt   time.Time
}

func (r *tenantRepository) Create(ctx context.Context, tenant *domain.Tenant) error {
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`INSERT INTO tenants (id, name, description, status)
VALUES (%s, %s, %s, %s)`, ph.Next(), ph.Next(), ph.Next(), ph.Next())

	desc := sql.NullString{}
	if tenant.Description != nil {
		desc = sql.NullString{String: *tenant.Description, Valid: true}
	}

	status := tenant.Status
	if status == "" {
		status = "active"
	}

	_, err := r.db.ExecContext(ctx, query, tenant.ID, tenant.Name, desc, status)
	return err
}

func (r *tenantRepository) GetByID(ctx context.Context, tenantID string) (*domain.Tenant, error) {
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`SELECT id, name, description, status, created_at, updated_at FROM tenants WHERE id = %s`, ph.Next())

	var row tenantRow
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(&row.id, &row.name, &row.description, &row.status, &row.createdAt, &row.updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	tenant := &domain.Tenant{
		ID:        row.id,
		Name:      row.name,
		Status:    row.status,
		CreatedAt: row.createdAt,
		UpdatedAt: row.updatedAt,
	}
	if row.description.Valid {
		tenant.Description = &row.description.String
	}
	return tenant, nil
}

func (r *tenantRepository) List(ctx context.Context, limit, offset int) ([]*domain.Tenant, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`SELECT id, name, description, status, created_at, updated_at FROM tenants ORDER BY created_at DESC LIMIT %s OFFSET %s`, ph.Next(), ph.Next())

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenants []*domain.Tenant
	for rows.Next() {
		var rrow tenantRow
		if err := rows.Scan(&rrow.id, &rrow.name, &rrow.description, &rrow.status, &rrow.createdAt, &rrow.updatedAt); err != nil {
			return nil, err
		}

		tenant := &domain.Tenant{
			ID:        rrow.id,
			Name:      rrow.name,
			Status:    rrow.status,
			CreatedAt: rrow.createdAt,
			UpdatedAt: rrow.updatedAt,
		}
		if rrow.description.Valid {
			tenant.Description = &rrow.description.String
		}
		tenants = append(tenants, tenant)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tenants, nil
}

type userRepository struct {
	db      *sql.DB
	dialect database.Dialect
}

type userRow struct {
	id             string
	tenantID       string
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
	query := fmt.Sprintf(`INSERT INTO users (id, tenant_id, email, hashed_password, role, status)
VALUES (%s, %s, %s, %s, %s, %s)`, ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next())

	role := user.Role
	if role == "" {
		role = "viewer"
	}
	status := user.Status
	if status == "" {
		status = "active"
	}

	_, err := r.db.ExecContext(ctx, query, user.ID, user.TenantID, user.Email, user.HashedPassword, role, status)
	return err
}

func (r *userRepository) GetByEmail(ctx context.Context, tenantID, email string) (*domain.User, error) {
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`SELECT id, tenant_id, email, hashed_password, role, status, last_login_at, created_at, updated_at
FROM users WHERE tenant_id = %s AND email = %s`, ph.Next(), ph.Next())

	var row userRow
	err := r.db.QueryRowContext(ctx, query, tenantID, email).Scan(&row.id, &row.tenantID, &row.email, &row.hashedPassword, &row.role, &row.status, &row.lastLoginAt, &row.createdAt, &row.updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	user := &domain.User{
		ID:             row.id,
		TenantID:       row.tenantID,
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

func (r *userRepository) UpdateLastLogin(ctx context.Context, tenantID, userID string) error {
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`UPDATE users SET last_login_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = %s AND tenant_id = %s`, ph.Next(), ph.Next())

	result, err := r.db.ExecContext(ctx, query, userID, tenantID)
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

type promptRepository struct {
	db      *sql.DB
	dialect database.Dialect
}

type promptRow struct {
	id              string
	tenantID        string
	name            string
	description     sql.NullString
	tags            sql.NullString
	activeVersionID sql.NullString
	createdBy       sql.NullString
	createdAt       time.Time
	updatedAt       time.Time
}

func (r *promptRepository) Create(ctx context.Context, prompt *domain.Prompt) error {
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`INSERT INTO prompts (id, tenant_id, name, description, tags, active_version_id, created_by)
VALUES (%s, %s, %s, %s, %s, %s, %s)`, ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next())

	desc := sql.NullString{}
	if prompt.Description != nil {
		desc = sql.NullString{String: *prompt.Description, Valid: true}
	}
	tags := sql.NullString{}
	if len(prompt.Tags) > 0 {
		tags = sql.NullString{String: string(prompt.Tags), Valid: true}
	}
	activeVersion := sql.NullString{}
	if prompt.ActiveVersionID != nil {
		activeVersion = sql.NullString{String: *prompt.ActiveVersionID, Valid: true}
	}
	createdBy := sql.NullString{}
	if prompt.CreatedBy != nil {
		createdBy = sql.NullString{String: *prompt.CreatedBy, Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query, prompt.ID, prompt.TenantID, prompt.Name, desc, tags, activeVersion, createdBy)
	return err
}

func (r *promptRepository) GetByID(ctx context.Context, tenantID, promptID string) (*domain.Prompt, error) {
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`SELECT id, tenant_id, name, description, tags, active_version_id, created_by, created_at, updated_at
FROM prompts WHERE tenant_id = %s AND id = %s`, ph.Next(), ph.Next())

	var row promptRow
	err := r.db.QueryRowContext(ctx, query, tenantID, promptID).Scan(&row.id, &row.tenantID, &row.name, &row.description, &row.tags, &row.activeVersionID, &row.createdBy, &row.createdAt, &row.updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	prompt := &domain.Prompt{
		ID:        row.id,
		TenantID:  row.tenantID,
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
	if row.createdBy.Valid {
		prompt.CreatedBy = &row.createdBy.String
	}
	return prompt, nil
}

func (r *promptRepository) ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*domain.Prompt, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`SELECT id, tenant_id, name, description, tags, active_version_id, created_by, created_at, updated_at
FROM prompts WHERE tenant_id = %s ORDER BY updated_at DESC LIMIT %s OFFSET %s`, ph.Next(), ph.Next(), ph.Next())

	rows, err := r.db.QueryContext(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prompts []*domain.Prompt
	for rows.Next() {
		var row promptRow
		if err := rows.Scan(&row.id, &row.tenantID, &row.name, &row.description, &row.tags, &row.activeVersionID, &row.createdBy, &row.createdAt, &row.updatedAt); err != nil {
			return nil, err
		}
		prompt := &domain.Prompt{
			ID:        row.id,
			TenantID:  row.tenantID,
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

func (r *promptRepository) UpdateActiveVersion(ctx context.Context, tenantID, promptID string, versionID *string) error {
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`UPDATE prompts SET active_version_id = %s, updated_at = CURRENT_TIMESTAMP WHERE tenant_id = %s AND id = %s`, ph.Next(), ph.Next(), ph.Next())

	activeVersion := sql.NullString{}
	if versionID != nil {
		activeVersion = sql.NullString{String: *versionID, Valid: true}
	}

	result, err := r.db.ExecContext(ctx, query, activeVersion, tenantID, promptID)
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

type promptVersionRepository struct {
	db      *sql.DB
	dialect database.Dialect
}

type promptVersionRow struct {
	id              string
	tenantID        string
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
	query := fmt.Sprintf(`INSERT INTO prompt_versions (id, tenant_id, prompt_id, version_number, body, variables_schema, status, metadata, created_by)
VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)`, ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next())

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

	_, err := r.db.ExecContext(ctx, query, version.ID, version.TenantID, version.PromptID, version.VersionNumber, version.Body, variables, status, metadata, createdBy)
	return err
}

func (r *promptVersionRepository) GetByID(ctx context.Context, tenantID, versionID string) (*domain.PromptVersion, error) {
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`SELECT id, tenant_id, prompt_id, version_number, body, variables_schema, status, metadata, created_by, created_at
FROM prompt_versions WHERE tenant_id = %s AND id = %s`, ph.Next(), ph.Next())

	var row promptVersionRow
	err := r.db.QueryRowContext(ctx, query, tenantID, versionID).Scan(&row.id, &row.tenantID, &row.promptID, &row.versionNumber, &row.body, &row.variablesSchema, &row.status, &row.metadata, &row.createdBy, &row.createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	version := &domain.PromptVersion{
		ID:            row.id,
		TenantID:      row.tenantID,
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

func (r *promptVersionRepository) ListByPrompt(ctx context.Context, tenantID, promptID string, limit, offset int) ([]*domain.PromptVersion, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`SELECT id, tenant_id, prompt_id, version_number, body, variables_schema, status, metadata, created_by, created_at
FROM prompt_versions WHERE tenant_id = %s AND prompt_id = %s ORDER BY version_number DESC LIMIT %s OFFSET %s`, ph.Next(), ph.Next(), ph.Next(), ph.Next())

	rows, err := r.db.QueryContext(ctx, query, tenantID, promptID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []*domain.PromptVersion
	for rows.Next() {
		var row promptVersionRow
		if err := rows.Scan(&row.id, &row.tenantID, &row.promptID, &row.versionNumber, &row.body, &row.variablesSchema, &row.status, &row.metadata, &row.createdBy, &row.createdAt); err != nil {
			return nil, err
		}
		version := &domain.PromptVersion{
			ID:            row.id,
			TenantID:      row.tenantID,
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

func (r *promptVersionRepository) GetLatestVersionNumber(ctx context.Context, tenantID, promptID string) (int, error) {
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`SELECT COALESCE(MAX(version_number), 0) FROM prompt_versions WHERE tenant_id = %s AND prompt_id = %s`, ph.Next(), ph.Next())

	var latest sql.NullInt64
	if err := r.db.QueryRowContext(ctx, query, tenantID, promptID).Scan(&latest); err != nil {
		return 0, err
	}
	if latest.Valid {
		return int(latest.Int64), nil
	}
	return 0, nil
}

type promptExecutionLogRepository struct {
	db      *sql.DB
	dialect database.Dialect
}

type executionLogRow struct {
	id               string
	tenantID         string
	promptID         string
	promptVersionID  string
	userID           sql.NullString
	status           string
	durationMs       sql.NullInt64
	requestPayload   sql.NullString
	responseMetadata sql.NullString
	createdAt        time.Time
}

func (r *promptExecutionLogRepository) Create(ctx context.Context, log *domain.PromptExecutionLog) error {
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`INSERT INTO prompt_execution_logs (id, tenant_id, prompt_id, prompt_version_id, user_id, status, duration_ms, request_payload, response_metadata)
VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)`, ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next(), ph.Next())

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

	_, err := r.db.ExecContext(ctx, query, log.ID, log.TenantID, log.PromptID, log.PromptVersionID, userID, log.Status, duration, request, response)
	return err
}

func (r *promptExecutionLogRepository) ListRecent(ctx context.Context, tenantID, promptID string, limit int) ([]*domain.PromptExecutionLog, error) {
	if limit <= 0 {
		limit = 20
	}
	ph := database.NewPlaceholderBuilder(r.dialect)
	query := fmt.Sprintf(`SELECT id, tenant_id, prompt_id, prompt_version_id, user_id, status, duration_ms, request_payload, response_metadata, created_at
FROM prompt_execution_logs WHERE tenant_id = %s AND prompt_id = %s ORDER BY created_at DESC LIMIT %s`, ph.Next(), ph.Next(), ph.Next())

	rows, err := r.db.QueryContext(ctx, query, tenantID, promptID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*domain.PromptExecutionLog
	for rows.Next() {
		var row executionLogRow
		if err := rows.Scan(&row.id, &row.tenantID, &row.promptID, &row.promptVersionID, &row.userID, &row.status, &row.durationMs, &row.requestPayload, &row.responseMetadata, &row.createdAt); err != nil {
			return nil, err
		}
		log := &domain.PromptExecutionLog{
			ID:              row.id,
			TenantID:        row.tenantID,
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
