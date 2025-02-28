package service

import (
	"context"
	"mime/multipart"
	"net/http"

	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/dify"
	"github.com/yockii/dify_tools/internal/model"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type BaseService[T model.Model] interface {
	Create(ctx context.Context, record T) error
	Update(ctx context.Context, record T) error
	Delete(ctx context.Context, id uint64) error
	Get(ctx context.Context, id uint64) (T, error)
	List(ctx context.Context, condition T, offset, limit int) ([]T, int64, error)
}

type DictService interface {
	BaseService[*model.Dict]
	GetByCode(ctx context.Context, code string) (*model.Dict, error)
}

type UserService interface {
	BaseService[*model.User]
	UpdatePassword(ctx context.Context, id uint64, oldPassword, newPassword string) error
	UpdateStatus(ctx context.Context, id uint64, status int) error
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
}

type AuthService interface {
	Login(ctx context.Context, username, password string) (uint64, string, error)
	Refresh(ctx context.Context, token string) (string, error)
	Logout(ctx context.Context, token string) error
	Verify(ctx context.Context, token string) (*model.User, error)
}

type RoleService interface {
	BaseService[*model.Role]
}

type LogService interface {
	CreateLoginLog(ctx context.Context, uid uint64, ip, userAgent string, success bool) error
	CreateOperationLog(ctx context.Context, uid uint64, action int, ip, userAgent string) error
	ListLogs(ctx context.Context, uid uint64, actions []int, offset, limit int) ([]*model.Log, int64, error)
	CreateLog(ctx context.Context, log *model.Log) error
	DeleteOldLogs(ctx context.Context, days int) error
	BatchCreateLogs(ctx context.Context, logs []*model.Log) error
	GetUserLoginHistory(ctx context.Context, userID uint64, limit int) ([]*model.Log, error)
	GetUserLastLogin(ctx context.Context, userID uint64) (*model.Log, error)
}

type SessionService interface {
	// CreateSession 创建会话
	CreateSession(ctx context.Context, user *model.User) (string, error)
	// GetSession 获取会话
	GetSession(ctx context.Context, sessionID string) (*model.User, error)
	// RefreshSession 刷新会话
	RefreshSession(ctx context.Context, sessionID string) error
	// DeleteSession 删除会话
	DeleteSession(ctx context.Context, sessionID string) error
	// BlockToken 将token加入黑名单
	BlockToken(ctx context.Context, token string) error
	// IsTokenBlocked 检查token是否在黑名单中
	IsTokenBlocked(ctx context.Context, token string) bool
	// CachePermissions 缓存用户权限
	CachePermissions(ctx context.Context, userID uint64, permissions []*model.Permission) error
	// GetCachedPermissions 获取缓存的权限
	GetCachedPermissions(ctx context.Context, userID uint64) ([]*model.Permission, error)
}

type ApplicationService interface {
	BaseService[*model.Application]
	GetByApiKey(ctx context.Context, apiKey string) (*model.Application, error)
}

type DataSourceService interface {
	BaseService[*model.DataSource]
	Sync(ctx context.Context, id uint64) error
}

type TableInfoService interface {
	BaseService[*model.TableInfo]
}

type ColumnInfoService interface {
	BaseService[*model.ColumnInfo]
}

type KnowledgeBaseService interface {
	BaseService[*model.KnowledgeBase]
	GetDifyKnowledgeBaseClient(ctx context.Context) (*dify.KnowledgeBaseClient, error)
	DeleteByApplicationID(ctx context.Context, applicationID uint64) error
	GetByApplicationIDAndCustomID(ctx context.Context, applicationID uint64, customID string) (*model.KnowledgeBase, error)
}

type DocumentService interface {
	BaseService[*model.Document]
	AddDocument(ctx context.Context, document *model.Document, fileHeader *multipart.FileHeader) (*model.KnowledgeBase, error)
}

// /////////////////////////////
// Response 通用响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func OK(data interface{}) *Response {
	return NewResponse(data, nil)
}

func Error(err error) *Response {
	return NewResponse(nil, err)
}

// NewResponse 创建响应
func NewResponse(data interface{}, err error, msg ...string) *Response {
	message := "success"
	if len(msg) > 0 {
		message = msg[0]
	}
	if err == nil {
		return &Response{
			Code:    http.StatusOK,
			Message: message,
			Data:    data,
		}
	}

	code := constant.GetErrorCode(err)
	return &Response{
		Code:    code,
		Message: err.Error(),
		Data:    data,
	}
}

// ListResponse 列表响应结构
type ListResponse struct {
	Total  int64       `json:"total"`
	Items  interface{} `json:"items"`
	Offset int         `json:"offset"`
	Limit  int         `json:"limit"`
}

// NewListResponse 创建列表响应
func NewListResponse(items interface{}, total int64, offset, limit int) *ListResponse {
	return &ListResponse{
		Total:  total,
		Items:  items,
		Offset: offset,
		Limit:  limit,
	}
}
