package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/pkg/config"
	"github.com/yockii/dify_tools/pkg/logger"
)

const (
	// key前缀
	sessionPrefix = "session:"
	tokenPrefix   = "token:"
	permPrefix    = "perm:"

	// 过期时间
	permExpire = time.Hour // 权限缓存有效期
)

type sessionService struct {
	rdb           *redis.Client
	sessionExpire time.Duration
	tokenExpire   time.Duration
}

func (s *sessionService) CreateSession(ctx context.Context, user *model.User) (string, error) {
	sessionID := uuid.New().String()

	// 序列化用户信息
	userData, err := json.Marshal(user)
	if err != nil {
		logger.Error("序列化用户信息失败", logger.F("error", err))
		return "", constant.ErrSerializeError
	}

	// 存储会话信息
	key := sessionPrefix + sessionID
	if err := s.rdb.Set(ctx, key, userData, s.sessionExpire).Err(); err != nil {
		logger.Error("存储会话信息失败", logger.F("error", err))
		return "", constant.ErrCacheError
	}

	return sessionID, nil
}

func (s *sessionService) GetSession(ctx context.Context, sessionID string) (*model.User, error) {
	// 获取会话信息
	key := sessionPrefix + sessionID
	data, err := s.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, constant.ErrRecordNotFound
		}
		logger.Error("获取会话信息失败", logger.F("error", err))
		return nil, constant.ErrCacheError
	}

	// 反序列化用户信息
	var user model.User
	if err := json.Unmarshal(data, &user); err != nil {
		logger.Error("反序列化用户信息失败", logger.F("error", err))
		return nil, constant.ErrDeserializeError
	}

	return &user, nil
}

func (s *sessionService) RefreshSession(ctx context.Context, sessionID string) error {
	key := sessionPrefix + sessionID
	if err := s.rdb.Expire(ctx, key, s.sessionExpire).Err(); err != nil {
		logger.Error("刷新会话信息失败", logger.F("error", err))
		return constant.ErrCacheError
	}
	return nil
}

func (s *sessionService) DeleteSession(ctx context.Context, sessionID string) error {
	key := sessionPrefix + sessionID
	if err := s.rdb.Del(ctx, key).Err(); err != nil {
		logger.Error("删除会话信息失败", logger.F("error", err))
		return constant.ErrCacheError
	}
	return nil
}

func (s *sessionService) BlockToken(ctx context.Context, token string) error {
	key := tokenPrefix + token
	if err := s.rdb.Set(ctx, key, "blocked", s.tokenExpire).Err(); err != nil {
		logger.Error("禁用token失败", logger.F("error", err))
		return constant.ErrCacheError
	}
	return nil
}

func (s *sessionService) IsTokenBlocked(ctx context.Context, token string) bool {
	key := tokenPrefix + token
	_, err := s.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false
		}
		logger.Error("查询token失败", logger.F("error", err))
		return true
	}
	return true
}

func (s *sessionService) CachePermissions(ctx context.Context, userID uint64, permissions []*model.Permission) error {
	// 序列化权限信息
	data, err := json.Marshal(permissions)
	if err != nil {
		logger.Error("序列化权限信息失败", logger.F("error", err))
		return constant.ErrSerializeError
	}

	// 存储权限缓存
	key := fmt.Sprintf("%s%d", permPrefix, userID)
	if err := s.rdb.Set(ctx, key, data, permExpire).Err(); err != nil {
		logger.Error("存储权限信息失败", logger.F("error", err))
		return constant.ErrCacheError
	}
	return nil
}

func (s *sessionService) GetCachedPermissions(ctx context.Context, userID uint64) ([]*model.Permission, error) {
	key := fmt.Sprintf("%s%d", permPrefix, userID)
	data, err := s.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		logger.Error("获取缓存权限失败", logger.F("error", err))
		return nil, constant.ErrCacheError
	}

	var permissions []*model.Permission
	if err := json.Unmarshal(data, &permissions); err != nil {
		logger.Error("反序列化权限失败", logger.F("error", err))
		return nil, constant.ErrDeserializeError
	}

	return permissions, nil
}

func NewSessionService() SessionService {
	return &sessionService{
		rdb: redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("%s:%d",
				config.GetString("cache.redis.host"),
				config.GetInt("cache.redis.port")),
			Password:     config.GetString("cache.redis.password"),
			DB:           config.GetInt("cache.redis.db"),
			PoolSize:     config.GetInt("cache.redis.pool_size"),
			MinIdleConns: config.GetInt("cache.redis.pool_size") / 2,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		}),
		sessionExpire: time.Duration(config.GetInt64("security.session_timeout")) * time.Second,
		tokenExpire:   time.Duration(config.GetInt64("security.token_timeout")) * time.Second,
	}
}
