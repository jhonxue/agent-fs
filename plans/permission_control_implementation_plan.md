# 权限控制系统实施计划（混合方案）

## 1. 项目概览

本文档将设计文档 `plans/permission_control_plan.md` 转化为可执行的具体实施步骤。
采用**混合方案**：规则模型 + RBAC 角色结合。

### 1.1 核心设计决策

- **规则引擎**：提供灵活的路径、Provider、Bucket 匹配
- **角色系统**：预定义权限模板，支持复用
- **优先级**：显式规则 > 角色权限 > 默认行为

### 1.2 混合模型架构

```
┌─────────────────────────────────────────────────────────────────┐
│                     权限检查请求                                  │
│        (operation, source_path, target_path, bucket)           │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      规则匹配层                                  │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │ 显式规则检查                                             │   │
│   │ - path_prefix / path_regex 匹配                         │   │
│   │ - provider / bucket 匹配                                │   │
│   │ - operation 匹配                                        │   │
│   └───────────────────────┬─────────────────────────────────┘   │
│                           │ 匹配到规则                           │
│                           ▼                                     │
│                    ┌──────────────┐                              │
│                    │ 返回决策     │                              │
│                    │ (allow/deny) │                              │
│                    └──────────────┘                              │
│                           │ 未匹配规则                           │
│                           ▼                                     │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      角色权限层                                  │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │ 1. 解析请求中的 role 标签                                │   │
│   │ 2. 查找角色定义                                          │   │
│   │ 3. 检查角色是否具有请求的操作权限                         │   │
│   └─────────────────────────────────────────────────────────┘   │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      默认行为                                    │
│   - overwrite/delete/copy/move: deny (默认拒绝)               │
│   - read: allow (默认允许)                                      │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. 文件创建/修改清单

### 2.1 新建文件列表（pkg/permission/）

| 文件 | 描述 | 优先级 | 状态 |
|------|------|--------|------|
| [`pkg/permission/types.go`](pkg/permission/types.go) | 权限类型定义：Permission、Effect、Role | P0 | ✅ 已完成 |
| [`pkg/permission/rule.go`](pkg/permission/rule.go) | 规则结构定义：Rule、Policy、RuleMatchConditions | P0 | ✅ 已完成 |
| [`pkg/permission/role.go`](pkg/permission/role.go) | 角色结构定义：Role、RoleBinding | P0 | ✅ 已完成 |
| [`pkg/permission/request.go`](pkg/permission/request.go) | 权限检查请求和结果结构 | P0 | ✅ 已完成 |
| [`pkg/permission/checker.go`](pkg/permission/checker.go) | PermissionChecker 接口定义 | P0 | ✅ 已完成 |
| [`pkg/permission/engine.go`](pkg/permission/engine.go) | 规则匹配引擎实现 | P0 | ✅ 已完成 |
| [`pkg/permission/rbac.go`](pkg/permission/rbac.go) | 角色权限评估器 | P1 | ✅ 已完成 |
| [`pkg/permission/config.go`](pkg/permission/config.go) | 权限配置加载器（支持规则+角色） | P1 | ✅ 已完成 |
| [`pkg/permission/manager.go`](pkg/permission/manager.go) | 权限管理器主类 | P1 | ✅ 已完成 |
| [`pkg/permission/evaluator.go`](pkg/permission/evaluator.go) | Evaluator 接口和 RuleEvaluator/RoleEvaluator 实现 | P1 | ✅ 已完成 |
| [`pkg/permission/errors.go`](pkg/permission/errors.go) | 权限相关错误定义 | P2 | ✅ 已完成 |

> **注意**：`context.go` 文件被取消，相关功能已整合到 `manager.go` 中。

### 2.2 修改文件列表

| 文件 | 修改内容 | 优先级 | 状态 |
|------|----------|--------|------|
| [`pkg/provider/provider.go`](pkg/provider/provider.go) | 扩展 StorageProvider 接口，新增 CheckPermission 方法 | P1 | ⏳ 待实施 |
| [`pkg/apperr/error.go`](pkg/apperr/error.go) | 新增权限错误码 CodePermission | P1 | ✅ 已完成 |
| [`cmd/fs.go`](cmd/fs.go) | 在 cp 等操作中添加权限检查调用 | P1 | ✅ 已完成 |
| [`cmd/root.go`](cmd/root.go) | 初始化权限管理器 | P2 | ✅ 已完成 |
| [`pkg/config/config.go`](pkg/config/config.go) | 支持权限配置加载 | P2 | ⏳ 待实施 |

---

## 3. 实现依赖顺序

### 阶段 0：基础类型定义（P0）

```
1. pkg/permission/types.go (扩展角色类型)
   ↓
