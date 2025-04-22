/*
 * @Descripttion:
 * @version:
 * @Author: hujianghong
 * @Date: 2025-04-22 22:00:08
 * @LastEditors: hujianghong
 * @LastEditTime: 2025-04-22 22:40:43
 */
package main

import (
	"fmt"
	"log"
	"net"

	"demo/config"
	"demo/internal/repository/mysql"
	"demo/internal/repository/redis"
	"demo/internal/service"
	"demo/pkg/auth"
	"demo/pkg/database"

	pb "demo/api/proto/user"

	"google.golang.org/grpc"
)

func main() {
	// 加载配置
	cfg := config.GetDefaultConfig()

	// 连接MySQL
	mysqlDB, err := database.NewMySQLConnection(cfg.MySQL)
	if err != nil {
		log.Fatalf("无法连接到MySQL: %v", err)
	}

	// 获取底层SQL连接以便最后关闭
	sqlDB, err := mysqlDB.DB()
	if err != nil {
		log.Fatalf("获取SQL连接失败: %v", err)
	}
	defer sqlDB.Close()

	// 连接Redis
	redisClient, err := database.NewRedisClient(cfg.Redis)
	if err != nil {
		log.Fatalf("无法连接到Redis: %v", err)
	}
	defer redisClient.Close()

	// 创建JWT管理器
	jwtManager := auth.NewJWTManager(cfg.JWTSecretKey, cfg.TokenDuration)

	// 创建仓库
	mysqlUserRepo := mysql.NewUserRepository(mysqlDB)
	redisUserRepo := redis.NewUserRepository(redisClient)

	// 创建用户服务
	userService := service.NewUserService(mysqlUserRepo, redisUserRepo, jwtManager)

	// 创建gRPC服务器
	server := grpc.NewServer()
	pb.RegisterUserServiceServer(server, userService)

	// 启动服务器
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.UserService.Port))
	if err != nil {
		log.Fatalf("无法监听端口: %v", err)
	}

	log.Printf("用户服务启动在端口 %d", cfg.UserService.Port)
	if err := server.Serve(lis); err != nil {
		log.Fatalf("无法启动服务: %v", err)
	}
}
