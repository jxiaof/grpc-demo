/*
 * @Descripttion:
 * @version:
 * @Author: hujianghong
 * @Date: 2025-04-22 22:00:08
 * @LastEditors: hujianghong
 * @LastEditTime: 2025-04-23 00:24:28
 */
package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"demo/config"
	"demo/internal/repository/mysql"
	"demo/internal/repository/redis"
	"demo/internal/service"
	"demo/pkg/auth"
	"demo/pkg/database"
	"demo/pkg/logger"

	pb "demo/api/proto/user"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	// 加载配置
	cfg := config.GetDefaultConfig()

	// 初始化日志
	log := logger.InitLogger(cfg.Logger)
	defer logger.Sync() // 确保所有日志都被刷新

	log.Info("用户服务启动中...",
		zap.Int("port", cfg.UserService.Port),
		zap.String("mysql_host", cfg.MySQL.Host),
		zap.String("redis_host", cfg.Redis.Host))

	// 连接MySQL
	mysqlDB, err := database.NewMySQLConnection(cfg.MySQL)
	if err != nil {
		log.Fatal("无法连接到MySQL", zap.Error(err))
	}

	// 获取底层SQL连接以便最后关闭
	sqlDB, err := mysqlDB.DB()
	if err != nil {
		log.Fatal("获取SQL连接失败", zap.Error(err))
	}
	defer sqlDB.Close()
	log.Info("成功连接到MySQL数据库")

	// 连接Redis
	redisClient, err := database.NewRedisClient(cfg.Redis)
	if err != nil {
		log.Fatal("无法连接到Redis", zap.Error(err))
	}
	defer redisClient.Close()
	log.Info("成功连接到Redis服务器")

	// 创建Session管理器
	sessionManager := auth.NewSessionManager(redisClient, "session_id", cfg.TokenDuration)
	log.Debug("创建Session管理器",
		zap.Duration("token_duration", cfg.TokenDuration))

	// 创建仓库
	mysqlUserRepo := mysql.NewUserRepository(mysqlDB)
	redisUserRepo := redis.NewUserRepository(redisClient)
	log.Debug("创建数据仓库")

	// 创建用户服务
	userService := service.NewUserService(
		mysqlUserRepo,
		redisUserRepo,
		sessionManager,
	)
	log.Debug("创建用户服务")

	// 创建gRPC服务器
	server := grpc.NewServer()
	pb.RegisterUserServiceServer(server, userService)
	log.Debug("注册gRPC服务")

	// 启动服务器
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.UserService.Port))
	if err != nil {
		log.Fatal("无法监听端口",
			zap.Int("port", cfg.UserService.Port),
			zap.Error(err))
	}

	// 处理优雅关闭
	go func() {
		log.Info("用户服务启动在端口", zap.Int("port", cfg.UserService.Port))
		if err := server.Serve(lis); err != nil {
			log.Fatal("无法启动服务", zap.Error(err))
		}
	}()

	// 等待中断信号优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("正在关闭用户服务...")
	server.GracefulStop()
	log.Info("用户服务已关闭")
}
