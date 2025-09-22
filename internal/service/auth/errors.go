package auth

import "errors"

var (
	// ErrInvalidInput 表示注册时输入不完整。
	ErrInvalidInput = errors.New("invalid input")
	// ErrUserExists 表示邮箱已存在。
	ErrUserExists = errors.New("user already exists")
	// ErrInvalidCredentials 登录凭证错误。
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrUserDisabled 用户被禁用或未激活。
	ErrUserDisabled = errors.New("user disabled")
	// ErrTokenInvalid 刷新令牌无效。
	ErrTokenInvalid = errors.New("token invalid")
)
