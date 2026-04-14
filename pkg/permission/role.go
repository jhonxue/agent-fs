package permission

// RoleConditions 定义角色的约束条件
type RoleConditions struct {
	// 路径前缀约束
	PathPrefix []string `mapstructure:"path_prefix"`
	// Provider 约束
	Provider []string `mapstructure:"provider"`
	// Bucket 约束
	Bucket []string `mapstructure:"bucket"`
}

// Role 定义一组权限
type Role struct {
	// 角色名称（唯一标识）
	Name string `mapstructure:"name"`
	// 角色类型
	Type RoleType `mapstructure:"type"`
	// 角色描述
	Description string `mapstructure:"description"`
	// 权限列表
	Permissions []Permission `mapstructure:"permissions"`

	// 约束条件（可选）
	Conditions *RoleConditions `mapstructure:"conditions"`
}

// RoleBinding 将角色绑定到上下文
type RoleBinding struct {
	// 绑定名称
	Name string `mapstructure:"name"`
	// 角色名称列表
	Roles []string `mapstructure:"roles"`
	// 绑定主体（保留，未来支持用户）
	Subjects []string `mapstructure:"subjects"`
}

// NewRoleFromType 根据角色类型创建预定义角色
func NewRoleFromType(name string, roleType RoleType) *Role {
	permissions, ok := RoleTypePermissions[roleType]
	if !ok {
		return nil
	}
	return &Role{
		Name:        name,
		Type:        roleType,
		Description: getRoleDescription(roleType),
		Permissions: permissions,
	}
}

// HasPermission 检查角色是否具有指定权限
func (r *Role) HasPermission(p Permission) bool {
	for _, perm := range r.Permissions {
		if perm == p {
			return true
		}
	}
	return false
}

// HasPermissionWithContext 检查角色是否具有指定权限，并考虑约束条件
func (r *Role) HasPermissionWithContext(p Permission, path, provider, bucket string) bool {
	if !r.HasPermission(p) {
		return false
	}

	// 如果没有约束条件，则完全允许
	if r.Conditions == nil {
		return true
	}

	// 检查路径约束
	if len(r.Conditions.PathPrefix) > 0 {
		matched := false
		for _, prefix := range r.Conditions.PathPrefix {
			if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 检查 Provider 约束
	if len(r.Conditions.Provider) > 0 {
		matched := false
		for _, p := range r.Conditions.Provider {
			if p == provider {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 检查 Bucket 约束
	if len(r.Conditions.Bucket) > 0 && bucket != "" {
		matched := false
		for _, b := range r.Conditions.Bucket {
			if b == bucket {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// getRoleDescription 返回角色类型的描述
func getRoleDescription(rt RoleType) string {
	switch rt {
	case RoleTypeAdmin:
		return "管理员：拥有所有操作权限"
	case RoleTypeEditor:
		return "编辑：读写权限"
	case RoleTypeReader:
		return "读取者：只读权限"
	case RoleTypeBackup:
		return "备份员：读取和复制权限"
	case RoleTypeCustom:
		return "自定义角色"
	default:
		return ""
	}
}