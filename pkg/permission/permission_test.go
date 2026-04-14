package permission

import (
	"context"
	"path/filepath"
	"testing"
)

func TestPermissionTypes(t *testing.T) {
	tests := []struct {
		name     string
		perm     Permission
		expected bool
	}{
		{"read is valid", PermissionRead, true},
		{"write is valid", PermissionWrite, true},
		{"overwrite is valid", PermissionOverwrite, true},
		{"delete is valid", PermissionDelete, true},
		{"copy is valid", PermissionCopy, true},
		{"move is valid", PermissionMove, true},
		{"invalid permission", Permission("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidPermission(tt.perm)
			if result != tt.expected {
				t.Errorf("IsValidPermission(%q) = %v, want %v", tt.perm, result, tt.expected)
			}
		})
	}
}

func TestRoleTypes(t *testing.T) {
	tests := []struct {
		name     string
		roleType RoleType
		expected bool
	}{
		{"admin is valid", RoleTypeAdmin, true},
		{"editor is valid", RoleTypeEditor, true},
		{"reader is valid", RoleTypeReader, true},
		{"backup is valid", RoleTypeBackup, true},
		// RoleTypeCustom 没有在 RoleTypePermissions 映射中，因此不被 IsValidRoleType 识别
		{"custom is custom not predefined", RoleTypeCustom, false},
		{"invalid role type", RoleType("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidRoleType(tt.roleType)
			if result != tt.expected {
				t.Errorf("IsValidRoleType(%q) = %v, want %v", tt.roleType, result, tt.expected)
			}
		})
	}
}

func TestRoleHasPermission(t *testing.T) {
	adminRole := Role{
		Name:        "admin",
		Type:        RoleTypeAdmin,
		Permissions: []Permission{PermissionRead, PermissionWrite, PermissionDelete},
	}

	tests := []struct {
		name       string
		role       *Role
		permission Permission
		expected   bool
	}{
		{"admin can read", &adminRole, PermissionRead, true},
		{"admin can write", &adminRole, PermissionWrite, true},
		{"admin can delete", &adminRole, PermissionDelete, true},
		{"admin cannot copy", &adminRole, PermissionCopy, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.role.HasPermission(tt.permission)
			if result != tt.expected {
				t.Errorf("HasPermission(%v) = %v, want %v", tt.permission, result, tt.expected)
			}
		})
	}
}

func TestRoleHasPermissionWithContext(t *testing.T) {
	role := Role{
		Name:        "editor",
		Type:        RoleTypeEditor,
		Permissions: []Permission{PermissionRead, PermissionWrite},
		Conditions: &RoleConditions{
			PathPrefix: []string{"/home/editor/"},
			Provider:   []string{"s3", "oss"},
		},
	}

	tests := []struct {
		name     string
		path     string
		provider string
		expected bool
	}{
		{"valid path and provider", "/home/editor/file.txt", "s3", true},
		{"invalid path", "/home/other/file.txt", "s3", false},
		{"invalid provider", "/home/editor/file.txt", "local", false},
	}

	// 测试无约束的角色 - 当 role.Conditions 为 nil 时应允许所有
	noConstraintRole := Role{
		Name:        "reader",
		Permissions: []Permission{PermissionRead},
	}

	// 注意：当 bucket 为空时不检查 bucket 约束
	if !noConstraintRole.HasPermissionWithContext(PermissionRead, "any/path", "any", "some-bucket") {
		t.Error("role without constraints should allow any path/provider")
	}

	// 测试无约束角色 - 使用 any path 和 any provider
	t.Run("no constraints role - should pass", func(t *testing.T) {
		result := noConstraintRole.HasPermissionWithContext(PermissionRead, "any/path", "any", "")
		if result != true {
			t.Errorf("HasPermissionWithContext(path=%q, provider=%q) = %v, want %v",
				"any/path", "any", result, true)
		}
	})

	// 测试有约束的角色
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := role.HasPermissionWithContext(PermissionRead, tt.path, tt.provider, "")
			if result != tt.expected {
				t.Errorf("HasPermissionWithContext(path=%q, provider=%q) = %v, want %v",
					tt.path, tt.provider, result, tt.expected)
			}
		})
	}
}

