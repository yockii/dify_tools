package constant

import (
	"errors"
	"net/http"
)

// 自定义错误
var (
	// 通用错误
	ErrInternalError     = errors.New("内部错误")
	ErrInvalidParams     = errors.New("参数错误")
	ErrUnauthorized      = errors.New("未授权")
	ErrForbidden         = errors.New("禁止访问")
	ErrDatabaseError     = errors.New("数据库错误")
	ErrInvalidToken      = errors.New("无效的token")
	ErrInvalidOperation  = errors.New("无效的操作")
	ErrPermissionDenied  = errors.New("权限不足")
	ErrInvalidCredential = errors.New("凭证错误")
	ErrTokenExpired      = errors.New("token已过期")
	ErrRecordDuplicate   = errors.New("记录重复")
	ErrRecordNotFound    = errors.New("记录不存在")
	ErrRecordIDEmpty     = errors.New("ID不能为空")
	ErrSerializeError    = errors.New("序列化错误")
	ErrDeserializeError  = errors.New("反序列化错误")
	ErrCacheError        = errors.New("缓存错误")

	// 角色相关错误
	ErrRoleInUse = errors.New("角色正在使用")

	// 字典错误
	ErrDictNotConfigured = errors.New("字典未配置")
)

// 获取错误对应的HTTP状态码
func GetErrorCode(err error) int {
	switch err {
	// 通用错误
	case ErrInternalError:
		return http.StatusInternalServerError
	case ErrInvalidParams:
		return http.StatusBadRequest
	case ErrUnauthorized:
		return http.StatusUnauthorized
	case ErrForbidden:
		return http.StatusForbidden
	case ErrDatabaseError:
		return http.StatusInternalServerError
	case ErrInvalidToken:
		return http.StatusUnauthorized
	case ErrInvalidOperation:
		return http.StatusBadRequest
	case ErrPermissionDenied:
		return http.StatusForbidden
	case ErrInvalidCredential:
		return http.StatusUnauthorized
	case ErrTokenExpired:
		return http.StatusUnauthorized
	case ErrRecordDuplicate:
		return http.StatusBadRequest
	case ErrRecordNotFound:
		return http.StatusNotFound
	case ErrRecordIDEmpty:
		return http.StatusBadRequest
	case ErrSerializeError:
		return http.StatusInternalServerError
	case ErrDeserializeError:
		return http.StatusInternalServerError
	case ErrCacheError:
		return http.StatusInternalServerError

	// 角色相关错误
	case ErrRoleInUse:
		return http.StatusBadRequest

	// 字典相关错误
	case ErrDictNotConfigured:
		return http.StatusInternalServerError

	default:
		return http.StatusInternalServerError
	}
}
