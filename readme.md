# gRPC 微服务示例项目

这是一个使用 Go、gRPC、Gin、MySQL 和 Redis 实现的微服务架构示例项目。

## 环境准备

### 安装 Protocol Buffers 编译器

```bash
brew install protobuf
```

### 安装 Go 的 protobuf 插件

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### 安装 Swagger 工具

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

### 添加 Go bin 目录到环境变量

```bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc
```

## 数据库设置

### MySQL
1. 确保 MySQL 服务正在运行
2. 创建数据库：`CREATE DATABASE demo;`
3. 或者让系统自动创建表结构 (GORM AutoMigrate)

### Redis
确保 Redis 服务正在运行，默认端口为 6379

## 编译 Protocol Buffers

```bash
protoc --go_out=. --go-grpc_out=. api/proto/user.proto
```

## 生成 Swagger 文档(如需更新)

```bash
swag init -g cmd/api-gateway/main.go -o docs
```

## 运行服务

### 启动用户服务 (gRPC)

```bash
go run main.go user-service
```

### 启动 API 网关 (Gin HTTP)

在另一个终端中运行：
```bash
go run main.go api-gateway
```

## 访问 API 文档

启动服务后，访问以下地址查看 API 文档：

```
http://localhost:8080/swagger/index.html
```

## API 测试

### 注册用户

```bash
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser", "email":"test@example.com", "password":"123456"}'
```

### 用户登录

```bash
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser", "password":"123456"}'
```

### 获取用户信息

使用登录返回的 token：
```bash
curl -X GET http://localhost:8080/api/users/2 \
  -H "Authorization: Bearer uN347MK0DS_JoV5d4eIx8bPavX8hTKuQmkDo4C6x4Pg="
```

## 项目结构

- `api/`: API 定义，包含 Protocol Buffers 文件
- `cmd/`: 应用程序入口点
  - `api-gateway/`: API 网关服务 (Gin)
  - `user-service/`: 用户服务 (gRPC)
- `config/`: 配置管理
- `docs/`: Swagger API 文档
- `internal/`: 内部包
  - `model/`: 数据模型
  - `repository/`: 数据访问层
  - `service/`: 业务逻辑
- `pkg/`: 共享库
  - `auth/`: 认证相关
  - `database/`: 数据库连接
  - `util/`: 实用工具函数

## 使用 Docker 运行数据库（可选）

### MySQL

```bash
docker run --name mysql-demo \
  -p 3306:3306 \
  -e MYSQL_ROOT_PASSWORD=root \
  -e MYSQL_DATABASE=demo \
  -d mysql:8.0 \
  --character-set-server=utf8mb4 \
  --collation-server=utf8mb4_unicode_ci
```

### Redis

```bash
docker run --name redis-demo \
  -p 6379:6379 \
  -d redis:7.0 \
  --requirepass "" \
  --databases 16
```