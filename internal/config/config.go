package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	mapstructure "github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

const (
	defaultEnv        = "development"
	envKey            = "PROMPT_MANAGER_ENV"
	envPrefix         = "PROMPT_MANAGER"
	defaultConfigName = "default"
	configType        = "yaml"
)

// Config 聚合应用所需的全部配置项。
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Logging  LoggingConfig  `mapstructure:"logging"`
}

// AppConfig 描述应用级别的元信息。
type AppConfig struct {
	Name string `mapstructure:"name"`
	Env  string `mapstructure:"env"`
}

// ServerConfig 负责 HTTP 服务相关配置。
type ServerConfig struct {
	Host            string                `mapstructure:"host"`
	Port            int                   `mapstructure:"port"`
	ReadTimeout     time.Duration         `mapstructure:"readTimeout"`
	WriteTimeout    time.Duration         `mapstructure:"writeTimeout"`
	ShutdownTimeout time.Duration         `mapstructure:"shutdownTimeout"`
	MaxRequestBody  int64                 `mapstructure:"maxRequestBody"`
	CORS            CORSConfig            `mapstructure:"cors"`
	SecurityHeaders SecurityHeadersConfig `mapstructure:"securityHeaders"`
}

// CORSConfig 控制跨域访问白名单及相关选项。
type CORSConfig struct {
	AllowOrigins     []string `mapstructure:"allowOrigins"`
	AllowCredentials bool     `mapstructure:"allowCredentials"`
}

// SecurityHeadersConfig 控制通用安全响应头的行为。
type SecurityHeadersConfig struct {
	FrameOptions              string `mapstructure:"frameOptions"`
	ContentTypeNosniff        bool   `mapstructure:"contentTypeNosniff"`
	ReferrerPolicy            string `mapstructure:"referrerPolicy"`
	XSSProtection             string `mapstructure:"xssProtection"`
	ContentSecurityPolicy     string `mapstructure:"contentSecurityPolicy"`
	CrossOriginOpenerPolicy   string `mapstructure:"crossOriginOpenerPolicy"`
	CrossOriginEmbedderPolicy string `mapstructure:"crossOriginEmbedderPolicy"`
	CrossOriginResourcePolicy string `mapstructure:"crossOriginResourcePolicy"`
}

// DatabaseConfig 定义数据库连接选项，兼容 SQLite 与 PostgreSQL。
type DatabaseConfig struct {
	Driver          string        `mapstructure:"driver"`
	DSN             string        `mapstructure:"dsn"`
	MaxOpen         int           `mapstructure:"maxOpen"`
	MaxIdle         int           `mapstructure:"maxIdle"`
	ConnMaxLifetime time.Duration `mapstructure:"connMaxLifetime"`
}