2. pkg/permission/rule.go (规则结构)
   ↓
3. pkg/permission/role.go (角色结构) ← 新增
   ↓
4. pkg/permission/request.go (请求结构)
```

**交付物**：权限类型、规则结构、角色结构定义完成

### 阶段 1：核心引擎（P0）

```
5. pkg/permission/checker.go (接口)
   ↓
6. pkg/permission/engine.go (规则匹配)
   ↓
7. pkg/permission/rbac.go (角色评估) ← 新增
```

**交付物**：规则匹配引擎和角色权限引擎完成

### 阶段 2：配置和加载（P1）

```
8. pkg/permission/config.go (支持规则+角色)
   ↓
9. pkg/permission/manager.go (统一管理器)
   ↓
10. pkg/permission/errors.go (错误定义)
```

**交付物**：可从 YAML 加载配置并执行检查

### 阶段 3：Provider 接口扩展（P1）

```
11. pkg/provider/provider.go
    + SupportPermissionCheck() bool
    + CheckPermission(ctx, op, path) (bool, error)
```

**交付物**：Provider 接口扩展完成

### 阶段 4：CLI 集成（P1）

```
12. pkg/apperr/error.go 新增错误码
    ↓
13. cmd/fs.go 集成权限检查
```

**交付物**：CLI 命令具备权限控制能力

### 阶段 5：初始化和入口（P2）

```
14. cmd/root.go 初始化权限管理器
    ↓
15. pkg/config/config.go 支持权限配置
```

**交付物**：应用启动时自动加载权限配置

---

## 4. 各步骤工作量估算

### P0 关键路径（约 4-5 小时）

| 步骤 | 文件 | 工作量 | 说明 |
|------|------|--------|------|
| 1 | types.go | 0.5h | 定义 Permission、Effect、Role |
| 2 | rule.go | 1h | Rule、Policy、MatchConditions |
| 3 | role.go | 1h | **新增**：Role、RoleBinding |
| 4 | request.go | 0.5h | Request、Result |
| 5 | checker.go | 0.5h | 接口定义 |
| 6 | engine.go | 1.5h | 规则匹配逻辑 |
| 7 | rbac.go | 1.5h | **新增**：角色权限评估 |

### P1 重要功能（约 3 小时）

| 步骤 | 文件 | 工作量 | 说明 |
|------|------|--------|------|
| 8 | config.go | 1h | YAML 配置解析 |
| 9 | manager.go | 1h | 管理器逻辑 |
| 10 | errors.go | 0.5h | 错误定义 |
| 11 | provider.go | 0.5h | 接口扩展 |

### P1 集成（约 1.5 小时）

| 步骤 | 文件 | 工作量 | 说明 |
|------|------|--------|------|
| 12 | error.go | 0.5h | 新增错误码 |
| 13 | fs.go | 1h | 集成检查调用 |

### P2 完善（约 1 小时）

| 步骤 | 文件 | 工作量 | 说明 |
|------|------|--------|------|
| 14 | root.go | 0.5h | 初始化 |
| 15 | config.go | 0.5h | 配置支持 |

---

## 5. 数据模型设计

### 5.1 权限类型（types.go 扩展）

```go
// 权限操作类型
type Permission string

const (
    PermissionRead     Permission = "read"
    PermissionWrite   Permission = "write"
    PermissionOverwrite Permission = "overwrite"
    PermissionDelete  Permission = "delete"
    PermissionCopy    Permission = "copy"
    PermissionMove    Permission = "move"
)

// 权限效果
type Effect string

const (
    EffectAllow Effect = "allow"
    EffectDeny  Effect = "deny"
)

// 角色类型（新增）
type RoleType string

