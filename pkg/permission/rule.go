package permission

// RuleMatchConditions 定义规则匹配条件
type RuleMatchConditions struct {
	// 按路径前缀匹配（支持通配符）
	PathPrefix []string `mapstructure:"path_prefix"`
	// 按路径正则匹配
	PathRegex []string `mapstructure:"path_regex"`

	// 按 provider scheme 匹配
	Provider []string `mapstructure:"provider"`
	// 按 bucket 匹配（云存储）
	Bucket []string `mapstructure:"bucket"`

	// 按文件扩展名匹配
	Extension []string `mapstructure:"extension"`

	// 按操作类型匹配
	Operations []Permission `mapstructure:"operations"`

	// 文件大小范围（字节）
	MinSize int64 `mapstructure:"min_size"`
	MaxSize int64 `mapstructure:"max_size"`
}

// Rule 定义一条权限规则
type Rule struct {
	// 规则名称（唯一标识）
	Name string `mapstructure:"name"`
	// 规则描述
	Description string `mapstructure:"description"`

	// 匹配条件
	Match RuleMatchConditions `mapstructure:"match"`

	// 权限效果
	Effect Effect `mapstructure:"effect"`

	// 优先级（数值越小优先级越高，优先匹配）
	Priority int `mapstructure:"priority"`

	// 是否启用
	Enabled bool `mapstructure:"enabled"`

	// 引用的角色名称（可选）
	Role *string `mapstructure:"role"`
}

// Policy 定义一组规则及其默认行为
type Policy struct {
	// 策略名称
	Name string `mapstructure:"name"`
	// 默认效果（当没有规则匹配时）
	DefaultEffect Effect `mapstructure:"default_effect"`

	// 规则列表
	Rules []Rule `mapstructure:"rules"`

	// 是否启用
	Enabled bool `mapstructure:"enabled"`
}

// MatchOperation 检查规则是否匹配指定操作
func (r *Rule) MatchOperation(op Permission) bool {
	if len(r.Match.Operations) == 0 {
		return true // 未指定操作则匹配所有
	}
	for _, o := range r.Match.Operations {
		if o == op {
			return true
		}
	}
	return false
}

// HasRoleReference 检查规则是否引用了角色
func (r *Rule) HasRoleReference() bool {
	return r.Role != nil && *r.Role != ""
}