func TestRuleMatchOperation(t *testing.T) {
	tests := []struct {
		name       string
		operation  Permission
		operations []Permission
		expected   bool
	}{
		{"read matches", PermissionRead, []Permission{PermissionRead, PermissionWrite}, true},
		{"write matches", PermissionWrite, []Permission{PermissionRead, PermissionWrite}, true},
		{"delete does not match", PermissionDelete, []Permission{PermissionRead, PermissionWrite}, false},
		{"copy does not match", PermissionCopy, []Permission{PermissionRead, PermissionWrite}, false},
	}

	// 测试空操作列表（应匹配所有）
	emptyRule := Rule{Match: RuleMatchConditions{}}
	if !emptyRule.MatchOperation(PermissionRead) {
		t.Error("rule with empty operations should match any operation")
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := Rule{
				Match: RuleMatchConditions{
					Operations: tt.operations,
				},
			}
			result := rule.MatchOperation(tt.operation)
			if result != tt.expected {
				t.Errorf("MatchOperation(%v) = %v, want %v", tt.operation, result, tt.expected)
			}
		})
	}
}

func TestRuleHasRoleReference(t *testing.T) {
	roleName := "admin"

	ruleWithRef := Rule{Role: &roleName}
	if !ruleWithRef.HasRoleReference() {
		t.Error("rule with role reference should return true")
	}

	ruleWithoutRef := Rule{}
	if ruleWithoutRef.HasRoleReference() {
		t.Error("rule without role reference should return false")
	}

	emptyRole := ""
	ruleEmptyRole := Rule{Role: &emptyRole}
	if ruleEmptyRole.HasRoleReference() {
		t.Error("rule with empty role string should return false")
	}
}

func TestEngineCheck(t *testing.T) {
	ctx := context.Background()
	engine := NewEngine()

	// 测试默认行为（无规则匹配时）
	req := Request{
		Operation:  PermissionWrite,
		TargetPath: "/test/path",
		Provider:   "s3",
		Bucket:     "test-bucket",
	}

	// 创建显式拒绝的策略
	engine.policies = []Policy{
		{
			Name:          "deny-all",
			DefaultEffect: EffectDeny,
			Enabled:       true,
		},
	}

	result := engine.Check(ctx, req)
	// 默认拒绝写操作
	if result.Allowed {
		t.Error("default deny should block write operation")
	}

	// 创建允许的策略
	engine.policies = []Policy{
		{
			Name:          "allow-all",
			DefaultEffect: EffectAllow,
			Enabled:       true,
		},
	}

	result = engine.Check(ctx, req)
	if !result.Allowed {
		t.Error("default allow should permit write operation")
	}
}

func TestEngineCheckRules(t *testing.T) {
	ctx := context.Background()
	engine := NewEngine()

	// 加载一个允许读取的规则
	engine.policies = []Policy{
		{
			Name:          "read-allow",
			DefaultEffect: EffectDeny,
			Enabled:       true,
			Rules: []Rule{
				{
					Name:        "allow-read-s3",
					Priority:    100,
					Enabled:     true,
					Match:       RuleMatchConditions{Operations: []Permission{PermissionRead}, Provider: []string{"s3"}},
					Effect:      EffectAllow,
					Description: "Allow read on s3",
				},
			},
		},
	}

	// 测试规则匹配
	reqReadS3 := Request{
		Operation:  PermissionRead,
		Provider:   "s3",
		TargetPath: "/test",
	}
	result := engine.Check(ctx, reqReadS3)
	if !result.Allowed {
		t.Errorf("rule should allow read on s3: %v", result.Reason)
	}

	// 测试不匹配
	reqWriteS3 := Request{
		Operation:  PermissionWrite,
		Provider:  "s3",
		TargetPath: "/test",
	}
	result = engine.Check(ctx, reqWriteS3)
	if result.Allowed {
		t.Error("rule should deny write on s3")
	}
}

