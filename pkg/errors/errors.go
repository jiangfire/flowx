package errors

// BizError 业务错误
type BizError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error 实现error接口
func (e *BizError) Error() string {
	return e.Message
}

// 预定义业务错误
var (
	ErrInternal      = &BizError{Code: "INTERNAL_ERROR", Message: "内部服务器错误"}
	ErrNotFound      = &BizError{Code: "NOT_FOUND", Message: "资源不存在"}
	ErrUnauthorized  = &BizError{Code: "UNAUTHORIZED", Message: "未认证"}
	ErrForbidden     = &BizError{Code: "FORBIDDEN", Message: "无权限"}
	ErrBadRequest    = &BizError{Code: "BAD_REQUEST", Message: "请求参数错误"}
	ErrTenantMismatch = &BizError{Code: "TENANT_MISMATCH", Message: "租户不匹配"}
)
