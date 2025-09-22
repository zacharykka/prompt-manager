package database

import "fmt"

// Dialect 用于适配不同数据库的占位符风格。
type Dialect struct {
	driver string
}

// NewDialect 根据驱动名称构建方言。
func NewDialect(driver string) Dialect {
	return Dialect{driver: driver}
}

// Placeholder 返回指定序号的占位符。
func (d Dialect) Placeholder(index int) string {
	switch d.driver {
	case "postgres", "pgx", "postgresql":
		return fmt.Sprintf("$%d", index)
	default:
		return "?"
	}
}

// PlaceholderBuilder 用于生成顺序占位符，避免手动维护计数。
type PlaceholderBuilder struct {
	dialect Dialect
	index   int
}

// NewPlaceholderBuilder 创建一个计数器实例。
func NewPlaceholderBuilder(d Dialect) *PlaceholderBuilder {
	return &PlaceholderBuilder{dialect: d}
}

// Next 返回下一个可用占位符。
func (b *PlaceholderBuilder) Next() string {
	b.index++
	return b.dialect.Placeholder(b.index)
}
