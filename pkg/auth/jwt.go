/*
 * @Descripttion:
 * @version:
 * @Author: hujianghong
 * @Date: 2025-04-22 21:59:17
 * @LastEditors: hujianghong
 * @LastEditTime: 2025-04-22 21:59:20
 */
package auth

import (
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
)

type JWTClaims struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
	jwt.StandardClaims
}

// JWTManager 负责JWT的生成和解析
type JWTManager struct {
	secretKey     string
	tokenDuration time.Duration
}

// NewJWTManager 创建新的JWT管理器
func NewJWTManager(secretKey string, tokenDuration time.Duration) *JWTManager {
	return &JWTManager{secretKey: secretKey, tokenDuration: tokenDuration}
}

// GenerateToken 生成JWT Token
func (manager *JWTManager) GenerateToken(userID uint64, username string) (string, error) {
	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(manager.tokenDuration).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(manager.secretKey))
}

// ParseToken 解析JWT Token
func (manager *JWTManager) ParseToken(accessToken string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(
		accessToken,
		&JWTClaims{},
		func(token *jwt.Token) (interface{}, error) {
			_, ok := token.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, errors.New("非法的token")
			}
			return []byte(manager.secretKey), nil
		},
	)

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, errors.New("无法解析token claims")
	}

	if claims.ExpiresAt < time.Now().Unix() {
		return nil, errors.New("token已过期")
	}

	return claims, nil
}