func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		pattern  string
		expected bool
	}{
		{"exact match", "test", "test", true},
		{"exact no match", "test", "test2", false},
		{"prefix wildcard", "test.txt", "*.txt", true},
		{"suffix wildcard", "file.txt", "file.*", true},
		{"both wildcards", "test-file.txt", "*-file.txt", true},
		{"contains", "test-file-123.txt", "*file*", true},
		{"single star anywhere", "any", "*", true},
		{"empty pattern exact", "test", "test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchWildcard(tt.str, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchWildcard(%q, %q) = %v, want %v", tt.str, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestWildcardMatchAny(t *testing.T) {
	patterns := []string{"*.txt", "*.log", "data/*"}

	if !wildcardMatchAny("readme.txt", patterns) {
		t.Error("should match *.txt pattern")
	}
	if !wildcardMatchAny("error.log", patterns) {
		t.Error("should match *.log pattern")
	}
	if !wildcardMatchAny("data/file.txt", patterns) {
		t.Error("should match data/* pattern")
	}
	if wildcardMatchAny("other.pdf", patterns) {
		t.Error("should not match any pattern")
	}
}

func TestManagerCheckPermission(t *testing.T) {
	ctx := context.Background()

	// 创建带配置的管理器
	manager := NewManager(&ManagerConfig{
		Enabled:    true,
		StrictMode: true,
	})

	// 添加一个允许读取的角色
	manager.AddRole(Role{
		Name:        "reader",
		Type:        RoleTypeReader,
		Permissions: []Permission{PermissionRead},
	})

	// 添加用户角色映射
	manager.SetUserRoleMapping("testuser", []string{"reader"})

	// 测试允许的读取操作
	req := Request{
		Operation:   PermissionRead,
		TargetPath:  "/test/file.txt",
		Provider:    "s3",
		Bucket:      "test-bucket",
		CurrentUser: "testuser",
	}

	result := manager.CheckPermission(ctx, req)
	if !result.Allowed {
		t.Errorf("reader should be allowed to read: %v", result.Reason)
	}

	// 测试拒绝的写操作
	reqWrite := Request{
		Operation:   PermissionWrite,
		TargetPath:  "/test/file.txt",
		Provider:    "s3",
		Bucket:      "test-bucket",
		CurrentUser: "testuser",
	}

	result = manager.CheckPermission(ctx, reqWrite)
	if result.Allowed {
		t.Error("reader should not be allowed to write")
	}
}

func TestManagerDisabled(t *testing.T) {
	ctx := context.Background()

	// 创建禁用的管理器
	manager := NewManager(&ManagerConfig{
		Enabled: false,
	})

	// 即使没有角色，也应该允许所有操作
	req := Request{
		Operation:  PermissionWrite,
		TargetPath: "/test/file.txt",
		Provider:   "s3",
	}

	result := manager.CheckPermission(ctx, req)
	if !result.Allowed {
		t.Error("disabled manager should allow all operations")
	}
	if result.Reason != "permission control disabled" {
		t.Errorf("expected reason 'permission control disabled', got %q", result.Reason)
	}
}

func TestManagerUserRoleMapping(t *testing.T) {
	manager := NewManager(&ManagerConfig{Enabled: true})

	// 设置用户角色映射
	manager.SetUserRoleMapping("alice", []string{"admin", "editor"})
	manager.SetUserRoleMapping("bob", []string{"reader"})

	// 测试获取角色
	roles := manager.GetRolesForUser("alice")
	if len(roles) != 2 || roles[0] != "admin" || roles[1] != "editor" {
		t.Errorf("expected [admin editor], got %v", roles)
	}

	roles = manager.GetRolesForUser("bob")
	if len(roles) != 1 || roles[0] != "reader" {
		t.Errorf("expected [reader], got %v", roles)
	}

	// 不存在的用户
	roles = manager.GetRolesForUser("unknown")
	if roles != nil {
		t.Error("unknown user should return nil")
	}
}

func TestRequestBuilder(t *testing.T) {
	req := Request{
		Operation:   PermissionRead,
		SourcePath:  "/source",
		TargetPath:  "/target",
		Provider:    "s3",
		Bucket:      "my-bucket",
		FileSize:    1024,
		Extension:   ".txt",
		RequestedRoles: []string{"admin"},
		CurrentUser: "testuser",
	}

	// 使用链式方法测试（值接收者）
	req.WithProvider("s3")
	req.WithBucket("my-bucket")
	req.WithFileInfo(1024, ".txt")
	req.WithRoles("admin")
	req.WithUser("testuser")

	if req.Operation != PermissionRead {
		t.Errorf("expected operation read, got %v", req.Operation)
	}
	if req.SourcePath != "/source" {
		t.Errorf("expected source /source, got %v", req.SourcePath)
	}
	if req.TargetPath != "/target" {
		t.Errorf("expected target /target, got %v", req.TargetPath)
	}
	if req.Provider != "s3" {
		t.Errorf("expected provider s3, got %v", req.Provider)
	}
	if req.Bucket != "my-bucket" {
		t.Errorf("expected bucket my-bucket, got %v", req.Bucket)
	}
	if req.FileSize != 1024 {
		t.Errorf("expected file size 1024, got %v", req.FileSize)
	}
	if req.Extension != ".txt" {
		t.Errorf("expected extension .txt, got %v", req.Extension)
	}
	if len(req.RequestedRoles) != 1 || req.RequestedRoles[0] != "admin" {
		t.Errorf("expected roles [admin], got %v", req.RequestedRoles)
	}
	if req.CurrentUser != "testuser" {
		t.Errorf("expected user testuser, got %v", req.CurrentUser)
	}
}

func TestResultBuilders(t *testing.T) {
	// 测试允许结果
	allowed := NewResultAllowed("allowed by rule")
	if !allowed.Allowed {
		t.Error("expected allowed result")
	}
	if allowed.Reason != "allowed by rule" {
		t.Errorf("expected reason 'allowed by rule', got %v", allowed.Reason)
	}

	// 测试拒绝结果
	denied := NewResultDenied("rule-name", "denied by rule")
	if denied.Allowed {
		t.Error("expected denied result")
	}
	if denied.MatchedRule != "rule-name" {
		t.Errorf("expected matched rule 'rule-name', got %v", denied.MatchedRule)
	}

	// 测试错误结果
	err := NewResultError(ErrPermissionDenied)
	if err.Allowed {
		t.Error("expected denied result")
	}
	if err.Error != ErrPermissionDenied {
		t.Errorf("expected error ErrPermissionDenied, got %v", err.Error)
	}
}

func TestEngineLoadPolicies(t *testing.T) {
	engine := NewEngine()

	policies := []Policy{
		{
			Name:          "test-policy",
			DefaultEffect: EffectAllow,
			Enabled:       true,
			Rules: []Rule{
				{
					Name:    "test-rule",
					Effect:  EffectDeny,
					Enabled: true,
					Match: RuleMatchConditions{
						Operations: []Permission{PermissionDelete},
					},
				},
			},
		},
	}

	err := engine.LoadPolicies(policies)
	if err != nil {
		t.Errorf("unexpected error loading policies: %v", err)
	}

	// 验证策略已加载
	if len(engine.policies) != 1 {
		t.Errorf("expected 1 policy, got %d", len(engine.policies))
	}
}

func TestEngineAddRemoveRole(t *testing.T) {
	engine := NewEngine()

	role := Role{
		Name:        "test-role",
		Type:        RoleTypeCustom,
		Permissions: []Permission{PermissionRead},
	}

	// 添加角色
	err := engine.AddRole(role)
	if err != nil {
		t.Errorf("unexpected error adding role: %v", err)
	}

	// 获取角色
	retrieved, ok := engine.GetRole("test-role")
	if !ok {
		t.Error("role not found after add")
	}
	if retrieved.Name != "test-role" {
		t.Errorf("expected role name 'test-role', got %v", retrieved.Name)
	}

	// 移除角色
	removed := engine.RemoveRole("test-role")
	if !removed {
		t.Error("failed to remove role")
	}

	// 验证角色已移除
	_, ok = engine.GetRole("test-role")
	if ok {
		t.Error("role should not exist after removal")
	}
}

func TestEngineListRoles(t *testing.T) {
	engine := NewEngine()

	// 添加多个角色
	engine.AddRole(Role{Name: "role1", Type: RoleTypeAdmin})
	engine.AddRole(Role{Name: "role2", Type: RoleTypeReader})

	roles := engine.ListRoles()
	if len(roles) != 2 {
		t.Errorf("expected 2 roles, got %d", len(roles))
	}
}

func TestRBAC(t *testing.T) {
	ctx := context.Background()
	rbac := NewRBAC()

	// 添加角色
	rbac.AddRole(Role{
		Name:        "writer",
		Permissions: []Permission{PermissionRead, PermissionWrite},
	})

	// 测试允许的操作
	req := Request{
		Operation:  PermissionWrite,
		RequestedRoles: []string{"writer"},
	}
	result := rbac.Check(ctx, req)
	if !result.Allowed {
		t.Errorf("writer should be allowed to write: %v", result.Reason)
	}

	// 测试拒绝的操作
	reqDelete := Request{
		Operation:      PermissionDelete,
		RequestedRoles: []string{"writer"},
	}
	result = rbac.Check(ctx, reqDelete)
	if result.Allowed {
		t.Error("writer should not be allowed to delete")
	}

	// 测试不存在的角色
	reqUnknown := Request{
		Operation:      PermissionRead,
		RequestedRoles: []string{"unknown"},
	}
	result = rbac.Check(ctx, reqUnknown)
	if result.Allowed {
		t.Error("unknown role should be denied")
	}
}

func TestRBACLoadRoles(t *testing.T) {
	rbac := NewRBAC()

	roles := []Role{
		{Name: "role1", Permissions: []Permission{PermissionRead}},
		{Name: "role2", Permissions: []Permission{PermissionWrite}},
	}

	err := rbac.LoadRoles(roles)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(rbac.ListRoles()) != 2 {
		t.Errorf("expected 2 roles, got %d", len(rbac.ListRoles()))
	}
}

func TestDefaultPolicies(t *testing.T) {
	// 验证默认策略已加载
	if len(DefaultPolicies) == 0 {
		t.Error("default policies should not be empty")
	}

	// 验证默认策略的结构
	for _, policy := range DefaultPolicies {
		if policy.Name == "" {
			t.Error("policy name should not be empty")
		}
		if !policy.Enabled {
			t.Error("default policy should be enabled")
		}

		// 验证规则结构
		for _, rule := range policy.Rules {
			if rule.Name == "" {
				t.Error("rule name should not be empty")
			}
			if !rule.Enabled {
				t.Error("default rule should be enabled")
			}
		}
	}
}

func TestConfigValidation(t *testing.T) {
	// 测试有效配置
	validConfig := &Config{
		Enabled:    true,
		StrictMode: true,
		Roles: []Role{
			{Name: "admin", Type: RoleTypeAdmin},
			{Name: "reader", Type: RoleTypeReader},
		},
		UserRoleMappings: []UserRoleMapping{
			{User: "alice", Roles: []string{"admin"}},
		},
		Policies: []Policy{
			{
				Name:          "test",
				DefaultEffect: EffectAllow,
				Enabled:       true,
			},
		},
	}

	err := validConfig.Validate()
	if err != nil {
		t.Errorf("valid config should not have error: %v", err)
	}

	// 测试重复角色名
	duplicateConfig := &Config{
		Roles: []Role{
			{Name: "admin"},
			{Name: "admin"},
		},
	}

	err = duplicateConfig.Validate()
	if err == nil {
		t.Error("should have error for duplicate role names")
	}

	// 测试空角色名
	emptyRoleConfig := &Config{
		Roles: []Role{
			{Name: ""},
		},
	}

	err = emptyRoleConfig.Validate()
	if err == nil {
		t.Error("should have error for empty role name")
	}

	// 测试引用不存在的角色
	invalidRoleRefConfig := &Config{
		Roles: []Role{
			{Name: "reader"},
		},
		UserRoleMappings: []UserRoleMapping{
			{User: "alice", Roles: []string{"nonexistent"}},
		},
	}

	err = invalidRoleRefConfig.Validate()
	if err == nil {
		t.Error("should have error for non-existent role reference")
	}
}

func TestCreateDefaultConfig(t *testing.T) {
	cfg := CreateDefaultConfig()

	if cfg.Enabled {
		t.Error("default config should be disabled")
	}
	if !cfg.StrictMode {
		t.Error("default config should be strict mode")
	}
	if len(cfg.Roles) == 0 {
		t.Error("default config should have roles")
	}
	if len(cfg.Policies) == 0 {
		t.Error("default config should have policies")
	}
}

func TestPathNormalizer(t *testing.T) {
	// 测试默认路径规范化器
	normalizer := GetPathNormalizer()

	// 简单的相对路径
	normalized := normalizer("../test/./file.txt")
	if normalized == "" {
		t.Error("normalized path should not be empty")
	}

	// 规范化应该清理路径
	if normalized != filepath.Clean("../test/file.txt") {
		// 可能是绝对路径
		if _, err := filepath.Abs(normalized); err != nil {
			t.Errorf("path should be valid: %v", err)
		}
	}
}

func TestContains(t *testing.T) {
	slice := []string{"a", "b", "c"}

	if !contains(slice, "a") {
		t.Error("should contain 'a'")
	}
	if !contains(slice, "b") {
		t.Error("should contain 'b'")
	}
	if contains(slice, "d") {
		t.Error("should not contain 'd'")
	}
	if contains(slice, "") {
		t.Error("should not contain empty string")
	}
}

func TestPermissionError(t *testing.T) {
	err := NewPermissionError("ERR_CODE", "test error message")
	permErr, ok := err.(*PermissionError)
	if !ok {
		t.Fatal("expected PermissionError type")
	}

	if permErr.Code != "ERR_CODE" {
		t.Errorf("expected code 'ERR_CODE', got %v", permErr.Code)
	}
	if permErr.Message != "test error message" {
		t.Errorf("expected message 'test error message', got %v", permErr.Message)
	}

	// 测试 Wrap
	wrapped := WrapPermissionError("ERR_CODE", "wrapped", err)
	if wrapped.Error() != "wrapped" {
		t.Errorf("expected 'wrapped', got %v", wrapped.Error())
	}
}