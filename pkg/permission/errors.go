package permission

import "errors"

// 权限相关错误

var (
	// ErrPermissionDenied 权限被拒绝
	ErrPermissionDenied = errors.New("permission denied")

	// ErrRoleNotFound 角色未找到
	ErrRoleNotFound = errors.New("role not found")

	// ErrPolicyNotFound 策略未找到
	ErrPolicyNotFound = errors.New("policy not found")

	// ErrRuleNotFound 规则未找到
	ErrRuleNotFound = errors.New("rule not found")

	// ErrInvalidConfig 无效配置
	ErrInvalidConfig = errors.New("invalid permission config")

	// ErrRoleAlreadyExists 角色已存在
	ErrRoleAlreadyExists = errors.New("role already exists")

	// ErrPolicyAlreadyExists 策略已存在
	ErrPolicyAlreadyExists = errors.New("policy already exists")
)

// PermissionError 权限错误
type PermissionError struct {
	Code    string
	Message string
	Cause   error
}

func (e *PermissionError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "permission error"
}

func (e *PermissionError) Unwrap() error {
	return e.Cause
}

// NewPermissionError 创建权限错误
func NewPermissionError(code, message string) error {
	return &PermissionError{
		Code:    code,
		Message: message,
	}
}

// WrapPermissionError 包装权限错误
func WrapPermissionError(code, message string, cause error) error {
	return &PermissionError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// ErrorCode 错误码常量
const (
	ErrCodePermissionDenied = "ERR_PERMISSION_DENIED"
	ErrCodeRoleNotFound     = "ERR_ROLE_NOT_FOUND"
	ErrCodePolicyNotFound  = "ERR_POLICY_NOT_FOUND"
	ErrCodeInvalidConfig   = "ERR_INVALID_CONFIG"
	ErrCodeRoleDuplicate   = "ERR_ROLE_DUPLICATE"
)