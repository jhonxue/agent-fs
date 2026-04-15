package permission

import (
	"context"
	"sync"

	"github.com/geekjourneyx/agent-fs/pkg/apperr"
)

// ManagerConfig 权限管理器配置
type ManagerConfig struct {
	// 是否启用权限控制
	Enabled bool
	// 配置文件路径
	ConfigPath string
	// 严格模式（无匹配规则时拒绝）
	StrictMode bool
	// 是否启用索引驱动评估路径（灰度开关）
	EnableRuleIndexExecution bool
}

// Manager 权限管理器
type Manager struct {
	mu              sync.RWMutex
	config          *ManagerConfig
	engine          *Engine
	enabled         bool
	userRoleMapping map[string][]string // 用户名到角色列表的映射
}

// NewManager 创建权限管理器
func NewManager(cfg *ManagerConfig) *Manager {
	m := &Manager{
		config:          cfg,
		engine:          NewEngine(),
		enabled:         cfg.Enabled,
		userRoleMapping: make(map[string][]string),
	}

	// 设置引擎灰度开关
	m.engine.SetEnableRuleIndexExecution(cfg.EnableRuleIndexExecution)

	// 如果配置了文件路径，尝试加载配置
	if cfg.ConfigPath != "" {
		if err := m.LoadFromFile(cfg.ConfigPath); err != nil {
			// 加载失败使用默认配置
			m.engine = NewEngine()
			// 重新设置灰度开关
			m.engine.SetEnableRuleIndexExecution(cfg.EnableRuleIndexExecution)
		}
	}

	return m
}

// NewManagerWithConfig 使用 Config 创建权限管理器
func NewManagerWithConfig(cfg *Config) *Manager {
	m := &Manager{
		config: &ManagerConfig{
			Enabled:                  cfg.Enabled,
			ConfigPath:               cfg.ConfigPath,
			StrictMode:               cfg.StrictMode,
			EnableRuleIndexExecution: cfg.EnableRuleIndexExecution,
		},
		engine:          NewEngine(),
		enabled:         cfg.Enabled,
		userRoleMapping: make(map[string][]string),
	}

	// 加载角色
	for _, role := range cfg.Roles {
		m.engine.AddRole(role)
	}

	// 加载用户角色映射
	m.loadUserRoleMappings(cfg.UserRoleMappings)

	// 加载策略
	if err := m.engine.LoadPolicies(cfg.Policies); err != nil {
		// 加载失败使用默认策略
	}

	// 根据配置设置引擎灰度开关
	m.engine.SetEnableRuleIndexExecution(cfg.EnableRuleIndexExecution)

	return m
}

// loadUserRoleMappings 加载用户角色映射
func (m *Manager) loadUserRoleMappings(mappings []UserRoleMapping) {
	for _, mapping := range mappings {
		if mapping.User != "" {
			m.userRoleMapping[mapping.User] = mapping.Roles
		}
	}
}

// GetRolesForUser 获取用户对应的角色列表
func (m *Manager) GetRolesForUser(username string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	roles, ok := m.userRoleMapping[username]
	if !ok {
		return nil
	}
	return roles
}

// SetUserRoleMapping 设置用户角色映射
func (m *Manager) SetUserRoleMapping(username string, roles []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.userRoleMapping[username] = roles
}

// CheckPermission 检查权限
func (m *Manager) CheckPermission(ctx context.Context, req Request) Result {
	// 在一次锁操作中读取所有需要的状态，确保一致性
	m.mu.RLock()
	enabled := m.enabled
	var roles []string
	var hasRoles bool
	if req.CurrentUser != "" {
		roles, hasRoles = m.userRoleMapping[req.CurrentUser]
	}
	m.mu.RUnlock()

	// 如果未启用权限控制，允许所有操作
	if !enabled {
		return Result{
			Allowed: true,
			Reason:  "permission control disabled",
		}
	}

	// 如果请求中包含用户名且有映射角色，添加到请求中
	if hasRoles && len(roles) > 0 {
		newReq := req
		newReq.RequestedRoles = roles
		return m.engine.Check(ctx, newReq)
	}

	return m.engine.Check(ctx, req)
}