const (
    RoleTypeAdmin   RoleType = "admin"   // 管理员：所有权限
    RoleTypeEditor RoleType = "editor"   // 编辑：read/write/overwrite
    RoleTypeReader RoleType = "reader"   // 读取者：read only
    RoleTypeBackup RoleType = "backup"   // 备份员：read/copy
    RoleTypeCustom RoleType = "custom"   // 自定义角色
)
```

### 5.2 角色定义（role.go）

```go
// Role 定义一组权限
type Role struct {
    Name        string       `mapstructure:"name"`
    Type        RoleType     `mapstructure:"type"`
    Description string       `mapstructure:"description"`
    Permissions []Permission `mapstructure:"permissions"`
    
    // 约束条件（可选）
    Conditions  *RoleConditions `mapstructure:"conditions"`
}

type RoleConditions struct {
    // 路径约束
    PathPrefix []string `mapstructure:"path_prefix"`
    // Provider 约束
    Provider []string `mapstructure:"provider"`
    // Bucket 约束
    Bucket []string `mapstructure:"bucket"`
}

// RoleBinding 将角色绑定到上下文
type RoleBinding struct {
    Name      string   `mapstructure:"name"`
    Roles     []string `mapstructure:"roles"`     // 角色名称列表
    Subjects  []string `mapstructure:"subjects"`  // 绑定主体（保留，未来支持）
}
```

### 5.3 规则扩展（rule.go）

```go
// 规则支持引用角色
type Rule struct {
    Name        string   `mapstructure:"name"`
    Description string   `mapstructure:"description"`
    Match       RuleMatchConditions `mapstructure:"match"`
    Effect      Effect   `mapstructure:"effect"`
    Priority    int      `mapstructure:"priority"`
    Enabled     bool     `mapstructure:"enabled"`
    
    // 新增：引用角色
    Role *string `mapstructure:"role"`
}
```

### 5.4 权限请求（request.go 扩展）

```go
type Request struct {
    Operation   Permission
    SourcePath  string
    TargetPath  string
    Provider    string
    Bucket      string
    FileSize    int64
    Extension   string
    
    // 新增：角色上下文
    RequestedRoles []string  // 请求方声明的角色
    CurrentUser    string    // 当前用户（可选）
}
```

---

## 6. 配置示例（YAML）

```yaml
# ~/.afs-permission.yaml

permission:
  enabled: true
  config_path: ~/.afs-permission.yaml
  strict_mode: true

# ============================================
# 角色定义（新增角色层）
# ============================================
roles:
  - name: "admin"
    type: "admin"
    description: "管理员：所有权限"
    permissions:
      - "read"
      - "write"
      - "overwrite"
      - "delete"
      - "copy"
      - "move"
  
  - name: "editor"
    type: "editor"
    description: "编辑：读写权限"
    permissions:
      - "read"
      - "write"
      - "overwrite"
    conditions:
      path_prefix: ["/workspace/", "s3://workspace/"]
  
  - name: "backup-operator"
    type: "backup"
    description: "备份操作员"
    permissions:
      - "read"
      - "copy"

# ============================================
# 规则定义（显式规则，优先级高于角色）
# ============================================
policies:
  - name: "default"
    default_effect: "deny"
    enabled: true
    
    rules:
      # 规则1：显式允许 - 最高优先级
      - name: "allow-overwrite-temp"
        priority: 10
        enabled: true
        description: "临时目录允许覆盖"
        match:
          path_prefix: ["/tmp/", "s3://temp/"]
          operations: ["overwrite"]
        effect: "allow"
      
      # 规则2：显式拒绝 - 高优先级
      - name: "deny-delete-production"
        priority: 20
        enabled: true
        description: "禁止删除生产环境文件"
        match:
          provider: ["s3", "r2"]
          bucket: ["production-*", "prod-*"]
          operations: ["delete"]
        effect: "deny"
      
      # 规则3：引用角色
      - name: "role-based-access"
        priority: 100
        enabled: true
        description: "基于角色的访问控制"
        match:
          operations: ["read", "write", "overwrite", "delete", "copy", "move"]
        effect: "allow"
        role: "editor"  # 引用角色
      
      # 规则4：默认允许读取
      - name: "allow-read-default"
        priority: 1000
        enabled: true
        match:
          operations: ["read"]
        effect: "allow"
```

---

## 7. 评估流程

```
权限检查请求
      │
      ▼