// RedisConfig 描述 Redis 客户端所需的连接参数。
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"poolSize"`
}

// AuthConfig 管理 JWT 与 API Key 等认证参数。
type AuthConfig struct {
	AccessTokenSecret  string        `mapstructure:"accessTokenSecret"`
	RefreshTokenSecret string        `mapstructure:"refreshTokenSecret"`
	AccessTokenTTL     time.Duration `mapstructure:"accessTokenTTL"`
	RefreshTokenTTL    time.Duration `mapstructure:"refreshTokenTTL"`
	APIKeyHashSecret   string        `mapstructure:"apiKeyHashSecret"`
}

// LoggingConfig 控制日志输出级别等行为。
type LoggingConfig struct {
	Level string `mapstructure:"level"`
}

// Load 从给定路径加载配置；若 env 为空会自动读取环境变量或回退到默认值。
func Load(configDir string, env string) (*Config, error) {
	chosenEnv := determineEnv(env)

	v := viper.New()
	v.SetConfigType(configType)
	v.SetConfigName(defaultConfigName)
	v.AddConfigPath(configDir)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read base config: %w", err)
	}

	if chosenEnv != defaultConfigName {
		envConfig := viper.New()
		envConfig.SetConfigType(configType)
		envConfig.SetConfigName(chosenEnv)
		envConfig.AddConfigPath(configDir)

		if err := envConfig.ReadInConfig(); err == nil {
			if err := v.MergeConfigMap(envConfig.AllSettings()); err != nil {
				return nil, fmt.Errorf("merge %s config: %w", chosenEnv, err)
			}
		}
	}

	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg, func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "mapstructure"
		dc.DecodeHook = mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		)
	}); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	applyDefaults(&cfg, chosenEnv)

	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// determineEnv 统一处理环境变量回退逻辑。
func determineEnv(env string) string {
	if env != "" {
		return env
	}
	if fromEnv := os.Getenv(envKey); fromEnv != "" {
		return fromEnv
	}
	return defaultEnv
}

// applyDefaults 补齐缺失字段，避免配置不完整导致的崩溃。
func applyDefaults(cfg *Config, env string) {
	if cfg.App.Name == "" {
		cfg.App.Name = "prompt-manager"
	}
	if cfg.App.Env == "" {
		cfg.App.Env = env
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 10 * time.Second
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 10 * time.Second
	}
	if cfg.Server.ShutdownTimeout == 0 {
		cfg.Server.ShutdownTimeout = 10 * time.Second
	}
	if cfg.Server.MaxRequestBody <= 0 {
		cfg.Server.MaxRequestBody = 3 * 1024 * 1024
	}
	if len(cfg.Server.CORS.AllowOrigins) == 0 {
		cfg.Server.CORS.AllowOrigins = []string{"*"}
	}
	if cfg.Server.SecurityHeaders.FrameOptions == "" {
		cfg.Server.SecurityHeaders.FrameOptions = "DENY"
	}
	if cfg.Server.SecurityHeaders.ContentTypeNosniff == false {
		cfg.Server.SecurityHeaders.ContentTypeNosniff = true
	}
	if cfg.Server.SecurityHeaders.ReferrerPolicy == "" {
		cfg.Server.SecurityHeaders.ReferrerPolicy = "no-referrer"
	}
	if cfg.Server.SecurityHeaders.XSSProtection == "" {
		cfg.Server.SecurityHeaders.XSSProtection = "0"
	}
	if cfg.Server.SecurityHeaders.CrossOriginOpenerPolicy == "" {
		cfg.Server.SecurityHeaders.CrossOriginOpenerPolicy = "same-origin"
	}
	if cfg.Server.SecurityHeaders.CrossOriginResourcePolicy == "" {
		cfg.Server.SecurityHeaders.CrossOriginResourcePolicy = "same-site"
	}
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "sqlite"
	}
	if cfg.Database.DSN == "" {
		cfg.Database.DSN = filepath.ToSlash("file:./data/dev.db?cache=shared&_fk=1")
	}
	if cfg.Database.MaxOpen == 0 {
		cfg.Database.MaxOpen = 10
	}
	if cfg.Database.MaxIdle == 0 {
		cfg.Database.MaxIdle = 5
	}
	if cfg.Database.ConnMaxLifetime == 0 {
		cfg.Database.ConnMaxLifetime = 5 * time.Minute
	}
	if cfg.Redis.PoolSize == 0 {
		cfg.Redis.PoolSize = 10
	}
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
}

func validateConfig(cfg *Config) error {
	if err := validateSecret("auth.accessTokenSecret", cfg.Auth.AccessTokenSecret); err != nil {
		return err
	}
	if err := validateSecret("auth.refreshTokenSecret", cfg.Auth.RefreshTokenSecret); err != nil {
		return err
	}
	if err := validateSecret("auth.apiKeyHashSecret", cfg.Auth.APIKeyHashSecret); err != nil {
		return err
	}
	if err := validateCORSConfig(cfg.Server.CORS, cfg.App.Env); err != nil {
		return err
	}
	if err := validateSecurityHeaders(cfg.Server.SecurityHeaders); err != nil {
		return err
	}
	return nil
}

func validateSecret(field, secret string) error {
	clean := strings.TrimSpace(secret)
	if len(clean) < 32 {
		return fmt.Errorf("config %s must be at least 32 characters", field)
	}
	if strings.Contains(strings.ToLower(clean), "change-me") {
		return fmt.Errorf("config %s must not use default placeholder", field)
	}
	return nil
}

func validateCORSConfig(corsCfg CORSConfig, env string) error {
	for _, origin := range corsCfg.AllowOrigins {
		clean := strings.TrimSpace(origin)
		if clean == "" {
			return fmt.Errorf("config server.cors.allowOrigins must not contain empty entries")
		}
		if env == "production" && clean == "*" {
			return fmt.Errorf("config server.cors.allowOrigins must not use wildcard '*' in production")
		}
	}
	return nil
}

func validateSecurityHeaders(secCfg SecurityHeadersConfig) error {
	frame := strings.TrimSpace(strings.ToUpper(secCfg.FrameOptions))
	if frame != "" && frame != "DENY" && frame != "SAMEORIGIN" {
		return fmt.Errorf("config server.securityHeaders.frameOptions must be DENY or SAMEORIGIN when set")
	}
	return nil
}

// Addr 返回 HTTP 服务监听地址。
func (s ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}
