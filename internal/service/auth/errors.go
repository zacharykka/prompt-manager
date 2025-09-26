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
	// ErrOAuthDisabled 未开启指定 OAuth 流程。
	ErrOAuthDisabled = errors.New("oauth disabled")
	// ErrOAuthStateInvalid OAuth state 校验失败。
	ErrOAuthStateInvalid = errors.New("oauth state invalid")
	// ErrOAuthExchangeFailed OAuth 交换 access token 失败。
	ErrOAuthExchangeFailed = errors.New("oauth exchange failed")
	// ErrOAuthEmailMissing 无法获取有效的邮箱信息。
	ErrOAuthEmailMissing = errors.New("oauth email missing")
	// ErrOAuthOrgUnauthorized 用户不属于允许的组织。
	ErrOAuthOrgUnauthorized = errors.New("oauth organization not allowed")
)
