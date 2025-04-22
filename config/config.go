/*
 * @Descripttion: 配置管理
 * @version: 1.0
 * @Author: hujianghong
 * @Date: 2025-04-22 21:59:39
 * @LastEditors: hujianghong
 * @LastEditTime: 2025-04-23 00:48:33
 */
package config

import (
	"demo/pkg/database"
	"demo/pkg/logger"
	"time"
)

// ServiceConfig 服务配置
type ServiceConfig struct {
	APIGateway       APIGatewayConfig
	UserService      UserServiceConfig
	MySQL            database.MySQLConfig
	Redis            database.RedisConfig
	JWTSecretKey     string
	SessionSecretKey string // 添加会话密钥
	TokenDuration    time.Duration
	Logger           logger.LogConfig
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
			Password: "root",
			DBName:   "demo",
		},
		Redis: database.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
		},
		JWTSecretKey:     "sdafasdfsdvmsodkfoasjfiadfjasmfaewo",
		SessionSecretKey: "sdafasdfsdvmsodkfoasjfiadfjasmfaewo",
		TokenDuration:    24 * time.Hour,
		Logger: logger.LogConfig{
			Level:      "debug",   // 开发环境使用debug级别
			Encoding:   "console", // 开发环境使用console格式，生产环境建议使用json
			OutputPath: "stdout",  // 标准输出
			ErrorPath:  "stderr",  // 标准错误输出
		},
	}
}

// GetProductionConfig 获取生产环境配置
func GetProductionConfig() ServiceConfig {
	config := GetDefaultConfig()
	// 修改为生产环境的配置
	config.Logger.Level = "info"
	config.Logger.Encoding = "json"
	// 在实际生产中，密钥应该从环境变量或专用的密钥管理服务中获取
	// config.JWTSecretKey = os.Getenv("JWT_SECRET_KEY")
	// config.SessionSecretKey = os.Getenv("SESSION_SECRET_KEY")
	return config
}

// GetTestConfig 获取测试环境配置
func GetTestConfig() ServiceConfig {
	config := GetDefaultConfig()
	config.MySQL.DBName = "demo_test"
	config.Redis.DB = 1
	return config
}
