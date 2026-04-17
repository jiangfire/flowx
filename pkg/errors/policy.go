package errors

import (
	"fmt"
	"net/http"

	datagovapp "git.neolidy.top/neo/flowx/internal/application/datagov"
)

// PolicyViolationErrorCode 策略校验失败的业务错误码
const PolicyViolationErrorCode = 42201

// PolicyViolationError 策略校验失败错误
type PolicyViolationError struct {
	Code       int                         `json:"code"`
	Violations []datagovapp.PolicyViolation `json:"data"`
}

// Error 实现 error 接口
func (e *PolicyViolationError) Error() string {
	if len(e.Violations) == 0 {
		return "策略校验失败"
	}
	return fmt.Sprintf("策略校验失败: %s", e.Violations[0].Message)
}

// StatusCode 返回 HTTP 状态码
func (e *PolicyViolationError) StatusCode() int {
	return http.StatusUnprocessableEntity
}

// BusinessCode 返回业务错误码
func (e *PolicyViolationError) BusinessCode() int {
	return e.Code
}
