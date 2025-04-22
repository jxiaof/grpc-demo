/*
 * @Descripttion: Session 认证管理器
 * @version: 1.0
 * @Author: hujianghong
 * @Date: 2025-04-23 10:30:17
 * @LastEditors: hujianghong
 * @LastEditTime: 2025-04-23 00:43:29
 */
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// SessionData 会话数据结构
type SessionData struct {
	UserID   uint64    `json:"user_id"`
	Username string    `json:"username"`
	Email    string    `json:"email,omitempty"`
	IssuedAt time.Time `json:"issued_at"`
	ExpireAt time.Time `json:"expire_at"`
}

// SessionManager 会话管理器
type SessionManager struct {
	redisClient   *redis.Client
	sessionPrefix string        // Redis key 前缀
	cookieName    string        // Cookie 名称
	sessionTTL    time.Duration // 会话有效期
}

// NewSessionManager 创建新的会话管理器
func NewSessionManager(redisClient *redis.Client, cookieName string, sessionTTL time.Duration) *SessionManager {
	return &SessionManager{
		redisClient:   redisClient,
		sessionPrefix: "session:",
		cookieName:    cookieName,
		sessionTTL:    sessionTTL,
	}
}

// GenerateSessionID 生成一个随机的会话 ID
func (m *SessionManager) GenerateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// CreateSession 创建新会话
func (m *SessionManager) CreateSession(ctx context.Context, userID uint64, username, email string) (string, error) {
	sessionID, err := m.GenerateSessionID()
	if err != nil {
		return "", fmt.Errorf("无法生成会话ID: %w", err)
	}

	now := time.Now()
	sessionData := SessionData{
		UserID:   userID,
		Username: username,
		Email:    email,
		IssuedAt: now,
		ExpireAt: now.Add(m.sessionTTL),
	}

	// 序列化会话数据
	sessionJSON, err := json.Marshal(sessionData)
	if err != nil {
		return "", fmt.Errorf("序列化会话数据失败: %w", err)
	}

	// 存储到Redis，设置过期时间
	key := m.sessionPrefix + sessionID
	err = m.redisClient.Set(ctx, key, sessionJSON, m.sessionTTL).Err()
	if err != nil {
		return "", fmt.Errorf("保存会话到Redis失败: %w", err)
	}

	// 同时在用户ID索引中保存此会话ID (便于后续查找/删除)
	userSessionsKey := fmt.Sprintf("user_sessions:%d", userID)
	err = m.redisClient.SAdd(ctx, userSessionsKey, sessionID).Err()
	if err != nil {
		// 尝试删除已创建的会话
		m.redisClient.Del(ctx, key)
		return "", fmt.Errorf("保存会话索引失败: %w", err)
	}

	// 用户会话列表的过期时间应该比单个会话长一些
	m.redisClient.Expire(ctx, userSessionsKey, m.sessionTTL*2)

	return sessionID, nil
}

// GetSession 获取会话数据
func (m *SessionManager) GetSession(ctx context.Context, sessionID string) (*SessionData, error) {
	key := m.sessionPrefix + sessionID
	sessionJSON, err := m.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, errors.New("会话不存在或已过期")
		}
		return nil, fmt.Errorf("获取会话数据失败: %w", err)
	}

	var sessionData SessionData
	if err := json.Unmarshal(sessionJSON, &sessionData); err != nil {
		return nil, fmt.Errorf("解析会话数据失败: %w", err)
	}

	// 检查会话是否已过期
	if time.Now().After(sessionData.ExpireAt) {
		_ = m.DestroySession(ctx, sessionID)
		return nil, errors.New("会话已过期")
	}

	// 刷新会话过期时间 (滑动过期)
	m.RefreshSession(ctx, sessionID)

	return &sessionData, nil
}

// RefreshSession 刷新会话过期时间
func (m *SessionManager) RefreshSession(ctx context.Context, sessionID string) error {
	key := m.sessionPrefix + sessionID

	// 获取当前会话数据
	sessionJSON, err := m.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return errors.New("会话不存在或已过期")
		}
		return fmt.Errorf("获取会话数据失败: %w", err)
	}

	var sessionData SessionData
	if err := json.Unmarshal(sessionJSON, &sessionData); err != nil {
		return fmt.Errorf("解析会话数据失败: %w", err)
	}

	// 更新过期时间
	sessionData.ExpireAt = time.Now().Add(m.sessionTTL)

	// 重新序列化并保存
	updatedJSON, err := json.Marshal(sessionData)
	if err != nil {
		return fmt.Errorf("序列化更新后的会话数据失败: %w", err)
	}

	// 保存回Redis
	return m.redisClient.SetEX(ctx, key, updatedJSON, m.sessionTTL).Err()
}

// DestroySession 销毁会话
func (m *SessionManager) DestroySession(ctx context.Context, sessionID string) error {
	// 先获取会话数据，查找对应的用户ID
	key := m.sessionPrefix + sessionID
	sessionJSON, err := m.redisClient.Get(ctx, key).Bytes()

	if err == nil {
		var sessionData SessionData
		if err := json.Unmarshal(sessionJSON, &sessionData); err == nil {
			// 从用户的会话列表中移除
			userSessionsKey := fmt.Sprintf("user_sessions:%d", sessionData.UserID)
			m.redisClient.SRem(ctx, userSessionsKey, sessionID)
		}
	}

	// 删除会话，即使上面的操作失败也继续
	return m.redisClient.Del(ctx, key).Err()
}

// DestroyAllUserSessions 销毁用户的所有会话
func (m *SessionManager) DestroyAllUserSessions(ctx context.Context, userID uint64) error {
	userSessionsKey := fmt.Sprintf("user_sessions:%d", userID)

	// 获取用户所有会话ID
	sessionIDs, err := m.redisClient.SMembers(ctx, userSessionsKey).Result()
	if err != nil {
		return fmt.Errorf("获取用户会话列表失败: %w", err)
	}

	// 逐个删除会话
	for _, sessionID := range sessionIDs {
		key := m.sessionPrefix + sessionID
		m.redisClient.Del(ctx, key)
	}

	// 删除用户会话列表
	return m.redisClient.Del(ctx, userSessionsKey).Err()
}

// ValidateSession 验证会话是否有效
func (m *SessionManager) ValidateSession(ctx context.Context, sessionID string) bool {
	_, err := m.GetSession(ctx, sessionID)
	return err == nil
}
