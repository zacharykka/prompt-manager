package domain

import "errors"

var (
	// ErrNotFound 表示仓储查询结果为空。
	ErrNotFound = errors.New("domain: not found")
)
