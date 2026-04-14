package permission

// Permission 表示权限操作类型
type Permission string

const (
	PermissionRead      Permission = "read"       // 读取文件
	PermissionWrite     Permission = "write"      // 写入新文件
	PermissionOverwrite Permission = "overwrite"  // 覆盖已有文件
	PermissionDelete    Permission = "delete"     // 删除文件或目录
	PermissionCopy      Permission = "copy"       // 复制操作
	PermissionMove     Permission = "move"       // 移动/重命名操作
)

// AllPermissions 所有权限列表
var AllPermissions = []Permission{
	PermissionRead,
	PermissionWrite,
	PermissionOverwrite,
	PermissionDelete,
	PermissionCopy,
	PermissionMove,
}

// Effect 表示规则匹配后的效果
type Effect string

const (
	EffectAllow Effect = "allow" // 允许操作
	EffectDeny  Effect = "deny"  // 拒绝操作
)

// RoleType 表示预定义角色类型
type RoleType string

const (
	RoleTypeAdmin   RoleType = "admin"   // 管理员：所有权限
	RoleTypeEditor RoleType = "editor"   // 编辑：读写权限
	RoleTypeReader RoleType = "reader"   // 读取者：只读
	RoleTypeBackup RoleType = "backup"   // 备份员：读取+复制
	RoleTypeCustom RoleType = "custom"   // 自定义角色
)

// RoleTypePermissions 定义预定义角色的默认权限
var RoleTypePermissions = map[RoleType][]Permission{
	RoleTypeAdmin: {
		PermissionRead,
		PermissionWrite,
		PermissionOverwrite,
		PermissionDelete,
		PermissionCopy,
		PermissionMove,
	},
	RoleTypeEditor: {
		PermissionRead,
		PermissionWrite,
		PermissionOverwrite,
	},
	RoleTypeReader: {
		PermissionRead,
	},
	RoleTypeBackup: {
		PermissionRead,
		PermissionCopy,
	},
}

// IsValidPermission 检查权限是否有效
func IsValidPermission(p Permission) bool {
	for _, ap := range AllPermissions {
		if ap == p {
			return true
		}
	}
	return false
}

// IsValidRoleType 检查角色类型是否有效
func IsValidRoleType(rt RoleType) bool {
	_, ok := RoleTypePermissions[rt]
	return ok
}