package service

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yockii/dify_tools/internal/constant"
	"github.com/yockii/dify_tools/internal/model"
	"github.com/yockii/dify_tools/pkg/config"
	"github.com/yockii/dify_tools/pkg/logger"
)

type Claims struct {
	UserID uint64 `json:"uid"`
	jwt.RegisteredClaims
}

type authService struct {
	userService    UserService
	sessionService SessionService
	secret         []byte
	expire         time.Duration
}

// Login implements AuthService.
func (s *authService) Login(ctx context.Context, username string, password string) (uint64, string, error) {
	// 验证用户名密码
	user, err := s.userService.GetUserByUsername(ctx, username)
	if err != nil {
		return 0, "", err
	}

	if !user.ComparePassword(password) {
		return 0, "", constant.ErrInvalidCredential
	}

	// 生成token
	claims := Claims{
		UserID: user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.expire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.secret)
	if err != nil {
		logger.Error("生成token失败", logger.F("error", err))
		return 0, "", constant.ErrSerializeError
	}

	// 更新最后登录时间
	user.LastLogin = time.Now()
	if err := s.userService.Update(ctx, user); err != nil {
		return 0, "", err
	}

	// 创建会话
	if s.sessionService != nil {
		if _, err := s.sessionService.CreateSession(ctx, user); err != nil {
			return 0, "", err
		}
	}

	return user.ID, signedToken, nil
}

func (s *authService) Verify(ctx context.Context, tokenString string) (*model.User, error) {
	// 检查token是否在黑名单中
	if s.sessionService != nil && s.sessionService.IsTokenBlocked(ctx, tokenString) {
		logger.Warn("token被禁止", logger.F("token", tokenString))
		return nil, constant.ErrInvalidToken
	}

	// 解析token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			logger.Warn("非预期的token解析方式", logger.F("alg", token.Header["alg"]))
			return nil, constant.ErrInvalidToken
		}
		return s.secret, nil
	})

	if err != nil {
		logger.Warn("无效token", logger.F("error", err))
		return nil, constant.ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, constant.ErrInvalidToken
	}

	// 获取用户信息
	user, err := s.userService.Get(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	user.Password = ""

	return user, nil
}

func (s *authService) Refresh(ctx context.Context, tokenString string) (string, error) {
	user, err := s.Verify(ctx, tokenString)
	if err != nil {
		return "", err
	}

	// 生成新token
	claims := Claims{
		UserID: user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.expire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *authService) Logout(ctx context.Context, tokenString string) error {
	// 验证token有效性
	if _, err := s.Verify(ctx, tokenString); err != nil {
		return err
	}

	// 将token加入黑名单
	if s.sessionService != nil {
		if err := s.sessionService.BlockToken(ctx, tokenString); err != nil {
			return err
		}
	}

	return nil
}

func NewAuthService(
	userService UserService,
	sessionService SessionService,
) AuthService {
	return &authService{
		userService:    userService,
		sessionService: sessionService,
		secret:         []byte(config.GetJWTSecret()),
		expire:         time.Duration(config.GetInt("jwt.expire")) * time.Second,
	}
}
