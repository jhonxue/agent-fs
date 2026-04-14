package apperr

import "errors"

const (
	CodeInvalidArg    = `ERR_INVALID_ARGUMENT`
	CodePathTraversal = `ERR_PATH_TRAVERSAL`
	CodeNotFound      = `ERR_NOT_FOUND`
	CodeConflict      = `ERR_CONFLICT`
	CodeProvider      = `ERR_PROVIDER`
	CodeUpload        = `ERR_UPLOAD`
	CodeDownload      = `ERR_DOWNLOAD`
	CodeArchive       = `ERR_ARCHIVE`
	CodeConfig        = `ERR_CONFIG`
	CodeInternal      = `ERR_INTERNAL`
	CodePermission    = `ERR_PERMISSION`  // 新增：权限错误
)

type Error struct {
	Action  string
	Code    string
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e == nil {
		return ``
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func New(action, code, message string) error {
	return &Error{
		Action:  action,
		Code:    code,
		Message: message,
	}
}

func Wrap(action, code, message string, cause error) error {
	return &Error{
		Action:  action,
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

func Parse(err error, defaultAction string) (action, code, message string) {
	if err == nil {
		return defaultAction, CodeInternal, `unknown error`
	}
	var appErr *Error
	if errors.As(err, &appErr) {
		act := appErr.Action
		if act == `` {
			act = defaultAction
		}
		c := appErr.Code
		if c == `` {
			c = CodeInternal
		}
		msg := appErr.Message
		if msg == `` {
			msg = err.Error()
		}
		return act, c, msg
	}
	return defaultAction, CodeInternal, err.Error()
}
