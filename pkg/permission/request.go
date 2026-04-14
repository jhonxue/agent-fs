package permission

// Request 包含需要检查的操作请求
type Request struct {
	// 操作类型
	Operation Permission
	// 源路径（对于 copy/move）
	SourcePath string
	// 目标路径（对于 copy/move/overwrite）
	TargetPath string
	// Provider scheme
	Provider string
	// Bucket（云存储）
	Bucket string
	// 文件大小
	FileSize int64
	// 文件扩展名
	Extension string

	// 角色上下文
	// 请求方声明的角色
	RequestedRoles []string
	// 当前用户（可选）
	CurrentUser string
}

// Result 权限检查结果
type Result struct {
	// 是否允许操作
	Allowed bool
	// 匹配的规则名称
	MatchedRule string
	// 拒绝原因
	Reason string
	// 错误信息
	Error error
}

// NewResult 创建允许的结果
func NewResultAllowed(reason string) Result {
	return Result{
		Allowed: true,
		Reason:  reason,
	}
}

// NewResultDenied 创建拒绝的结果
func NewResultDenied(matchedRule, reason string) Result {
	return Result{
		Allowed:    false,
		MatchedRule: matchedRule,
		Reason:     reason,
	}
}

// NewResultError 创建错误结果
func NewResultError(err error) Result {
	return Result{
		Allowed: false,
		Error:   err,
	}
}

// NewRequest 创建权限检查请求
func NewRequest(op Permission, sourcePath, targetPath string) Request {
	return Request{
		Operation:   op,
		SourcePath:  sourcePath,
		TargetPath:  targetPath,
	}
}

// WithProvider 设置 Provider
func (r *Request) WithProvider(provider string) *Request {
	r.Provider = provider
	return r
}

// WithBucket 设置 Bucket
func (r *Request) WithBucket(bucket string) *Request {
	r.Bucket = bucket
	return r
}

// WithFileInfo 设置文件信息
func (r *Request) WithFileInfo(size int64, ext string) *Request {
	r.FileSize = size
	r.Extension = ext
	return r
}

// WithRoles 设置请求的角色
func (r *Request) WithRoles(roles ...string) *Request {
	r.RequestedRoles = roles
	return r
}

// WithUser 设置当前用户
func (r *Request) WithUser(user string) *Request {
	r.CurrentUser = user
	return r
}