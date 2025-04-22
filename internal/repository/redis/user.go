package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"demo/internal/model"

	"github.com/go-redis/redis/v8"
)

type UserRepository struct {
	client *redis.Client
}

func NewUserRepository(client *redis.Client) *UserRepository {
	return &UserRepository{client: client}
}

// 缓存用户
func (r *UserRepository) CacheUser(ctx context.Context, user *model.User) error {
	userData, err := json.Marshal(user)
	if err != nil {
		return err
	}

	// 同时使用用户ID和用户名作为键
	idKey := fmt.Sprintf("user:id:%d", user.ID)
	usernameKey := fmt.Sprintf("user:username:%s", user.Username)

	// 设置缓存，过期时间30分钟
	if err := r.client.Set(ctx, idKey, userData, 30*time.Minute).Err(); err != nil {
		return err
	}
	if err := r.client.Set(ctx, usernameKey, userData, 30*time.Minute).Err(); err != nil {
		return err
	}

	return nil
}

// 根据用户ID获取用户
func (r *UserRepository) GetUserByID(ctx context.Context, id uint64) (*model.User, error) {
	key := fmt.Sprintf("user:id:%d", id)
	userData, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 缓存未命中
		}
		return nil, err
	}

	var user model.User
	if err := json.Unmarshal(userData, &user); err != nil {
		return nil, err
	}

	// 刷新过期时间
	r.client.Expire(ctx, key, 30*time.Minute)

	return &user, nil
}

// 根据用户名获取用户
func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	key := fmt.Sprintf("user:username:%s", username)
	userData, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 缓存未命中
		}
		return nil, err
	}

	var user model.User
	if err := json.Unmarshal(userData, &user); err != nil {
		return nil, err
	}

	// 刷新过期时间
	r.client.Expire(ctx, key, 30*time.Minute)

	return &user, nil
}

// 缓存Token
func (r *UserRepository) CacheToken(ctx context.Context, userID uint64, token string, expiration time.Duration) error {
	key := fmt.Sprintf("token:user:%d", userID)
	return r.client.Set(ctx, key, token, expiration).Err()
}

// 获取用户Token
func (r *UserRepository) GetToken(ctx context.Context, userID uint64) (string, error) {
	key := fmt.Sprintf("token:user:%d", userID)
	token, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil // 缓存未命中
		}
		return "", err
	}
	return token, nil
}

// 删除用户Token
func (r *UserRepository) DeleteToken(ctx context.Context, userID uint64) error {
	key := fmt.Sprintf("token:user:%d", userID)
	return r.client.Del(ctx, key).Err()
}
