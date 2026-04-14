package permission

import (
	"context"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// PathNormalizer 路径规范化函数类型
type PathNormalizer func(path string) string

var (
	// DefaultPathNormalizer 默认路径规范化函数
	// 使用 filepath.Clean 清理路径并转换为绝对路径（如果可解析）
	DefaultPathNormalizer PathNormalizer = func(path string) string {
		cleaned := filepath.Clean(path)
		if abs, err := filepath.Abs(cleaned); err == nil {
			return abs
		}
		return cleaned
	}

	// 路径规范化器实例
	pathNormalizer PathNormalizer = DefaultPathNormalizer
)

// SetPathNormalizer 设置全局路径规范化器
func SetPathNormalizer(normalizer PathNormalizer) {
	pathNormalizer = normalizer
}

// GetPathNormalizer 获取当前路径规范化器
func GetPathNormalizer() PathNormalizer {
	return pathNormalizer
}

var (
	// DefaultPolicies 默认策略（当没有配置时使用）
	DefaultPolicies = []Policy{
		{
			Name:          "default",
			DefaultEffect: EffectDeny,
			Enabled:       true,
			Rules: []Rule{
				{
					Name:        "allow-read-default",
					Description: "默认允许读取操作",
					Priority:    1000,
					Enabled:     true,
					Match: RuleMatchConditions{
						Operations: []Permission{PermissionRead},
					},
					Effect: EffectAllow,
				},
			},
		},
	}
)

// Engine 分层匹配引擎
type Engine struct {
	mu       sync.RWMutex
	policies []Policy
	roles    map[string]*Role
}

// NewEngine 创建新的权限引擎
func NewEngine() *Engine {
	engine := &Engine{
		policies: make([]Policy, 0),
		roles:    make(map[string]*Role),
	}

	// 加载默认策略
	_ = engine.LoadPolicies(DefaultPolicies)

	return engine
}

// Check 执行权限检查
func (e *Engine) Check(ctx context.Context, req Request) Result {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// 1. 规则引擎检查（优先级最高）
	ruleResult := e.checkRules(req)
	if ruleResult != nil {
		return *ruleResult
	}

	// 2. 角色权限检查（回退到角色）
	roleResult := e.checkRoles(req)
	if roleResult != nil {
		return *roleResult
	}

	// 3. 默认行为
	return e.defaultResult(req)
}

// checkRules 检查规则匹配
func (e *Engine) checkRules(req Request) *Result {
	// 按优先级排序所有规则
	rules := e.getAllRulesSorted()

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		// 检查操作类型是否匹配
		if !rule.MatchOperation(req.Operation) {
			continue
		}

		// 检查规则是否匹配请求
		if e.matchRule(rule, req) {
			if rule.Effect == EffectDeny {
				return &Result{
					Allowed:    false,
					MatchedRule: rule.Name,
					Reason:     "denied by rule: " + rule.Description,
				}
			}

			// 如果是允许规则，但引用了角色，需要进一步检查角色权限
			if rule.HasRoleReference() {
				roleName := *rule.Role
				role, ok := e.roles[roleName]
				if !ok {
					return &Result{
						Allowed:    false,
						MatchedRule: rule.Name,
						Reason:     "role not found: " + roleName,
					}
				}
				// 检查角色是否有此权限
				if role.HasPermissionWithContext(req.Operation, req.TargetPath, req.Provider, req.Bucket) {
					return &Result{
						Allowed:     true,
						MatchedRule: rule.Name,
						Reason:      "allowed by rule with role: " + rule.Description,
					}
				}
				continue
			}

			// 纯规则允许
			return &Result{
				Allowed:     true,
				MatchedRule: rule.Name,
				Reason:      "allowed by rule: " + rule.Description,
			}
		}
	}

	return nil
}

// checkRoles 检查角色权限
func (e *Engine) checkRoles(req Request) *Result {
	// 优先检查请求中声明的角色
	for _, roleName := range req.RequestedRoles {
		if role, ok := e.roles[roleName]; ok {
			if role.HasPermissionWithContext(req.Operation, req.TargetPath, req.Provider, req.Bucket) {
				return &Result{
					Allowed: true,
					Reason:  "allowed by role: " + role.Name,
				}
			}
		}
	}

	return nil
}

// defaultResult 返回默认结果
func (e *Engine) defaultResult(req Request) Result {
	// 获取默认效果
	defaultEffect := EffectDeny
	for _, policy := range e.policies {
		if policy.Enabled {
			defaultEffect = policy.DefaultEffect
			break
		}
	}

	// 严格遵守策略的默认效果配置
	// 不再对 Read 操作做特殊处理，这可能导致安全漏洞
	return Result{
		Allowed: defaultEffect == EffectAllow,
		Reason:  "default effect: " + string(defaultEffect),
	}
}

