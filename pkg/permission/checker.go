package permission

import "context"

// PermissionChecker 权限检查器接口
type PermissionChecker interface {
	// Check 检查操作是否允许
	Check(ctx context.Context, req Request) Result

	// LoadPolicies 从配置加载策略
	LoadPolicies(policies []Policy) error

	// AddPolicy 添加策略
	AddPolicy(policy Policy) error

	// RemovePolicy 移除策略
	RemovePolicy(name string) error
}

// RoleManager 角色管理器接口
type RoleManager interface {
	// GetRole 根据名称获取角色
	GetRole(name string) (*Role, bool)

	// AddRole 添加角色
	AddRole(role Role) error

	// RemoveRole 移除角色
	RemoveRole(name string) bool

	// ListRoles 列出所有角色
	ListRoles() []*Role
}

// PermissionEvaluator 权限评估器接口（组合检查器和角色管理器）
type PermissionEvaluator interface {
	PermissionChecker
	RoleManager
}