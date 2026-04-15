package permission

import (
	"context"
	"time"
)

// Evaluator 统一权限评估接口
// 实现要求：
// 1) 遵循规则优先级与 Effect 语义（Deny 优先于 Allow）
// 2) 当规则引用角色时，需进一步校验角色权限与约束（复用 Role.HasPermissionWithContext）
// 3) 保持与现有 Engine 行为等价（在未接入索引主路径时，可作为独立单元测试目标）
type Evaluator interface {
	// Evaluate 对给定请求在候选规则集上进行评估，并结合角色映射作出最终决策
	// 参数:
	//   ctx   - 上下文（预留取消与超时控制）
	//   req   - 权限请求
	//   rules - 候选规则集合（已按优先级稳定排序）
	//   roles - 角色名到角色指针的映射（供角色引用时快速定位）
	// 返回:
	//   Result - 评估结果（包含 Allowed、MatchedRule、Reason 等信息）
	Evaluate(ctx context.Context, req Request, rules []Rule, roles map[string]*Role) Result
}

// MetricsSink 可观测性接口（空实现可直接使用 NoopMetrics）
// 建议在 Engine 接入时由配置注入具体实现（日志/指标系统等）
type MetricsSink interface {
	// ObserveIndexHits 记录索引命中情况（例如 provider/bucket/prefix 的命中标签与命中数）
	ObserveIndexHits(labels map[string]string, hits int)
	// ObserveCandidateCount 记录候选集规模（用于评估索引收敛效果）
	ObserveCandidateCount(labels map[string]string, n int)
	// ObserveEvalDuration 记录单次评估耗时
	ObserveEvalDuration(labels map[string]string, d time.Duration)
}

// noopMetricsSink 默认空实现，避免对使用方造成侵入
type noopMetricsSink struct{}

func (n *noopMetricsSink) ObserveIndexHits(_ map[string]string, _ int)        {}
func (n *noopMetricsSink) ObserveCandidateCount(_ map[string]string, _ int)   {}
func (n *noopMetricsSink) ObserveEvalDuration(_ map[string]string, _ time.Duration) {
}

// NoopMetrics 全局可复用的空指标实例
var NoopMetrics MetricsSink = &noopMetricsSink{}

// EvalOptions Evaluator 可选项
type EvalOptions struct {
	// Metrics 指标上报接口（默认 NoopMetrics）
	Metrics MetricsSink
}

// EvalOption 选项函数
type EvalOption func(*EvalOptions)

// WithMetrics 注入自定义指标上报接口
func WithMetrics(sink MetricsSink) EvalOption {
	return func(o *EvalOptions) {
		if sink != nil {
			o.Metrics = sink
		}
	}
}

// BaseEvaluator 提供通用的选项管理与指标上报能力，具体评估逻辑由子类实现
type BaseEvaluator struct {
	opts EvalOptions
}

// NewBaseEvaluator 创建带可选项的基础评估器
func NewBaseEvaluator(opts ...EvalOption) BaseEvaluator {
	eo := EvalOptions{
		Metrics: NoopMetrics,
	}
	for _, fn := range opts {
		fn(&eo)
	}
	return BaseEvaluator{opts: eo}
}

// Metrics 返回指标接口（便于子类上报）
func (b *BaseEvaluator) Metrics() MetricsSink {
	if b.opts.Metrics == nil {
		return NoopMetrics
	}
	return b.opts.Metrics
}

// RuleMatchFunc 规则匹配回调，由调用方（如 Engine）注入具体匹配逻辑
// 该回调应保证与现有 Engine.matchRule 语义一致（路径、正则、通配符、大小等）
type RuleMatchFunc func(rule Rule, req Request) bool

// RuleEvaluator 默认规则评估器：在候选规则集上评估并应用角色引用
type RuleEvaluator struct {
	BaseEvaluator
	match RuleMatchFunc
}

// NewRuleEvaluator 创建规则评估器
// match 不能为空；建议由 Engine 传入其内部的匹配逻辑，以保持行为等价
func NewRuleEvaluator(match RuleMatchFunc, opts ...EvalOption) *RuleEvaluator {
	be := NewBaseEvaluator(opts...)
	return &RuleEvaluator{
		BaseEvaluator: be,
		match:         match,
	}
}

// Evaluate 遍历候选规则（已按优先级排序），按 Effect 与角色引用作出决策
func (ev *RuleEvaluator) Evaluate(ctx context.Context, req Request, rules []Rule, roles map[string]*Role) Result {
	start := time.Now()
	m := ev.Metrics()
	if m != nil {
		m.ObserveCandidateCount(map[string]string{"phase": "rule"}, len(rules))
	}
	defer func() {
		if m != nil {
			m.ObserveEvalDuration(map[string]string{"phase": "rule"}, time.Since(start))
		}
	}()

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		// 操作类型预筛
		if !rule.MatchOperation(req.Operation) {
			continue
		}
		// 具体匹配逻辑由注入的 match 回调决定（与 Engine.matchRule 等价）
		if ev.match == nil || !ev.match(rule, req) {
			continue
		}

		// Deny 优先
		if rule.Effect == EffectDeny {
			return NewResultDenied(rule.Name, "denied by rule: "+rule.Description)
		}

		// Allow 且存在角色引用 -> 继续校验角色约束
		if rule.HasRoleReference() {
			roleName := *rule.Role
			role, ok := roles[roleName]
			if !ok {
				return NewResultDenied(rule.Name, "role not found: "+roleName)
			}
			if role.HasPermissionWithContext(req.Operation, req.TargetPath, req.Provider, req.Bucket) {
				res := NewResultAllowed("allowed by rule with role: " + rule.Description)
				res.MatchedRule = rule.Name
				return res
			}
			// 角色未满足，继续尝试下一条规则
			continue
		}

		// 纯规则允许
		res := NewResultAllowed("allowed by rule: " + rule.Description)
		res.MatchedRule = rule.Name
		return res
	}

	// 未命中任何规则：返回拒绝占位，由上层（Engine）决定是否继续走角色或默认策略
	return NewResultDenied("", "no matching rule")
}

// RoleEvaluator 角色评估器：仅基于请求携带的角色进行授权判断
type RoleEvaluator struct {
	BaseEvaluator
}

// NewRoleEvaluator 创建角色评估器
func NewRoleEvaluator(opts ...EvalOption) *RoleEvaluator {
	return &RoleEvaluator{BaseEvaluator: NewBaseEvaluator(opts...)}
}

// Evaluate 忽略规则集，按请求的 RequestedRoles 对角色映射进行授权判定
func (ev *RoleEvaluator) Evaluate(ctx context.Context, req Request, _ []Rule, roles map[string]*Role) Result {
	start := time.Now()
	m := ev.Metrics()
	defer func() {
		if m != nil {
			m.ObserveEvalDuration(map[string]string{"phase": "role"}, time.Since(start))
		}
	}()

	for _, roleName := range req.RequestedRoles {
		role, ok := roles[roleName]
		if !ok {
			continue
		}
		if role.HasPermissionWithContext(req.Operation, req.TargetPath, req.Provider, req.Bucket) {
			return NewResultAllowed("allowed by role: " + role.Name)
		}
	}

	// 未找到匹配角色：返回拒绝占位，由上层（Engine）决定后续行为（如默认策略）
	return NewResultDenied("", "no matching role found")
}