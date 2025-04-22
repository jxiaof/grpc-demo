package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "demo/api/proto/user"
	"demo/cmd/api-gateway/handler"
	"demo/config"
	"demo/docs"
	"demo/pkg/auth"
	"demo/pkg/database"
	"demo/pkg/logger"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 加载配置
	cfg := config.GetDefaultConfig()

	// 初始化日志
	log := logger.InitLogger(cfg.Logger)
	defer logger.Sync() // 确保所有日志都被刷新

	log.Info("API网关启动中...",
		zap.Int("port", cfg.APIGateway.Port),
		zap.String("user_service_addr", cfg.APIGateway.UserServiceAddr))

	// 设置Swagger信息
	docs.SwaggerInfo.Host = fmt.Sprintf("localhost:%d", cfg.APIGateway.Port)
	docs.SwaggerInfo.BasePath = "/api"
	docs.SwaggerInfo.Title = "用户服务 API"
	docs.SwaggerInfo.Description = "微服务用户系统API文档"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Schemes = []string{"http"}

	// 连接到用户服务
	conn, err := grpc.Dial(
		cfg.APIGateway.UserServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal("无法连接到用户服务", zap.Error(err))
	}
	defer conn.Close()
	log.Info("成功连接到用户服务")

	userClient := pb.NewUserServiceClient(conn)

	// 创建Redis客户端(用于会话管理)
	redisClient, err := database.NewRedisClient(cfg.Redis)
	if err != nil {
		log.Fatal("无法连接到Redis", zap.Error(err))
	}
	defer redisClient.Close()
	log.Info("成功连接到Redis服务器")

	// 创建Session管理器
	sessionManager := auth.NewSessionManager(redisClient, "session_id", cfg.TokenDuration)

	// 创建处理器
	userHandler := handler.NewUserHandler(userClient, sessionManager, log)

	// 设置Gin日志模式
	if cfg.Logger.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 设置Gin路由
	router := gin.Default()

	// 配置中间件和路由...
	configureRouter(router, userHandler, log)

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.APIGateway.Port),
		Handler: router,
	}

	// 优雅启动服务器
	startServerGracefully(srv, log, cfg.APIGateway.Port)
}

// 配置路由和中间件
func configureRouter(router *gin.Engine, userHandler *handler.UserHandler, log *zap.Logger) {
	// 添加日志中间件
	router.Use(createLogMiddleware(log))

	// 启用CORS中间件
	router.Use(createCorsMiddleware())

	// 添加Swagger文档路由
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 公共API
	apiGroup := router.Group("/api")
	{
		apiGroup.POST("/register", userHandler.Register)
		apiGroup.POST("/login", userHandler.Login)
	}

	// 需要认证的API
	authRouter := apiGroup.Group("")
	authRouter.Use(userHandler.AuthMiddleware())
	{
		authRouter.GET("/users/:id", userHandler.GetUserInfo)
		authRouter.POST("/logout", userHandler.Logout)
		// 可以添加更多需要认证的API路由
	}

	// 添加一个重定向到Swagger UI的根路由
	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})
}

// 创建日志中间件
func createLogMiddleware(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 获取请求耗时
		latency := time.Since(start)

		// 获取客户端信息
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		// 记录访问日志
		log.Info("HTTP请求",
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("client_ip", clientIP),
		)
	}
}

// 创建CORS中间件
func createCorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// 优雅启动和关闭服务器
func startServerGracefully(srv *http.Server, log *zap.Logger, port int) {
	// 优雅启动服务器
	go func() {
		log.Info("API网关启动在端口",
			zap.Int("port", port),
			zap.String("swagger_url", fmt.Sprintf("http://localhost:%d/swagger/index.html", port)))

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("无法启动API网关", zap.Error(err))
		}
	}()

	// 等待中断信号优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("正在关闭API网关服务器...")

	// 创建一个5秒的上下文用于超时
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("API网关关闭过程中出错", zap.Error(err))
	}

	log.Info("API网关服务器已安全关闭")
}
