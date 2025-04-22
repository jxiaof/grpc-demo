package mysql

/*
 * @Descripttion: 用户仓库
 * @version:
 * @Author: hujianghong
 * @Date: 2025-04-22 21:58:51
 * @LastEditors: hujianghong
 * @LastEditTime: 2025-04-22 22:30:20
 */

import (
	"demo/internal/model"
	"errors"
	"time"

	"gorm.io/gorm"
)

// UserRepository 用户仓库
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓库
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser 创建用户
func (r *UserRepository) CreateUser(user *model.User) (uint64, error) {
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	result := r.db.Create(user)
	if result.Error != nil {
		return 0, result.Error
	}

	return user.ID, nil
}

// GetUserByUsername 根据用户名获取用户
func (r *UserRepository) GetUserByUsername(username string) (*model.User, error) {
	var user model.User
	result := r.db.Where("username = ?", username).First(&user)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, result.Error
	}

	return &user, nil
}

// GetUserByID 根据ID获取用户
func (r *UserRepository) GetUserByID(id uint64) (*model.User, error) {
	var user model.User
	result := r.db.First(&user, id)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, result.Error
	}

	return &user, nil
}

// IsUsernameExists 检查用户名是否存在
func (r *UserRepository) IsUsernameExists(username string) (bool, error) {
	var count int64
	result := r.db.Model(&model.User{}).Where("username = ?", username).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}

	return count > 0, nil
}

// IsEmailExists 检查邮箱是否存在
func (r *UserRepository) IsEmailExists(email string) (bool, error) {
	var count int64
	result := r.db.Model(&model.User{}).Where("email = ?", email).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}

	return count > 0, nil
}