// matchRule 检查规则是否匹配请求
func (e *Engine) matchRule(rule Rule, req Request) bool {
	match := rule.Match

	// 1. 检查 Provider 匹配
	if len(match.Provider) > 0 {
		if !contains(match.Provider, req.Provider) {
			return false
		}
	}

	// 2. 检查 Bucket 匹配
	if len(match.Bucket) > 0 {
		if !wildcardMatchAny(req.Bucket, match.Bucket) {
			return false
		}
	}

	// 3. 检查路径前缀匹配
	if len(match.PathPrefix) > 0 {
		if !e.matchPathPrefix(req, match.PathPrefix) {
			return false
		}
	}

	// 4. 检查路径正则匹配
	if len(match.PathRegex) > 0 {
		if !e.matchPathRegex(req, match.PathRegex) {
			return false
		}
	}

	// 5. 检查文件扩展名匹配
	if len(match.Extension) > 0 {
		if !contains(match.Extension, req.Extension) {
			return false
		}
	}

	// 6. 检查文件大小范围
	if match.MinSize > 0 && req.FileSize < match.MinSize {
		return false
	}
	if match.MaxSize > 0 && req.FileSize > match.MaxSize {
		return false
	}

	return true
}

// matchPathPrefix 检查路径前缀匹配
func (e *Engine) matchPathPrefix(req Request, prefixes []string) bool {
	// 使用规范化路径进行匹配
	normalizedTarget := pathNormalizer(req.TargetPath)
	normalizedSource := pathNormalizer(req.SourcePath)

	for _, prefix := range prefixes {
		normalizedPrefix := pathNormalizer(prefix)
		if strings.HasPrefix(normalizedTarget, normalizedPrefix) {
			return true
		}
		if strings.HasPrefix(normalizedSource, normalizedPrefix) {
			return true
		}
	}
	return false
}

// matchPathRegex 检查路径正则匹配
func (e *Engine) matchPathRegex(req Request, patterns []string) bool {
	// 使用规范化路径进行匹配
	normalizedTarget := pathNormalizer(req.TargetPath)
	normalizedSource := pathNormalizer(req.SourcePath)

	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		if re.MatchString(normalizedTarget) || re.MatchString(normalizedSource) {
			return true
		}
	}
	return false
}

// getAllRulesSorted 按优先级排序所有规则
func (e *Engine) getAllRulesSorted() []Rule {
	var rules []Rule
	for _, policy := range e.policies {
		if !policy.Enabled {
			continue
		}
		rules = append(rules, policy.Rules...)
	}

	// 按优先级排序（数值越小优先级越高）
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority < rules[j].Priority
	})

	return rules
}

// LoadPolicies 加载策略
func (e *Engine) LoadPolicies(policies []Policy) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.policies = policies
	return nil
}

// AddPolicy 添加策略
func (e *Engine) AddPolicy(policy Policy) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.policies = append(e.policies, policy)
	return nil
}

// RemovePolicy 移除策略
func (e *Engine) RemovePolicy(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	newPolicies := make([]Policy, 0)
	for _, p := range e.policies {
		if p.Name != name {
			newPolicies = append(newPolicies, p)
		}
	}
	e.policies = newPolicies
	return nil
}

// GetRole 获取角色
func (e *Engine) GetRole(name string) (*Role, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	role, ok := e.roles[name]
	return role, ok
}

// AddRole 添加角色
func (e *Engine) AddRole(role Role) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.roles[role.Name] = &role
	return nil
}

// RemoveRole 移除角色
func (e *Engine) RemoveRole(name string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.roles[name]; ok {
		delete(e.roles, name)
		return true
	}
	return false
}

// ListRoles 列出所有角色
func (e *Engine) ListRoles() []*Role {
	e.mu.RLock()
	defer e.mu.RUnlock()

	roles := make([]*Role, 0, len(e.roles))
	for _, role := range e.roles {
		roles = append(roles, role)
	}
	return roles
}

// contains 检查切片是否包含元素
func contains(slice []string, item string) bool {
	if item == "" {
		return false
	}
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// wildcardMatchAny 检查字符串是否匹配任意通配符模式
func wildcardMatchAny(str string, patterns []string) bool {
	if str == "" {
		return false
	}
	for _, pattern := range patterns {
		if matchWildcard(str, pattern) {
			return true
		}
	}
	return false
}

// matchWildcard 简单的通配符匹配（支持 *）
func matchWildcard(str, pattern string) bool {
	if pattern == "" {
		return str == ""
	}
	if pattern == "*" {
		return true
	}

	// 简单实现：支持 * 前缀和后缀
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		mid := pattern[1 : len(pattern)-1]
		return strings.Contains(str, mid)
	}
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(str, pattern[1:])
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(str, pattern[:len(pattern)-1])
	}

	return str == pattern
}