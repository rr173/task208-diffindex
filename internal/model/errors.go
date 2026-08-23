// Package model 定义蛋白质晶体衍射峰索引校验服务的核心实体、状态机与错误。
package model

import (
	"errors"
	"fmt"
)

// 领域错误，供上层用 errors.Is 判定。
var (
	// ErrNotFound 目标实体不存在。
	ErrNotFound = errors.New("entity not found")
	// ErrInvalidState 状态机流转非法（当前状态不允许该动作）。
	ErrInvalidState = errors.New("invalid state transition")
	// ErrDuplicate 重复写入（幂等键冲突）。
	ErrDuplicate = errors.New("duplicate record")
	// ErrConflict 峰与当前晶格索引矛盾（冲突峰）。
	ErrConflict = errors.New("peak conflicts with lattice index")
	// ErrInvalidInput 输入参数非法（波长缺失、峰位负值、几何矛盾等）。
	ErrInvalidInput = errors.New("invalid input")
	// ErrInsufficientData 数据不足以完成操作（峰过少、无确认晶格等）。
	ErrInsufficientData = fmt.Errorf("%w: insufficient data", ErrInvalidInput)
	// ErrSealed 实体已封存，禁止修改。
	ErrSealed = errors.New("entity is sealed")
	// ErrUnsupported 尚不支持的操作。
	ErrUnsupported = errors.New("operation not supported")
)

// InvalidInputf 构造带上下文的非法输入错误。
func InvalidInputf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidInput, fmt.Sprintf(format, args...))
}
