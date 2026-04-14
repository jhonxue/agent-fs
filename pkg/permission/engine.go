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

// RuleIndex 规则索引，用于快速定位候选规则
type RuleIndex struct {
	mu sync.RWMutex
	// 按 Provider 索引
	byProvider map[string][]Rule
	// 按 Bucket 通配符索引
	byBucket map[string][]Rule
	// 按路径前缀索引（存储前缀长度用于排序）
	byPathPrefix []pathPrefixRule
	// 全量规则（保留用于无法索引匹配的场景）
	allRules []Rule
	// 已排序的规则（按优先级）
	sortedRules []Rule
	// Provider 列表缓存 set
	providerSet map[string]struct{}
	// Bucket 列表缓存 set
	bucketSet map[string]struct{}
	// Extension 列表缓存 set
	extSet map[string]struct{}
}

type pathPrefixRule struct {
	prefix string
	rule   Rule
	length int
}

// NewRuleIndex 创建新的规则索引
func NewRuleIndex() *RuleIndex {
	return &RuleIndex{
		byProvider:  make(map[string][]Rule),
		byBucket:   make(map[string][]Rule),
		allRules:   make([]Rule, 0),
		sortedRules: make([]Rule, 0),
		providerSet: make(map[string]struct{}),
		bucketSet:   make(map[string]struct{}),
		extSet:      make(map[string]struct{}),
	}
}

// Build 从策略构建索引
func (ri *RuleIndex) Build(policies []Policy) {
	ri.mu.Lock()
	defer ri.mu.Unlock()

	// 清空现有索引
	ri.byProvider = make(map[string][]Rule)
	ri.byBucket = make(map[string][]Rule)
	ri.byPathPrefix = make([]pathPrefixRule, 0)
	ri.allRules = make([]Rule, 0)
	ri.sortedRules = make([]Rule, 0)
	ri.providerSet = make(map[string]struct{})
	ri.bucketSet = make(map[string]struct{})
	ri.extSet = make(map[string]struct{})

	// 遍历所有策略和规则
	for _, policy := range policies {
		if !policy.Enabled {
			continue
		}
		for _, rule := range policy.Rules {
			if !rule.Enabled {
				continue
			}

			// 添加到全量规则
			ri.allRules = append(ri.allRules, rule)

			// 构建 Provider 索引
			for _, p := range rule.Match.Provider {
				ri.providerSet[p] = struct{}{}
				ri.byProvider[p] = append(ri.byProvider[p], rule)
			}

			// 构建 Bucket 索引
			for _, b := range rule.Match.Bucket {
				ri.bucketSet[b] = struct{}{}
				ri.byBucket[b] = append(ri.byBucket[b], rule)
			}

			// 构建 PathPrefix 索引
			for _, prefix := range rule.Match.PathPrefix {
				ri.byPathPrefix = append(ri.byPathPrefix, pathPrefixRule{
					prefix: prefix,
					rule:   rule,
					length: len(prefix),
				})
			}

			// 构建 Extension 索引
			for _, ext := range rule.Match.Extension {
				ri.extSet[ext] = struct{}{}
			}
		}
	}

	// 对路径前缀按长度降序排序（更长的前缀优先匹配）
	sort.Slice(ri.byPathPrefix, func(i, j int) bool {
		return ri.byPathPrefix[i].length > ri.byPathPrefix[j].length
	})

	// 复制并排序全量规则（按优先级）
	ri.sortedRules = make([]Rule, len(ri.allRules))
	copy(ri.sortedRules, ri.allRules)
	sort.Slice(ri.sortedRules, func(i, j int) bool {
		return ri.sortedRules[i].Priority < ri.sortedRules[j].Priority
	})

	// 排序各索引中的规则
	for _, rules := range ri.byProvider {
		sort.Slice(rules, func(i, j int) bool { return rules[i].Priority < rules[j].Priority })
	}
	for _, rules := range ri.byBucket {
		sort.Slice(rules, func(i, j int) bool { return rules[i].Priority < rules[j].Priority })
	}
}

// HasProvider 检查 Provider 是否在索引中
func (ri *RuleIndex) HasProvider(provider string) bool {
	ri.mu.RLock()
	defer ri.mu.RUnlock()
	_, ok := ri.providerSet[provider]
	return ok
}

// HasBucket 检查 Bucket 是否在索引中
func (ri *RuleIndex) HasBucket(bucket string) bool {
	ri.mu.RLock()
	defer ri.mu.RUnlock()
	_, ok := ri.bucketSet[bucket]
	return ok
}

// HasExtension 检查 Extension 是否在索引中
func (ri *RuleIndex) HasExtension(ext string) bool {
	ri.mu.RLock()
	defer ri.mu.RUnlock()
	_, ok := ri.extSet[ext]
	return ok
}

