package service

import (
	"errors"
	"net/http"
)

// 自定义错误
var (
	// 通用错误
	ErrInvalidParams     = errors.New("参数错误")
	ErrServerError       = errors.New("服务器内部错误")
	ErrUnauthorized      = errors.New("未授权")
	ErrForbidden         = errors.New("禁止访问")
	ErrUserExists        = errors.New("用户已存在")
	ErrDatabaseError     = errors.New("数据库错误")
	ErrInvalidToken      = errors.New("无效的token")
	ErrInvalidOperation  = errors.New("无效的操作")
	ErrPermissionDenied  = errors.New("权限不足")
	ErrInvalidCredential = errors.New("凭证错误")
	// 用户相关错误
	ErrUserNotFound    = errors.New("用户不存在")
	ErrInvalidPassword = errors.New("密码错误")
	ErrTokenExpired    = errors.New("token已过期")
	ErrUserDisabled    = errors.New("用户已禁用")

	// 角色相关错误
	ErrRoleNotFound = errors.New("角色不存在")
	ErrRoleExists   = errors.New("角色已存在")
	ErrRoleInUse    = errors.New("角色正在使用")
	ErrInvalidRole  = errors.New("无效的角色")
)

// 获取错误对应的HTTP状态码
func GetErrorCode(err error) int {
	switch err {
	case ErrInvalidParams:
		return http.StatusBadRequest
	case ErrUserNotFound:
		return http.StatusNotFound
	case ErrServerError:
		return http.StatusInternalServerError
	case ErrUnauthorized:
		return http.StatusUnauthorized
	case ErrForbidden:
		return http.StatusForbidden
	case ErrUserExists:
		return http.StatusBadRequest
	case ErrInvalidPassword:
		return http.StatusBadRequest
	case ErrInvalidToken:
		return http.StatusUnauthorized
	case ErrTokenExpired:
		return http.StatusUnauthorized
	case ErrUserDisabled:
		return http.StatusForbidden
	case ErrInvalidCredential:
		return http.StatusUnauthorized
	case ErrDatabaseError:
		return http.StatusInternalServerError
	case ErrInvalidOperation:
		return http.StatusBadRequest
	case ErrRoleNotFound:
		return http.StatusNotFound
	case ErrRoleExists:
		return http.StatusBadRequest
	case ErrRoleInUse:
		return http.StatusBadRequest
	case ErrInvalidRole:
		return http.StatusBadRequest
	case ErrPermissionDenied:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}
