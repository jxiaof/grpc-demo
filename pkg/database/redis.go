/*
 * @Descripttion:
 * @version:
 * @Author: hujianghong
 * @Date: 2025-04-22 21:58:38
 * @LastEditors: hujianghong
 * @LastEditTime: 2025-04-22 21:58:42
 */
package database

import (
	"context"
	"fmt"
	"log"

	"github.com/go-redis/redis/v8"
)

// RedisConfig 保存Redis配置
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// NewRedisClient 创建Redis客户端
func NewRedisClient(config RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
		DB:       config.DB,
	})

	// 测试连接
	ctx := context.Background()
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, err
	}

	log.Println("成功连接到Redis服务器")
	return client, nil
}
