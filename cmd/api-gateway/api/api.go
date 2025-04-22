/*
 * @Descripttion:
 * @version:
 * @Author: hujianghong
 * @Date: 2025-04-22 23:42:34
 * @LastEditors: hujianghong
 * @LastEditTime: 2025-04-23 01:02:48
 */
package api

import (
	"fmt"
	"time"
)

// 以下是为Swagger自动生成文档定义的结构体和接口

// @title 用户服务 API
// @version 1.0
// @description 这是一个 gRPC 微服务的 API 网关，提供用户注册、登录和信息获取功能
// @host localhost:8080
// @BasePath /api
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description 请输入 'Bearer ' + token

// RegisterRequest 注册请求结构
type RegisterRequest struct {
	Username string `json:"username" binding:"required" example:"testuser"`            // 用户名
	Email    string `json:"email" binding:"required,email" example:"test@example.com"` // 电子邮件
	Password string `json:"password" binding:"required,min=6" example:"123456"`        // 密码
}

// RegisterResponse 注册响应结构
type RegisterResponse struct {
	Success bool   `json:"success" example:"true"` // 是否成功
	Message string `json:"message" example:"注册成功"` // 返回消息
	UserID  uint64 `json:"user_id" example:"1"`    // 用户ID
}

// LoginRequest 登录请求结构
type LoginRequest struct {
	Username string `json:"username" binding:"required" example:"testuser"` // 用户名
	Password string `json:"password" binding:"required" example:"123456"`   // 密码
}

// UserInfo 用户信息结构
type UserInfo struct {
	ID        uint64    `json:"id" example:"1"`                                 // 用户ID
	Username  string    `json:"username" example:"testuser"`                    // 用户名
	Email     string    `json:"email" example:"test@example.com"`               // 电子邮件
	CreatedAt time.Time `json:"created_at" example:"2025-04-22T15:04:05+08:00"` // 创建时间
}

// LoginResponse 登录响应结构
type LoginResponse struct {
	Success bool     `json:"success" example:"true"`                                  // 是否成功
	Message string   `json:"message" example:"登录成功"`                                  // 返回消息
	Token   string   `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."` // JWT令牌
	User    UserInfo `json:"user"`                                                    // 用户信息
}

// GetUserInfoResponse 获取用户信息响应
type GetUserInfoResponse struct {
	Success bool     `json:"success" example:"true"` // 是否成功
	User    UserInfo `json:"user"`                   // 用户信息
}

// ErrorResponse 错误响应结构
type ErrorResponse struct {
	Success bool   `json:"success" example:"false"` // 是否成功
	Message string `json:"message" example:"错误信息"`  // 错误信息
}

// LogoutResponse 登出响应结构
type LogoutResponse struct {
	Success bool   `json:"success" example:"true"` // 是否成功
	Message string `json:"message" example:"成功登出"` // 返回消息
}

func init() {
	fmt.Println("------------------API package initialized------------------")
}