┌─────────────────┐
│ 1. 规则引擎检查  │  ◀── 显式规则优先匹配
│   - 按优先级    │
│   - 匹配则返回  │
└────────┬────────┘
         │ 匹配到规则
         ▼ 返回决策
         │
         │ 未匹配
         ▼
┌─────────────────┐
│ 2. 角色权限检查  │  ◀── 回退到角色评估
│   - 解析角色    │
│   - 评估权限    │
└────────┬────────┘
         │ 有角色权限
         ▼ 返回决策
         │
         │ 无角色权限
         ▼
┌─────────────────┐
│ 3. 默认行为     │  ◀── 最终回退
│   - read: allow │
│   - 其他: deny  │
└─────────────────┘
```

---

## 8. 测试策略

### 8.1 单元测试

| 文件 | 测试内容 |
|------|----------|
| [`pkg/permission/engine_test.go`](pkg/permission/engine_test.go) | 规则匹配：路径前缀、正则、Provider、Bucket |
| [`pkg/permission/rbac_test.go`](pkg/permission/rbac_test.go) | **新增**：角色权限评估 |
| [`pkg/permission/config_test.go`](pkg/permission/config_test.go) | 配置解析（规则+角色） |
| [`pkg/permission/manager_test.go`](pkg/permission/manager_test.go) | 完整流程测试 |

### 8.2 测试用例设计

```go
// rbac_test.go 测试用例

// T1: 角色权限评估
func TestRole_AdminHasAllPermissions() {
    role := Role{Type: RoleTypeAdmin, Permissions: AllPermissions}
    assert.True(role.HasPermission(PermissionDelete))
    assert.True(role.HasPermission(PermissionMove))
}

func TestRole_Editor_ConditionalAccess() {
    role := Role{
        Type: RoleTypeEditor,
        Permissions: []Permission{PermissionRead, PermissionWrite},
        Conditions: &RoleConditions{
            PathPrefix: []string{"/workspace/"},
        },
    }
    // 有条件限制
    assert.True(role.HasPermissionWithContext(
        PermissionWrite, "/workspace/file.txt"))
    // 无条件限制
    assert.False(role.HasPermissionWithContext(
        PermissionWrite, "/etc/passwd"))
}

// T2: 规则优先于角色
func TestRule_OverridesRole() {
    // 规则明确拒绝
    rule := Rule{Priority: 10, Effect: EffectDeny, Match: ...}
    // 角色允许
    role := Role{Permissions: []Permission{PermissionDelete}}
    // 规则应优先
    result := evaluate(rule, role, defaultAllow)
    assert.Equal(EffectDeny, result)
}
```

---

## 9. 风险和缓解措施

### 9.1 高风险

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 规则与角色优先级混乱 | 权限判断不一致 | 明确优先级文档化，测试覆盖 |
| 角色条件评估复杂度 | 性能下降 | 缓存编译后的角色定义 |
| 接口变更破坏兼容性 | 现有 provider 报错 | 方法设为可选，类型断言检查 |

### 9.2 中风险

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 角色爆炸 | 配置复杂难维护 | 内置角色模板，限制自定义角色数量 |
| 循环引用 | 规则互相引用 | 检测并报错 |

### 9.3 低风险

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 测试覆盖不足 | 边界条件漏测 | 完整覆盖规则+角色交互 |

---

## 10. 实施检查清单

### 开发前
- [x] 确认混合方案设计
- [x] 定义角色模板
- [x] 明确优先级规则

### 开发中 (13 个文件)
- [x] Phase 0: types.go, rule.go, role.go, request.go, checker.go
- [x] Phase 1: engine.go, rbac.go
- [x] Phase 2: config.go, manager.go, errors.go
- [ ] Phase 3: provider.go 扩展
- [x] Phase 4: error.go, fs.go

### 测试中
- [x] engine_test.go（基础测试已在 permission_test.go 中）
- [x] rbac_test.go（基础测试已在 permission_test.go 中）
- [ ] config_test.go
- [ ] manager_test.go
- [ ] 集成测试

### 上线前
- [ ] 代码审查
- [ ] 文档更新
- [ ] 示例配置验证

---

## 11. 实施顺序总结

```
Phase 0: 基础类型 (2.5h)    → types, rule, role, request, checker  ✅ 已完成
Phase 1: 核心引擎 (3h)      → engine, rbac                         ✅ 已完成
Phase 2: 配置管理 (2.5h)    → config, manager, errors              ✅ 已完成
Phase 3: Provider 接口 (0.5h) → provider.go 扩展                   ⏳ 待实施
Phase 4: CLI 集成 (1.5h)    → error.go, fs.go                      ✅ 已完成
Phase 5: 初始化 (1h)        → root.go, config.go                   ✅ 已完成（root.go）

