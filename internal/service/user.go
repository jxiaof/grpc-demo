package service

import (
	"context"
	"demo/internal/model"
	"demo/internal/repository/mysql"
	"demo/internal/repository/redis"
	"demo/pkg/auth"
	"demo/pkg/util"
	"time"

	pb "demo/api/proto/user"
)

// UserService 用户服务
type UserService struct {
	pb.UnimplementedUserServiceServer
	mysqlRepo  *mysql.UserRepository
	redisRepo  *redis.UserRepository
	jwtManager *auth.JWTManager
}

// NewUserService 创建用户服务
func NewUserService(
	mysqlRepo *mysql.UserRepository,
	redisRepo *redis.UserRepository,
	jwtManager *auth.JWTManager,
) *UserService {
	return &UserService{
		mysqlRepo:  mysqlRepo,
		redisRepo:  redisRepo,
		jwtManager: jwtManager,
	}
}

// Register 用户注册
func (s *UserService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	// 检查用户名是否已存在
	exists, err := s.mysqlRepo.IsUsernameExists(req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return &pb.RegisterResponse{
			Success: false,
			Message: "用户名已存在",
		}, nil
	}

	// 检查邮箱是否已存在
	exists, err = s.mysqlRepo.IsEmailExists(req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return &pb.RegisterResponse{
			Success: false,
			Message: "邮箱已注册",
		}, nil
	}

	// 哈希密码
	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// 创建新用户
	user := &model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashedPassword,
	}

	// 保存到MySQL
	userID, err := s.mysqlRepo.CreateUser(user)
	if err != nil {
		return nil, err
	}

	user.ID = userID

	// 缓存到Redis
	if err := s.redisRepo.CacheUser(ctx, user); err != nil {
		// 仅记录错误，不影响注册流程
		// 应该添加日志记录
	}

	return &pb.RegisterResponse{
		Success: true,
		Message: "注册成功",
		UserId:  userID,
	}, nil
}

// Login 用户登录
func (s *UserService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	// 先从Redis缓存查询
	user, err := s.redisRepo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		// 日志记录Redis错误
	}

	// Redis缓存未命中，从MySQL查询
	if user == nil {
		user, err = s.mysqlRepo.GetUserByUsername(req.Username)
		if err != nil {
			return &pb.LoginResponse{
				Success: false,
				Message: "用户不存在或密码错误",
			}, nil
		}

		// 将用户信息缓存到Redis
		if err := s.redisRepo.CacheUser(ctx, user); err != nil {
			// 记录日志
		}
	}

	// 验证密码
	if !util.CheckPassword(req.Password, user.Password) {
		return &pb.LoginResponse{
			Success: false,
			Message: "用户不存在或密码错误",
		}, nil
	}

	// 生成JWT token
	token, err := s.jwtManager.GenerateToken(user.ID, user.Username)
	if err != nil {
		return nil, err
	}

	// 缓存token到Redis
	if err := s.redisRepo.CacheToken(ctx, user.ID, token, 24*time.Hour); err != nil {
		// 记录日志
	}

	return &pb.LoginResponse{
		Success: true,
		Message: "登录成功",
		Token:   token,
		User: &pb.UserInfo{
			Id:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}

// GetUserInfo 获取用户信息
func (s *UserService) GetUserInfo(ctx context.Context, req *pb.GetUserInfoRequest) (*pb.GetUserInfoResponse, error) {
	// 先从Redis缓存查询
	user, err := s.redisRepo.GetUserByID(ctx, req.UserId)
	if err != nil {
		// 日志记录Redis错误
	}

	// Redis缓存未命中，从MySQL查询
	if user == nil {
		user, err = s.mysqlRepo.GetUserByID(req.UserId)
		if err != nil {
			if err.Error() == "用户不存在" {
				return &pb.GetUserInfoResponse{
					Success: false,
					Message: "用户不存在",
				}, nil
			}
			return nil, err
		}

		// 将用户信息缓存到Redis
		if err := s.redisRepo.CacheUser(ctx, user); err != nil {
			// 记录日志
		}
	}

	return &pb.GetUserInfoResponse{
		Success: true,
		User: &pb.UserInfo{
			Id:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}
