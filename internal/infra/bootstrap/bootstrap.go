package bootstrap

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/zacharykka/prompt-manager/internal/config"
	domain "github.com/zacharykka/prompt-manager/internal/domain"
	authutil "github.com/zacharykka/prompt-manager/pkg/auth"
	"go.uber.org/zap"
)

// EnsureDefaultAdmin 创建默认租户与管理员账号（若不存在）。
func EnsureDefaultAdmin(ctx context.Context, repos *domain.Repositories, cfg config.BootstrapConfig, logger *zap.Logger) error {
	if !cfg.Enabled {
		logger.Info("bootstrap skipped (disabled)")
		return nil
	}

	tenantID := strings.TrimSpace(cfg.TenantID)
	if tenantID == "" {
		tenantID = "default-tenant"
	}

	tenantName := cfg.TenantName
	if tenantName == "" {
		tenantName = "Default Tenant"
	}

	if _, err := repos.Tenants.GetByID(ctx, tenantID); err != nil {
		if err == domain.ErrNotFound {
			tenant := &domain.Tenant{
				ID:          tenantID,
				Name:        tenantName,
				Description: optionalString(cfg.TenantDescription),
				Status:      "active",
			}
			if err := repos.Tenants.Create(ctx, tenant); err != nil {
				return err
			}
			logger.Info("bootstrap tenant created", zap.String("tenant_id", tenantID))
		} else {
			return err
		}
	}

	adminEmail := strings.TrimSpace(strings.ToLower(cfg.AdminEmail))
	if adminEmail == "" {
		adminEmail = "admin"
	}

	if _, err := repos.Users.GetByEmail(ctx, tenantID, adminEmail); err == nil {
		logger.Info("bootstrap admin exists", zap.String("tenant_id", tenantID), zap.String("email", adminEmail))
		return nil
	} else if err != domain.ErrNotFound {
		return err
	}

	return ensureAdmin(ctx, repos, cfg, tenantID, adminEmail, logger)
}

func optionalString(val string) *string {
	trimmed := strings.TrimSpace(val)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizedRole(role string) string {
	value := strings.TrimSpace(strings.ToLower(role))
	switch value {
	case "admin", "editor", "viewer":
		return value
	default:
		return "admin"
	}
}

func ensureAdmin(ctx context.Context, repos *domain.Repositories, cfg config.BootstrapConfig, tenantID, adminEmail string, logger *zap.Logger) error {
	hash, err := authutil.HashPassword(cfg.AdminPassword)
	if err != nil {
		return err
	}

	admin := &domain.User{
		ID:             uuid.NewString(),
		TenantID:       tenantID,
		Email:          adminEmail,
		HashedPassword: hash,
		Role:           normalizedRole(cfg.AdminRole),
		Status:         "active",
	}

	if err := repos.Users.Create(ctx, admin); err != nil {
		return err
	}

	logger.Info("bootstrap admin created", zap.String("tenant_id", tenantID), zap.String("email", adminEmail))
	return nil
}
