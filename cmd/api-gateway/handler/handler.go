package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	pb "demo/api/proto/user"
	"demo/pkg/auth"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// UserHandler 用户处理器
type UserHandler struct {
	userClient     pb.UserServiceClient
	logger         *zap.Logger
	sessionManager *auth.SessionManager
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userClient pb.UserServiceClient, ssm *auth.SessionManager, logger *zap.Logger) *UserHandler {
	return &UserHandler{
		userClient:     userClient,
		logger:         logger,
		sessionManager: ssm,
	}
}

// Register 用户注册
// @Summary 用户注册
// @Description 注册新用户
// @Tags 用户
// @Accept json
// @Produce json
// @Param user body RegisterRequest true "注册信息"
// @Success 200 {object} RegisterResponse "成功"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 500 {object} ErrorResponse "服务器内部错误"
// @Router /register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	// 绑定请求参数
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Debug("请求参数错误",
			zap.Error(err),
			zap.String("client_ip", c.ClientIP()))

		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数错误"})
		return
	}

	h.logger.Debug("收到注册请求",
		zap.String("username", req.Username),
		zap.String("email", req.Email),
		zap.String("client_ip", c.ClientIP()))

	// 调用gRPC服务
	startTime := time.Now()
	resp, err := h.userClient.Register(c.Request.Context(), &pb.RegisterRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	duration := time.Since(startTime)

	if err != nil {
		h.logger.Error("注册服务调用失败",
			zap.Error(err),
			zap.String("username", req.Username),
			zap.Duration("duration", duration))

		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "服务器内部错误"})
		return
	}

	// 记录结果日志
	if resp.Success {
		h.logger.Info("用户注册成功",
			zap.String("username", req.Username),
			zap.Uint64("user_id", resp.UserId),
			zap.Duration("duration", duration))
	} else {
		h.logger.Warn("用户注册失败",
			zap.String("username", req.Username),
			zap.String("reason", resp.Message),
			zap.Duration("duration", duration))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": resp.Success,
		"message": resp.Message,
		"user_id": resp.UserId,
	})
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户登录并获取认证会话
// @Tags 用户
// @Accept json
// @Produce json
// @Param user body LoginRequest true "登录信息"
// @Success 200 {object} LoginResponse "成功"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 401 {object} ErrorResponse "认证失败"
// @Failure 500 {object} ErrorResponse "服务器内部错误"
// @Router /login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数错误"})
		return
	}

	resp, err := h.userClient.Login(c.Request.Context(), &pb.LoginRequest{
		Username: req.Username,
		Password: req.Password,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "服务器内部错误"})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": resp.Message,
		})
		return
	}

	// 设置Cookie
	c.SetCookie(
		"session_id",                // 名称
		resp.Token,                  // 值（sessionID）
		int(24*time.Hour.Seconds()), // 过期时间（24小时）
		"/",                         // 路径
		"",                          // 域名
		false,                       // 是否仅HTTPS
		true,                        // HttpOnly，防止JavaScript访问
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "登录成功",
		"token":   resp.Token, // 同时在响应中返回token
		"user": gin.H{
			"id":         resp.User.Id,
			"username":   resp.User.Username,
			"email":      resp.User.Email,
			"created_at": resp.User.CreatedAt,
		},
	})
}

// GetUserInfo 获取用户信息
// @Summary 获取用户信息
// @Description 获取指定ID的用户信息
// @Tags 用户
// @Accept json
// @Produce json
// @Param id path integer true "用户ID"
// @Security Bearer
// @Success 200 {object} GetUserInfoResponse "成功"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 401 {object} ErrorResponse "未授权"
// @Failure 404 {object} ErrorResponse "用户不存在"
// @Failure 500 {object} ErrorResponse "服务器内部错误"
// @Router /users/{id} [get]
func (h *UserHandler) GetUserInfo(c *gin.Context) {
	// 从路径获取userID
	userID := c.Param("id")

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "用户ID不能为空"})
		return
	}

	// 解析userID
	var id uint64
	if _, err := fmt.Sscanf(userID, "%d", &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的用户ID"})
		return
	}

	resp, err := h.userClient.GetUserInfo(c.Request.Context(), &pb.GetUserInfoRequest{
		UserId: id,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "服务器内部错误"})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": resp.Message,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user": gin.H{
			"id":         resp.User.Id,
			"username":   resp.User.Username,
			"email":      resp.User.Email,
			"created_at": resp.User.CreatedAt,
		},
	})
}

// AuthMiddleware Session认证中间件
func (h *UserHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// 尝试从Cookie中获取
			sessionID, err := c.Cookie("session_id")
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未提供认证信息"})
				c.Abort()
				return
			}
			authHeader = "Bearer " + sessionID
		}

		// Bearer token格式处理
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "认证格式错误"})
			c.Abort()
			return
		}

		h.logger.Info("----------> 验证Session authHeader", zap.String("authHeader", authHeader))
		// 获取SessionID
		sessionID := strings.TrimSpace(parts[1])
		h.logger.Info("----------> 验证Session sessionID", zap.String("sessionID", sessionID))

		// 验证Session
		resp, err := h.userClient.ValidateSession(c.Request.Context(), &pb.ValidateSessionRequest{
			SessionId: sessionID,
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "会话验证失败"})
			c.Abort()
			return
		}

		if !resp.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": resp.Message})
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("user_id", resp.UserId)
		c.Set("username", resp.Username)
		c.Set("session_id", sessionID)

		c.Next()
	}
}

// Logout 用户登出
// @Summary 用户登出
// @Description 退出登录并销毁会话
// @Tags 用户
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} map[string]interface{} "成功"
// @Failure 401 {object} ErrorResponse "未授权"
// @Failure 500 {object} ErrorResponse "服务器内部错误"
// @Router /logout [post]
func (h *UserHandler) Logout(c *gin.Context) {
	sessionID, exists := c.Get("session_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未授权"})
		return
	}

	// 调用服务销毁会话
	resp, err := h.userClient.Logout(c.Request.Context(), &pb.LogoutRequest{
		SessionId: sessionID.(string),
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "服务器内部错误"})
		return
	}

	// 清除客户端Cookie
	c.SetCookie(
		"session_id", // 名称
		"",           // 值（空）
		-1,           // 过期时间（立即过期）
		"/",          // 路径
		"",           // 域名
		false,        // 是否仅HTTPS
		true,         // HttpOnly，防止JavaScript访问
	)

	c.JSON(http.StatusOK, gin.H{
		"success": resp.Success,
		"message": resp.Message,
	})
}
