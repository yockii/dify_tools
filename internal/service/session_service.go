package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/pkg/config"
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
		return "", fmt.Errorf("序列化用户信息失败: %v", err)
	}

	// 存储会话信息
	key := sessionPrefix + sessionID
	if err := s.rdb.Set(ctx, key, userData, s.sessionExpire).Err(); err != nil {
		return "", fmt.Errorf("存储会话信息失败: %v", err)
	}

	return sessionID, nil
}

func (s *sessionService) GetSession(ctx context.Context, sessionID string) (*model.User, error) {
	// 获取会话信息
	key := sessionPrefix + sessionID
	data, err := s.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("未找到会话信息")
		}
		return nil, fmt.Errorf("获取会话信息失败: %v", err)
	}

	// 反序列化用户信息
	var user model.User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("反序列化用户信息失败: %v", err)
	}

	return &user, nil
}

func (s *sessionService) RefreshSession(ctx context.Context, sessionID string) error {
	key := sessionPrefix + sessionID
	return s.rdb.Expire(ctx, key, s.sessionExpire).Err()
}

func (s *sessionService) DeleteSession(ctx context.Context, sessionID string) error {
	key := sessionPrefix + sessionID
	return s.rdb.Del(ctx, key).Err()
}

func (s *sessionService) BlockToken(ctx context.Context, token string) error {
	key := tokenPrefix + token
	return s.rdb.Set(ctx, key, "blocked", s.tokenExpire).Err()
}

func (s *sessionService) IsTokenBlocked(ctx context.Context, token string) bool {
	key := tokenPrefix + token
	_, err := s.rdb.Get(ctx, key).Result()
	return err == nil
}

func (s *sessionService) CachePermissions(ctx context.Context, userID uint64, permissions []*model.Permission) error {
	// 序列化权限信息
	data, err := json.Marshal(permissions)
	if err != nil {
		return fmt.Errorf("序列化权限信息失败: %v", err)
	}

	// 存储权限缓存
	key := fmt.Sprintf("%s%d", permPrefix, userID)
	return s.rdb.Set(ctx, key, data, permExpire).Err()
}

func (s *sessionService) GetCachedPermissions(ctx context.Context, userID uint64) ([]*model.Permission, error) {
	key := fmt.Sprintf("%s%d", permPrefix, userID)
	data, err := s.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("获取缓存权限失败: %v", err)
	}

	var permissions []*model.Permission
	if err := json.Unmarshal(data, &permissions); err != nil {
		return nil, fmt.Errorf("反序列化权限失败: %v", err)
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
