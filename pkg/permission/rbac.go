package permission

import "context"

// RBAC 角色权限评估器
type RBAC struct {
	roles map[string]*Role
}

// NewRBAC 创建新的 RBAC 评估器
func NewRBAC() *RBAC {
	return &RBAC{
		roles: make(map[string]*Role),
	}
}

// Check 检查角色是否具有指定权限
func (r *RBAC) Check(ctx context.Context, req Request) Result {
	// 优先检查请求中声明的角色
	for _, roleName := range req.RequestedRoles {
		role, ok := r.roles[roleName]
		if !ok {
			continue
		}
		if role.HasPermissionWithContext(req.Operation, req.TargetPath, req.Provider, req.Bucket) {
			return Result{
				Allowed: true,
				Reason:  "allowed by role: " + role.Name,
			}
		}
	}

	// 未找到匹配角色，返回拒绝
	return Result{
		Allowed: false,
		Reason:  "no matching role found",
	}
}

// LoadRoles 加载角色列表
func (r *RBAC) LoadRoles(roles []Role) error {
	for i := range roles {
		r.roles[roles[i].Name] = &roles[i]
	}
	return nil
}

// AddRole 添加角色
func (r *RBAC) AddRole(role Role) error {
	r.roles[role.Name] = &role
	return nil
}

// RemoveRole 移除角色
func (r *RBAC) RemoveRole(name string) bool {
	if _, ok := r.roles[name]; ok {
		delete(r.roles, name)
		return true
	}
	return false
}

// GetRole 获取角色
func (r *RBAC) GetRole(name string) (*Role, bool) {
	role, ok := r.roles[name]
	return role, ok
}

// ListRoles 列出所有角色
func (r *RBAC) ListRoles() []*Role {
	roles := make([]*Role, 0, len(r.roles))
	for _, role := range r.roles {
		roles = append(roles, role)
	}
	return roles
}

// GetAllRoles 获取所有角色映射
func (r *RBAC) GetAllRoles() map[string]*Role {
	result := make(map[string]*Role)
	for k, v := range r.roles {
		result[k] = v
	}
	return result
}

// LoadFromMap 从映射加载角色
func (r *RBAC) LoadFromMap(roles map[string]*Role) error {
	for name, role := range roles {
		r.roles[name] = role
	}
	return nil
}

// IsEmpty 检查是否没有角色
func (r *RBAC) IsEmpty() bool {
	return len(r.roles) == 0
}