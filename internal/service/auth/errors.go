package auth

import "errors"

var (
	// ErrTenantRequired 表示租户信息缺失。
	ErrTenantRequired = errors.New("tenant required")
	// ErrInvalidInput 表示注册时输入不完整。
	ErrInvalidInput = errors.New("invalid input")
	// ErrTenantNotFound 找不到指定租户。
	ErrTenantNotFound = errors.New("tenant not found")
	// ErrUserExists 表示邮箱已存在。
	ErrUserExists = errors.New("user already exists")
	// ErrInvalidCredentials 登录凭证错误。
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrUserDisabled 用户被禁用或未激活。
	ErrUserDisabled = errors.New("user disabled")
	// ErrTokenInvalid 刷新令牌无效。
	ErrTokenInvalid = errors.New("token invalid")
)
