/*
 * @Descripttion:
 * @version:
 * @Author: hujianghong
 * @Date: 2025-04-22 21:59:39
 * @LastEditors: hujianghong
 * @LastEditTime: 2025-04-22 23:31:03
 */
package config

import (
	"demo/pkg/database"
	"time"
)

// ServiceConfig 服务配置
type ServiceConfig struct {
	APIGateway    APIGatewayConfig
	UserService   UserServiceConfig
	MySQL         database.MySQLConfig
	Redis         database.RedisConfig
	JWTSecretKey  string
	TokenDuration time.Duration
}

// APIGatewayConfig API网关配置
type APIGatewayConfig struct {
	Port            int
	UserServiceAddr string
}

// UserServiceConfig 用户服务配置
type UserServiceConfig struct {
	Port int
}

// GetDefaultConfig 获取默认配置
func GetDefaultConfig() ServiceConfig {
	return ServiceConfig{
		APIGateway: APIGatewayConfig{
			Port:            8080,
			UserServiceAddr: "localhost:50051",
		},
		UserService: UserServiceConfig{
			Port: 50051,
		},
		MySQL: database.MySQLConfig{
			Host:     "localhost",
			Port:     3306,
			User:     "root",
			Password: "root", // 更常见的默认密码
			DBName:   "demo",
		},
		Redis: database.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       1,
		},
		JWTSecretKey:  "sdafasdfsdvmsodkfoasjfiadfjasmfaewo",
		TokenDuration: 24 * time.Hour,
	}
}