// GetRulesByProvider 获取 Provider 对应的规则
func (ri *RuleIndex) GetRulesByProvider(provider string) []Rule {
	ri.mu.RLock()
	defer ri.mu.RUnlock()
	return ri.byProvider[provider]
}

// GetRulesByBucket 获取 Bucket 对应的规则
func (ri *RuleIndex) GetRulesByBucket(bucket string) []Rule {
	ri.mu.RLock()
	defer ri.mu.RUnlock()
	return ri.byBucket[bucket]
}

// GetSortedRules 获取已按优先级排序的规则
func (ri *RuleIndex) GetSortedRules() []Rule {
	ri.mu.RLock()
	defer ri.mu.RUnlock()
	return ri.sortedRules
}

// maxRegexCacheSize 正则表达式缓存最大容量
const maxRegexCacheSize = 100

// Engine 分层匹配引擎
type Engine struct {
	mu       sync.RWMutex
	policies []Policy
	roles    map[string]*Role
	// 正则表达式 LRU 缓存：避免每次匹配时重新编译正则
	regexCache *LRUCache
	// 规则索引：加速规则查找
	ruleIndex *RuleIndex
}

// NewEngine 创建新的权限引擎
func NewEngine() *Engine {
	engine := &Engine{
		policies:   make([]Policy, 0),
		roles:      make(map[string]*Role),
		regexCache: NewLRUCache(maxRegexCacheSize),
		ruleIndex:  NewRuleIndex(),
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

// LRUCache 正则表达式 LRU 缓存
type LRUCache struct {
	mu       sync.RWMutex
	cache    map[string]*regexp.Regexp
	order    []string
	maxSize  int
}

// NewLRUCache 创建新的 LRU 缓存
func NewLRUCache(maxSize int) *LRUCache {
	return &LRUCache{
		cache:   make(map[string]*regexp.Regexp),
		order:   make([]string, 0, maxSize),
		maxSize: maxSize,
	}
}

// Get 获取缓存的正则表达式，并更新访问顺序（LRU 策略）
func (c *LRUCache) Get(pattern string) (*regexp.Regexp, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	re, ok := c.cache[pattern]
	if ok {
		// 标记为最近使用
		c.moveToEnd(pattern)
	}
	return re, ok
}

// Put 添加缓存项，返回是否成功添加
func (c *LRUCache) Put(pattern string, re *regexp.Regexp) {
	if re == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查是否已存在
	if _, ok := c.cache[pattern]; ok {
		// 移动到末尾（最新）
		c.moveToEnd(pattern)
		return
	}

	// 超过容量时淘汰最早的项
	if len(c.cache) >= c.maxSize && len(c.order) > 0 {
		oldest := c.order[0]
		delete(c.cache, oldest)
		c.order = c.order[1:]
	}

	// 添加新项
	c.cache[pattern] = re
	c.order = append(c.order, pattern)
}

// moveToEnd 将指定项移动到末尾
func (c *LRUCache) moveToEnd(pattern string) {
	for i, p := range c.order {
		if p == pattern {
			c.order = append(c.order[:i], c.order[i+1:]...)
			c.order = append(c.order, pattern)
			return
		}
	}
}

// getRegexFromCache 从 LRU 缓存获取正则表达式
func (e *Engine) getRegexFromCache(pattern string) *regexp.Regexp {
	// 先尝试从缓存获取
	if re, ok := e.regexCache.Get(pattern); ok {
		return re
	}

	// 未命中，编译
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}

	// 添加到缓存
	e.regexCache.Put(pattern, re)
	return re
}

// matchPathRegex 检查路径正则匹配
func (e *Engine) matchPathRegex(req Request, patterns []string) bool {
	// 使用规范化路径进行匹配
	normalizedTarget := pathNormalizer(req.TargetPath)
	normalizedSource := pathNormalizer(req.SourcePath)

	for _, pattern := range patterns {
		re := e.getRegexFromCache(pattern)
		if re == nil {
			continue
		}
		if re.MatchString(normalizedTarget) || re.MatchString(normalizedSource) {
			return true
		}
	}
	return false
}

// getAllRulesSorted 按优先级排序所有规则（使用规则索引优化）
func (e *Engine) getAllRulesSorted() []Rule {
	// 直接使用已排序的规则索引
	return e.ruleIndex.GetSortedRules()
}

// LoadPolicies 加载策略并构建索引
func (e *Engine) LoadPolicies(policies []Policy) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.policies = policies
	// 构建规则索引以加速查找
	e.ruleIndex.Build(policies)
	return nil
}

// AddPolicy 添加策略并重建索引
func (e *Engine) AddPolicy(policy Policy) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.policies = append(e.policies, policy)
	// 重建索引
	e.ruleIndex.Build(e.policies)
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