package service

import (
	"context"
	"demo/internal/model"
	"demo/internal/repository/mysql"
	"demo/internal/repository/redis"
	"demo/pkg/auth"
	"demo/pkg/logger"
	"demo/pkg/util"
	"time"

	pb "demo/api/proto/user"

	"go.uber.org/zap"
)

// UserService 用户服务
type UserService struct {
	pb.UnimplementedUserServiceServer
	mysqlRepo      *mysql.UserRepository
	redisRepo      *redis.UserRepository
	sessionManager *auth.SessionManager
	logger         *zap.Logger
}

// NewUserService 创建用户服务
func NewUserService(
	mysqlRepo *mysql.UserRepository,
	redisRepo *redis.UserRepository,
	ssm *auth.SessionManager,
) *UserService {
	return &UserService{
		mysqlRepo:      mysqlRepo,
		redisRepo:      redisRepo,
		sessionManager: ssm,
		logger:         logger.GetLogger(),
	}
}

// Register 用户注册
func (s *UserService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	s.logger.Debug("---> 处理注册请求",
		zap.String("username", req.Username),
		zap.String("email", req.Email))

	// 检查用户名是否已存在
	startTime := time.Now()
	exists, err := s.mysqlRepo.IsUsernameExists(req.Username)
	if err != nil {
		s.logger.Error("检查用户名是否存在失败",
			zap.String("username", req.Username),
			zap.Error(err))
		return nil, err
	}
	if exists {
		s.logger.Info("用户名已存在",
			zap.String("username", req.Username),
			zap.Duration("duration", time.Since(startTime)))
		return &pb.RegisterResponse{
			Success: false,
			Message: "用户名已存在",
		}, nil
	}

	// 检查邮箱是否已存在
	exists, err = s.mysqlRepo.IsEmailExists(req.Email)
	if err != nil {
		s.logger.Error("检查邮箱是否存在失败",
			zap.String("email", req.Email),
			zap.Error(err))
		return nil, err
	}
	if exists {
		s.logger.Info("邮箱已注册",
			zap.String("email", req.Email),
			zap.Duration("duration", time.Since(startTime)))
		return &pb.RegisterResponse{
			Success: false,
			Message: "邮箱已注册",
		}, nil
	}

	// 哈希密码
	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		s.logger.Error("密码哈希失败", zap.Error(err))
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
		s.logger.Error("创建用户失败",
			zap.String("username", req.Username),
			zap.Error(err))
		return nil, err
	}

	user.ID = userID

	// 缓存到Redis
	if err := s.redisRepo.CacheUser(ctx, user); err != nil {
		// 仅记录错误，不影响注册流程
		s.logger.Warn("缓存用户信息失败",
			zap.Uint64("user_id", userID),
			zap.Error(err))
	}

	s.logger.Info("用户注册成功",
		zap.String("username", req.Username),
		zap.String("email", req.Email),
		zap.Uint64("user_id", userID),
		zap.Duration("duration", time.Since(startTime)))

	return &pb.RegisterResponse{
		Success: true,
		Message: "注册成功",
		UserId:  userID,
	}, nil
}

// Login 用户登录
func (s *UserService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	s.logger.Debug("处理登录请求",
		zap.String("username", req.Username))

	// 先从Redis缓存查询
	user, err := s.redisRepo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		s.logger.Warn("从Redis获取用户信息失败",
			zap.String("username", req.Username),
			zap.Error(err))
	}

	// Redis缓存未命中，从MySQL查询
	if user == nil {
		user, err = s.mysqlRepo.GetUserByUsername(req.Username)
		if err != nil {
			s.logger.Error("从MySQL获取用户信息失败",
				zap.String("username", req.Username),
				zap.Error(err))
			return &pb.LoginResponse{
				Success: false,
				Message: "用户不存在或密码错误",
			}, nil
		}

		// 将用户信息缓存到Redis
		if err := s.redisRepo.CacheUser(ctx, user); err != nil {
			s.logger.Warn("缓存用户信息失败",
				zap.String("username", req.Username),
				zap.Error(err))
		}
	}

	// 验证密码
	if !util.CheckPassword(req.Password, user.Password) {
		s.logger.Info("密码验证失败",
			zap.String("username", req.Username))
		return &pb.LoginResponse{
			Success: false,
			Message: "用户不存在或密码错误",
		}, nil
	}

	// 生成session token
	token, err := s.sessionManager.CreateSession(ctx, user.ID, user.Username, user.Email)
	if err != nil {
		s.logger.Error("生成session token失败",
			zap.String("username", req.Username),
			zap.Error(err))
		return nil, err
	}

	// 缓存token到Redis
	if err := s.redisRepo.CacheToken(ctx, user.ID, token, 24*time.Hour); err != nil {
		s.logger.Warn("缓存token失败",
			zap.Uint64("user_id", user.ID),
			zap.Error(err))
	}

	s.logger.Info("用户登录成功",
		zap.String("username", req.Username),
		zap.Uint64("user_id", user.ID))

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
	s.logger.Debug("处理获取用户信息请求",
		zap.Uint64("user_id", req.UserId))

	// 先从Redis缓存查询
	user, err := s.redisRepo.GetUserByID(ctx, req.UserId)
	if err != nil {
		s.logger.Warn("从Redis获取用户信息失败",
			zap.Uint64("user_id", req.UserId),
			zap.Error(err))
	}

	// Redis缓存未命中，从MySQL查询
	if user == nil {
		user, err = s.mysqlRepo.GetUserByID(req.UserId)
		if err != nil {
			if err.Error() == "用户不存在" {
				s.logger.Info("用户不存在",
					zap.Uint64("user_id", req.UserId))
				return &pb.GetUserInfoResponse{
					Success: false,
					Message: "用户不存在",
				}, nil
			}
			s.logger.Error("从MySQL获取用户信息失败",
				zap.Uint64("user_id", req.UserId),
				zap.Error(err))
			return nil, err
		}

		// 将用户信息缓存到Redis
		if err := s.redisRepo.CacheUser(ctx, user); err != nil {
			s.logger.Warn("缓存用户信息失败",
				zap.Uint64("user_id", req.UserId),
				zap.Error(err))
		}
	}

	s.logger.Info("获取用户信息成功",
		zap.Uint64("user_id", req.UserId))

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
