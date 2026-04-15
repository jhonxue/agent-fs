package permission

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// UserRoleMapping 用户到角色的映射
type UserRoleMapping struct {
	// 用户名
	User string `mapstructure:"user" yaml:"user"`
	// 角色名称列表
	Roles []string `mapstructure:"roles" yaml:"roles"`
}

// Config 权限配置
type Config struct {
	// 是否启用权限控制
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`
	// 配置文件路径
	ConfigPath string `mapstructure:"config_path" yaml:"config_path"`
	// 严格模式（无匹配规则时拒绝）
	StrictMode bool `mapstructure:"strict_mode" yaml:"strict_mode"`
	// 是否启用索引驱动的评估路径（灰度开关）
	EnableRuleIndexExecution bool `mapstructure:"enable_rule_index_execution" yaml:"enable_rule_index_execution"`

	// 角色定义（YAML 中为 roles 节点）
	Roles []Role `mapstructure:"roles" yaml:"roles"`

	// 用户角色映射（YAML 中为 user_role_mappings 节点）
	UserRoleMappings []UserRoleMapping `mapstructure:"user_role_mappings" yaml:"user_role_mappings"`

	// 策略定义（YAML 中为 policies 节点）
	Policies []Policy `mapstructure:"policies" yaml:"policies"`
}

// LoadConfig 从文件加载配置
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return LoadConfigFromBytes(data)
}

// LoadConfigFromBytes 从字节数组加载配置
func LoadConfigFromBytes(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// LoadConfigFromMap 从映射加载配置（用于 Viper）
func LoadConfigFromMap(m map[string]interface{}) (*Config, error) {
	data, err := yaml.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	return LoadConfigFromBytes(data)
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证角色定义
	roleNames := make(map[string]bool)
	for _, role := range c.Roles {
		if role.Name == "" {
			return fmt.Errorf("role name is required")
		}
		if roleNames[role.Name] {
			return fmt.Errorf("duplicate role name: %s", role.Name)
		}
		roleNames[role.Name] = true

		// 如果是预定义类型，验证类型有效性
		if role.Type != RoleTypeCustom {
			if !IsValidRoleType(role.Type) {
				return fmt.Errorf("invalid role type: %s", role.Type)
			}
		}
	}

	// 验证用户角色映射中引用的角色是否存在
	for _, mapping := range c.UserRoleMappings {
		if mapping.User == "" {
			return fmt.Errorf("user role mapping: user is required")
		}
		for _, roleName := range mapping.Roles {
			if !roleNames[roleName] {
				return fmt.Errorf("user role mapping for %s references undefined role: %s", mapping.User, roleName)
			}
		}
	}

	// 验证规则名全局唯一性（包含空名检查）
	globalRuleNames := make(map[string]bool)
	for _, policy := range c.Policies {
		for _, rule := range policy.Rules {
			if rule.Name == "" {
				return fmt.Errorf("rule name is required (policy=%s)", policy.Name)
			}
			if globalRuleNames[rule.Name] {
				return fmt.Errorf("duplicate rule name: %s (policy=%s)", rule.Name, policy.Name)
			}
			globalRuleNames[rule.Name] = true
		}
	}

	// 验证策略中的规则引用
	roleRefChecker := make(map[string]bool)
	for _, role := range c.Roles {
		roleRefChecker[role.Name] = true
	}

	for _, policy := range c.Policies {
		if policy.Name == "" {
			return fmt.Errorf("policy name is required")
		}
		for _, rule := range policy.Rules {
			if rule.HasRoleReference() {
				roleName := *rule.Role
				if !roleRefChecker[roleName] {
					return fmt.Errorf("rule %s references undefined role: %s", rule.Name, roleName)
				}
			}
		}
	}

	return nil
}

// GetEnabledRoles 获取启用的角色列表
func (c *Config) GetEnabledRoles() []Role {
	// 目前所有角色都是启用的
	return c.Roles
}

// GetEnabledPolicies 获取启用的策略列表
func (c *Config) GetEnabledPolicies() []Policy {
	var policies []Policy
	for _, p := range c.Policies {
		if p.Enabled {
			policies = append(policies, p)
		}
	}
	return policies
}

// CreateDefaultConfig 创建默认配置
func CreateDefaultConfig() *Config {
	return &Config{
		Enabled:                  false, // 默认禁用，需要显式启用
		StrictMode:               true,
		EnableRuleIndexExecution: false,
		Roles:                    getDefaultRoles(),
		Policies:                 DefaultPolicies,
	}
}

// getDefaultRoles 获取默认角色列表
func getDefaultRoles() []Role {
	return []Role{
		{
			Name:        "admin",
			Type:        RoleTypeAdmin,
			Description: "管理员：拥有所有操作权限",
			Permissions: []Permission{
				PermissionRead,
				PermissionWrite,
				PermissionOverwrite,
				PermissionDelete,
				PermissionCopy,
				PermissionMove,
			},
		},
		{
			Name:        "editor",
			Type:        RoleTypeEditor,
			Description: "编辑：读写权限",
			Permissions: []Permission{
				PermissionRead,
				PermissionWrite,
				PermissionOverwrite,
			},
		},
		{
			Name:        "reader",
			Type:        RoleTypeReader,
			Description: "读取者：只读权限",
			Permissions: []Permission{
				PermissionRead,
			},
		},
		{
			Name:        "backup-operator",
			Type:        RoleTypeBackup,
			Description: "备份操作员：读取和复制权限",
			Permissions: []Permission{
				PermissionRead,
				PermissionCopy,
			},
		},
	}
}

// DefaultPermissionConfigPath 返回默认配置文件路径
func DefaultPermissionConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".afs-permission.yaml"
	}
	return home + "/.afs-permission.yaml"
}

// ConfigExists 检查默认配置文件是否存在
func ConfigExists() bool {
	path := DefaultPermissionConfigPath()
	_, err := os.Stat(path)
	return err == nil
}