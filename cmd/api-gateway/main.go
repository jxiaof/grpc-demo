package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	pb "demo/api/proto/user"
	"demo/config"
	"demo/pkg/auth"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type UserHandler struct {
	userClient pb.UserServiceClient
	jwtManager *auth.JWTManager
}

func NewUserHandler(userClient pb.UserServiceClient, jwtManager *auth.JWTManager) *UserHandler {
	return &UserHandler{
		userClient: userClient,
		jwtManager: jwtManager,
	}
}

// 用户注册
func (h *UserHandler) Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求参数错误"})
		return
	}

	resp, err := h.userClient.Register(c.Request.Context(), &pb.RegisterRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "服务器内部错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": resp.Success,
		"message": resp.Message,
		"user_id": resp.UserId,
	})
}

// 用户登录
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

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "登录成功",
		"token":   resp.Token,
		"user": gin.H{
			"id":         resp.User.Id,
			"username":   resp.User.Username,
			"email":      resp.User.Email,
			"created_at": resp.User.CreatedAt,
		},
	})
}

// 获取用户信息
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

// JWT认证中间件
func (h *UserHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "未提供认证信息"})
			c.Abort()
			return
		}

		// Bearer token格式处理
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "认证格式错误"})
			c.Abort()
			return
		}

		// 解析Token
		claims, err := h.jwtManager.ParseToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "无效的认证令牌"})
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)

		c.Next()
	}
}

func main() {
	// 加载配置
	cfg := config.GetDefaultConfig()

	// 连接到用户服务
	conn, err := grpc.Dial(
		cfg.APIGateway.UserServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("无法连接到用户服务: %v", err)
	}
	defer conn.Close()

	userClient := pb.NewUserServiceClient(conn)

	// 创建JWT管理器
	jwtManager := auth.NewJWTManager(cfg.JWTSecretKey, cfg.TokenDuration)

	// 创建处理器
	userHandler := NewUserHandler(userClient, jwtManager)

	// 设置Gin路由
	router := gin.Default()

	// 公共API
	router.POST("/api/register", userHandler.Register)
	router.POST("/api/login", userHandler.Login)

	// 需要认证的API
	authRouter := router.Group("/api")
	authRouter.Use(userHandler.AuthMiddleware())
	{
		authRouter.GET("/users/:id", userHandler.GetUserInfo)
		// 可以添加更多需要认证的API路由
	}

	// 启动服务器
	addr := fmt.Sprintf(":%d", cfg.APIGateway.Port)
	log.Printf("API网关启动在端口 %d", cfg.APIGateway.Port)

	if err := router.Run(addr); err != nil {
		log.Fatalf("无法启动API网关: %v", err)
	}
}