总计: 约 11 小时
```

---

## 12. 实施进度追踪

> **实施时间**：2026-04
> **最后更新**：2026-04-15

### 12.1 阶段完成状态

| 阶段 | 名称 | 状态 | 完成时间 |
|------|------|------|----------|
| 阶段 0 | 基础类型定义 | ✅ 已完成 | 2026-04 |
| 阶段 1 | RBAC 核心 | ✅ 已完成 | 2026-04 |
| 阶段 2 | 规则引擎 | ✅ 已完成 | 2026-04 |
| 阶段 3 | Manager 集成 | ✅ 已完成 | 2026-04 |
| 阶段 4 | 命令行集成 | ✅ 已完成 | 2026-04 |
| 阶段 5 | 测试验证 | ⏳ 部分完成 | 进行中 |

### 12.2 文件创建状态

#### 新建文件（pkg/permission/）

| 文件 | 状态 | 备注 |
|------|------|------|
| [`types.go`](pkg/permission/types.go) | ✅ 已完成 | Permission、Effect、RoleType 类型定义 |
| [`rule.go`](pkg/permission/rule.go) | ✅ 已完成 | Rule、Policy、RuleMatchConditions 结构 |
| [`role.go`](pkg/permission/role.go) | ✅ 已完成 | Role、RoleBinding、RoleConditions 结构 |
| [`request.go`](pkg/permission/request.go) | ✅ 已完成 | Request、Result 结构 |
| [`checker.go`](pkg/permission/checker.go) | ✅ 已完成 | PermissionChecker 接口 |
| [`engine.go`](pkg/permission/engine.go) | ✅ 已完成 | 规则匹配引擎 |
| [`rbac.go`](pkg/permission/rbac.go) | ✅ 已完成 | 角色权限评估器 |
| [`config.go`](pkg/permission/config.go) | ✅ 已完成 | YAML 配置加载器 |
| [`manager.go`](pkg/permission/manager.go) | ✅ 已完成 | 权限管理器主类 |
| [`evaluator.go`](pkg/permission/evaluator.go) | ✅ 已完成 | Evaluator 接口和实现（新增） |
| [`errors.go`](pkg/permission/errors.go) | ✅ 已完成 | 权限错误定义 |

#### 修改文件

| 文件 | 状态 | 修改内容 |
|------|------|----------|
| [`cmd/fs.go`](cmd/fs.go) | ✅ 已完成 | 集成权限检查调用 |
| [`cmd/root.go`](cmd/root.go) | ✅ 已完成 | 添加配置加载初始化 |
| [`pkg/apperr/error.go`](pkg/apperr/error.go) | ✅ 已完成 | 新增权限错误码 |

### 12.3 测试覆盖

| 测试文件 | 状态 | 覆盖内容 |
|----------|------|----------|
| [`permission_test.go`](pkg/permission/permission_test.go) | ✅ 已完成 | 基础单元测试 |
| engine_test.go | ⏳ 待补充 | 规则匹配详细测试 |
| rbac_test.go | ⏳ 待补充 | 角色权限评估详细测试 |
| config_test.go | ⏳ 待补充 | 配置解析测试 |
| manager_test.go | ⏳ 待补充 | 完整流程测试 |

### 12.4 待完成项

1. **Provider 接口扩展**：`pkg/provider/provider.go` 添加 `CheckPermission` 方法
2. **配置文件支持**：`pkg/config/config.go` 支持权限配置加载
3. **完善测试**：补充 engine、rbac、config、manager 的详细测试
4. **集成测试**：端到端权限控制测试
5. **文档更新**：用户文档和配置示例

---

*本文档为混合方案实施计划，融合规则模型和 RBAC 角色系统*