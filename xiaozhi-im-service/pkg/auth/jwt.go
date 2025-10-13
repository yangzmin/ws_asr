package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT声明结构
type Claims struct {
	UserID   string `json:"user_id"`
	DeviceID string `json:"device_id"`
	ClientID string `json:"client_id"`
	jwt.RegisteredClaims
}

// JWTManager JWT管理器
type JWTManager struct {
	secret []byte
}

// NewJWTManager 创建JWT管理器
func NewJWTManager(secret string) *JWTManager {
	return &JWTManager{
		secret: []byte(secret),
	}
}

// ValidateToken 验证JWT Token
func (j *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	// 移除Bearer前缀
	if strings.HasPrefix(tokenString, "Bearer ") {
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	}

	// 解析token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("token解析失败: %v", err)
	}

	// 验证token有效性
	if !token.Valid {
		return nil, errors.New("token无效")
	}

	// 获取claims
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("无法获取token声明")
	}

	// 验证过期时间
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, errors.New("token已过期")
	}

	return claims, nil
}

// GenerateToken 生成JWT Token（用于测试）
func (j *JWTManager) GenerateToken(userID, deviceID, clientID string, duration time.Duration) (string, error) {
	claims := &Claims{
		UserID:   userID,
		DeviceID: deviceID,
		ClientID: clientID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secret)
}