// CheckPermissionWithResult 检查权限并返回错误（如果拒绝）
func (m *Manager) CheckPermissionWithResult(ctx context.Context, req Request) error {
	result := m.CheckPermission(ctx, req)
	if !result.Allowed {
		if result.Error != nil {
			return result.Error
		}
		return apperr.New("permission", apperr.CodePermission, result.Reason)
	}
	return nil
}

// CheckOperationPermission 便捷方法：检查操作权限
func (m *Manager) CheckOperationPermission(
	ctx context.Context,
	op Permission,
	path string,
	provider string,
	bucket string,
) (bool, string, error) {
	req := Request{
		Operation:  op,
		TargetPath: path,
		Provider:   provider,
		Bucket:     bucket,
	}

	result := m.CheckPermission(ctx, req)
	return result.Allowed, result.Reason, result.Error
}

// Enable 启用权限控制
func (m *Manager) Enable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = true
}

// Disable 禁用权限控制
func (m *Manager) Disable() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = false
}

// IsEnabled 检查是否启用
func (m *Manager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
}

// LoadFromFile 从文件加载配置
func (m *Manager) LoadFromFile(path string) error {
	cfg, err := LoadConfig(path)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 加载角色
	for _, role := range cfg.Roles {
		m.engine.AddRole(role)
	}

	// 加载用户角色映射
	m.loadUserRoleMappings(cfg.UserRoleMappings)

	// 加载策略
	if err := m.engine.LoadPolicies(cfg.Policies); err != nil {
		return err
	}

	// 更新配置
	m.config.ConfigPath = path
	m.enabled = cfg.Enabled

	// 根据配置设置引擎灰度开关
	m.engine.SetEnableRuleIndexExecution(cfg.EnableRuleIndexExecution)

	return nil
}

// Reload 重新加载配置
func (m *Manager) Reload() error {
	if m.config.ConfigPath == "" {
		return nil
	}
	return m.LoadFromFile(m.config.ConfigPath)
}

// AddPolicy 添加策略
func (m *Manager) AddPolicy(policy Policy) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.engine.AddPolicy(policy)
}

// RemovePolicy 移除策略
func (m *Manager) RemovePolicy(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.engine.RemovePolicy(name)
}

// AddRole 添加角色
func (m *Manager) AddRole(role Role) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.engine.AddRole(role)
}

// RemoveRole 移除角色
func (m *Manager) RemoveRole(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.engine.RemoveRole(name)
}

// GetRole 获取角色
func (m *Manager) GetRole(name string) (*Role, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.engine.GetRole(name)
}

// ListRoles 列出所有角色
func (m *Manager) ListRoles() []*Role {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.engine.ListRoles()
}

// GetConfig 获取当前配置
func (m *Manager) GetConfig() *ManagerConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// SetStrictMode 设置严格模式
func (m *Manager) SetStrictMode(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.StrictMode = enabled
}

// 创建全局默认管理器
var (
	defaultManager     *Manager
	defaultManagerOnce sync.Once
)

// Default 返回默认权限管理器
func Default() *Manager {
	defaultManagerOnce.Do(func() {
		defaultManager = NewManager(&ManagerConfig{
			Enabled:    false, // 默认禁用
			StrictMode: true,
		})
	})
	return defaultManager
}

// InitDefault 初始化默认权限管理器
func InitDefault(cfg *ManagerConfig) {
	defaultManagerOnce.Do(func() {
		defaultManager = NewManager(cfg)
	})
}

// ResetDefault 重置默认权限管理器（包级函数）
// 警告：此函数必须在测试中使用，因为它会释放全局单例
func ResetDefault() {
	defaultManagerOnce = sync.Once{}
	defaultManager = nil
}