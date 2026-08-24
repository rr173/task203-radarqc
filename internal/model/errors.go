package model

import "fmt"

// 错误码分类：HTTP 层依据错误码映射为 400/404/409/422。
const (
	ErrCodeInvalidInput   = "invalid_input"   // 参数或数据不合法（400）
	ErrCodeNotFound       = "not_found"       // 资源不存在（404）
	ErrCodeConflict       = "conflict"        // 状态冲突（409）
	ErrCodeForbidden      = "forbidden"       // 封存/冻结资源拒绝修改（409）
	ErrCodeUnavailable    = "unavailable"     // 资源不可用（409）
)

// AppError 统一业务错误。
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewInvalidInput 构造输入非法错误。
func NewInvalidInput(format string, args ...any) *AppError {
	return &AppError{Code: ErrCodeInvalidInput, Message: fmt.Sprintf(format, args...)}
}

// NewNotFound 构造资源不存在错误。
func NewNotFound(format string, args ...any) *AppError {
	return &AppError{Code: ErrCodeNotFound, Message: fmt.Sprintf(format, args...)}
}

// NewConflict 构造状态冲突错误。
func NewConflict(format string, args ...any) *AppError {
	return &AppError{Code: ErrCodeConflict, Message: fmt.Sprintf(format, args...)}
}

// NewForbidden 构造冻结资源拒绝修改错误。
func NewForbidden(format string, args ...any) *AppError {
	return &AppError{Code: ErrCodeForbidden, Message: fmt.Sprintf(format, args...)}
}

// NewUnavailable 构造资源不可用错误。
func NewUnavailable(format string, args ...any) *AppError {
	return &AppError{Code: ErrCodeUnavailable, Message: fmt.Sprintf(format, args...)}
}

// AsAppError 将 error 归一为 *AppError；非业务错误保留原文。
func AsAppError(err error) *AppError {
	if err == nil {
		return nil
	}
	if ae, ok := err.(*AppError); ok {
		return ae
	}
	return &AppError{Code: ErrCodeInvalidInput, Message: err.Error()}
}
