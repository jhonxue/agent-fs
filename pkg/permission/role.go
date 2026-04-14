package permission

import (
	"path/filepath"
	"sync"
)

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

	// 缓存初始化同步锁（确保线程安全的延迟初始化）
	initOnce sync.Once
	// 缓存读写锁（保护 permSet、providerSet、bucketSet 的并发访问）
	cacheMu sync.RWMutex

	// 权限集合缓存（避免每次检查时遍历）
	permSet map[Permission]struct{}
	// Provider 集合缓存
	providerSet map[string]struct{}
	// Bucket 集合缓存
	bucketSet map[string]struct{}
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

// ensurePermSet 确保缓存已初始化（使用 sync.Once 确保线程安全）
func (r *Role) ensurePermSet() {
	r.initOnce.Do(func() {
		r.permSet = make(map[Permission]struct{}, len(r.Permissions))
		for _, p := range r.Permissions {
			r.permSet[p] = struct{}{}
		}

		// 初始化约束条件缓存
		if r.Conditions != nil {
			if len(r.Conditions.Provider) > 0 {
				r.providerSet = make(map[string]struct{}, len(r.Conditions.Provider))
				for _, p := range r.Conditions.Provider {
					r.providerSet[p] = struct{}{}
				}
			}
			if len(r.Conditions.Bucket) > 0 {
				r.bucketSet = make(map[string]struct{}, len(r.Conditions.Bucket))
				for _, b := range r.Conditions.Bucket {
					r.bucketSet[b] = struct{}{}
				}
			}
		}
	})
}

// HasPermission 检查角色是否具有指定权限
func (r *Role) HasPermission(p Permission) bool {
	r.ensurePermSet()
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()
	_, ok := r.permSet[p]
	return ok
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
		// 规范化路径，防止路径穿越攻击
		normalizedPath := filepath.Clean(path)
		matched := false
		for _, prefix := range r.Conditions.PathPrefix {
			normalizedPrefix := filepath.Clean(prefix)
			if len(normalizedPath) >= len(normalizedPrefix) && normalizedPath[:len(normalizedPrefix)] == normalizedPrefix {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 检查 Provider 约束（使用缓存的 map 将 O(n) 优化为 O(1)）
	if len(r.Conditions.Provider) > 0 {
		r.cacheMu.RLock()
		if _, ok := r.providerSet[provider]; !ok {
			r.cacheMu.RUnlock()
			return false
		}
		r.cacheMu.RUnlock()
	}

	// 检查 Bucket 约束（使用缓存的 map 将 O(n) 优化为 O(1)）
	if len(r.Conditions.Bucket) > 0 && bucket != "" {
		r.cacheMu.RLock()
		if _, ok := r.bucketSet[bucket]; !ok {
			r.cacheMu.RUnlock()
			return false
		}
		r.cacheMu.RUnlock()
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