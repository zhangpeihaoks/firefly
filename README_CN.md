# Firefly (萤火虫)

Firefly 是一个模块化、可扩展的 Go 后端服务器框架，采用分层架构设计。提供应用生命周期管理、传输层抽象、中间件系统、统一错误处理、结构化日志、序列化切换等核心能力。

[![Go Version](https://img.shields.io/github/go-mod/go-version/quajiu/firefly)](https://github.com/zhangpeihaoks/firefly)
[![License](https://img.shields.io/github/license/quajiu/firefly)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/zhangpeihaoks/firefly)](https://goreportcard.com/report/github.com/zhangpeihaoks/firefly)

[English Documentation](README.md)

## 特性

- **模块化架构**：传输层、中间件、服务层、数据层清晰分离
- **多协议支持**：HTTP 和 gRPC 服务器实现
- **中间件系统**：Recovery、Logging、Tracing、Metrics、Auth、RateLimit、CORS
- **统一错误处理**：一致的错误结构，支持 HTTP/gRPC 状态码转换
- **结构化日志**：JSON 格式日志，支持 lumberjack 日志轮转
- **服务注册发现**：文件、Consul 服务发现
- **数据库支持**：MySQL、PostgreSQL、MongoDB、Redis 连接器
- **可观测性**：OpenTelemetry 分布式追踪集成
- **指标监控**：Prometheus 指标自动收集
- **配置管理**：YAML 配置，支持环境变量覆盖
- **序列化**：可插拔序列化（JSON 和 Protobuf）
- **依赖注入**：自定义编译时依赖注入容器
- **插件系统**：可扩展插件架构，支持生命周期管理
- **完善测试**：核心组件全覆盖的属性测试
- **安全性**：TLS/HTTPS 支持、请求限制、日志脱敏

## 架构设计

```
┌─────────────────────────────────────────────────────────────────┐
│                        应用层 (Application Layer)                │
│                           (App)                                 │
│         生命周期管理 | 信号处理 | 优雅关闭                         │
└─────────────────────────────────────────────────────────────────┘
            │                              │
            ▼                              ▼
┌─────────────────────────┐    ┌─────────────────────────┐
│      HTTP Server        │    │      gRPC Server        │
│    (基于 Gin)           │    │    (grpc-go)            │
│  - 动态路由             │    │  - 拦截器               │
│  - 路由分组             │    │  - 健康检查             │
│  - 请求/响应处理        │    │  - 消息大小限制         │
└─────────────────────────┘    └─────────────────────────┘
            │                              │
            ▼                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      序列化层 (Serialization Layer)              │
│              JSON Serializer | Protobuf Serializer              │
└─────────────────────────────────────────────────────────────────┘
            │
            ▼
┌─────────────────────────────────────────────────────────────────┐
│                       中间件层 (Middleware Layer)                │
│  Recovery | Logging | Tracing | Metrics | Auth | RateLimit     │
│                      CORS                                          │
└─────────────────────────────────────────────────────────────────┘
            │
            ▼
┌─────────────────────────────────────────────────────────────────┐
│                      服务层 (Service Layer)                      │
│                  业务逻辑与处理函数                               │
└─────────────────────────────────────────────────────────────────┘
            │
            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    基础设施层 (Infrastructure Layer)              │
│  Config | Log | Registry | Discovery | Database | Cache        │
│         Health | Plugin | DI (自定义) | Tracing                   │
└─────────────────────────────────────────────────────────────────┘
```

## 快速开始

### 安装

```bash
go get github.com/zhangpeihaoks/firefly
```

### 基本 HTTP 服务器

```go
package main

import (
    "context"
    "net/http"
    "time"

    "github.com/zhangpeihaoks/firefly/app"
    "github.com/zhangpeihaoks/firefly/internal/log"
    "github.com/zhangpeihaoks/firefly/internal/middleware"
    httpserver "github.com/zhangpeihaoks/firefly/internal/transport/http"
)

func main() {
    // 初始化日志
    cleanup := log.New(&log.Config{
        FileName:   "app.log",
        MaxSize:    100,
        MaxBackups: 5,
        Level:      "info",
        JSONFormat: true,
    })
    defer cleanup()

    // 创建 HTTP 服务器
    server := httpserver.NewServer(
        httpserver.Address(":8080"),
        httpserver.Timeout(30*time.Second),
        httpserver.Middleware(
            middleware.Recovery(),
            middleware.Logging(),
        ),
    )

    // 注册路由
    server.Route(http.MethodGet, "/health", func(ctx context.Context, req any) (any, error) {
        return map[string]string{"status": "ok"}, nil
    })

    server.Route(http.MethodGet, "/users/:id", func(ctx context.Context, req any) (any, error) {
        userID, _ := httpserver.GetPathParamInt(ctx, "id")
        return map[string]interface{}{
            "id":   userID,
            "name": "张三",
        }, nil
    })

    // 创建并运行应用
    application := app.New(
        app.Name("my-service"),
        app.Server(server),
    )

    if code, err := application.Run(); err != nil {
        log.Error("应用启动失败", "error", err, "code", code)
    }
}
```

### 运行服务

```bash
go run main.go
```

服务将在 `http://localhost:8080` 启动。

## 配置管理

### YAML 配置文件

创建 `config.yaml` 文件：

```yaml
name: my-service
version: v1.0.0

http:
  network: tcp
  address: :8080
  timeout: 30s

grpc:
  network: tcp
  address: :9090
  timeout: 30s

log:
  filename: app.log
  max_size: 100      # MB
  max_backups: 5
  max_age: 7         # 天
  level: info
  json_format: true

registry:
  type: consul
  address: 127.0.0.1:8500

database:
  driver: mysql
  dsn: user:password@tcp(localhost:3306)/dbname
  pool:
    max_open_conns: 100
    max_idle_conns: 10
    conn_max_lifetime: 30m
    conn_max_idle_time: 10m

redis:
  address: localhost:6379
  password: ""
  db: 0
  pool_size: 100
  min_idle_conns: 10
  max_retries: 3

tracing:
  enabled: true
  endpoint: http://localhost:14268/api/traces
  sampler_ratio: 0.1

metrics:
  enabled: true
  path: /metrics

serializer:
  mode: json  # json 或 protobuf
```

### 加载配置

```go
import "github.com/zhangpeihaoks/firefly/internal/config"

cfg := config.New()
var appConfig Bootstrap
cfg.Load("config.yaml", &appConfig)
```

## 中间件使用

Firefly 提供内置中间件：

```go
server := httpserver.NewServer(
    httpserver.Address(":8080"),
    httpserver.Middleware(
        // Panic 恢复 - 防止服务崩溃
        middleware.Recovery(),
        
        // 请求/响应日志
        middleware.Logging(
            middleware.WithRequestHeader(true),
            middleware.WithRequestBody(true),
        ),
        
        // 分布式追踪
        middleware.Tracing(),
        
        // Prometheus 指标
        middleware.Metrics(),
        
        // 认证中间件
        middleware.Auth(authFunc),
        
        // 限流 (令牌桶: 100 req/s, 突发 200)
        middleware.RateLimit(middleware.WithRateLimiter(
            middleware.NewTokenBucketLimiter(100, 200),
        )),
        
        // CORS
        middleware.CORS(
            middleware.WithAllowOrigins("*"),
            middleware.WithAllowMethods("GET", "POST", "PUT", "DELETE"),
        ),
    ),
)
```

## 服务注册与发现

### 使用 Consul

```go
import (
    "github.com/zhangpeihaoks/firefly/internal/registry"
    "github.com/zhangpeihaoks/firefly/internal/registry/consul"
)

// 创建注册器
registrar := consul.NewRegistrar(&consul.RegistrarConfig{
    Address: "127.0.0.1:8500",
    Timeout: 10 * time.Second,
})

// 注册服务
err := registrar.Register(ctx, &registry.ServiceInstance{
    ID:        "my-service-1",
    Name:      "my-service",
    Version:   "v1.0.0",
    Endpoints: []string{"http://localhost:8080"},
    Metadata: map[string]string{
        "env": "production",
    },
})


// 注销服务
defer registrar.Deregister(ctx, "my-service-1")
```

### 使用文件服务发现

```yaml
# services.yaml
services:
  - id: user-service-1
    name: user-service
    version: v1.0.0
    endpoints:
      - http://localhost:8081
    metadata:
      env: production
```

```go
discovery := file.NewDiscovery("services.yaml")
instances, err := discovery.GetService(ctx, "user-service")
```

## 数据库连接

### MySQL/PostgreSQL

```go
import (
    "github.com/zhangpeihaoks/firefly/internal/database"
    "github.com/zhangpeihaoks/firefly/internal/database/mysql"
)

// 创建连接器
factory := mysql.NewFactory()
conn, err := factory.Create(&database.Config{
    Driver: "mysql",
    DSN:    "user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local",
    Pool: &database.PoolConfig{
        MaxOpenConns:    100,
        MaxIdleConns:    10,
        ConnMaxLifetime: 30 * time.Minute,
        ConnMaxIdleTime: 10 * time.Minute,
    },
})

// 获取 GORM DB
db := conn.DB()
db.AutoMigrate(&User{})
```

### MongoDB

```go
import "github.com/zhangpeihaoks/firefly/internal/database/mongodb"

factory := mongodb.NewFactory()
conn, err := factory.Create(&database.Config{
    Driver: "mongodb",
    DSN:    "mongodb://localhost:27017",
})
client := conn.Client()
```

### Redis

```go
import "github.com/zhangpeihaoks/firefly/internal/database/redis"

factory := redis.NewFactory()
conn, err := factory.Create(&database.Config{
    Driver: "redis",
    DSN:    "redis://localhost:6379/0",
})
rdb := conn.Client()
rdb.Set(ctx, "key", "value", time.Hour)
```

## 错误处理

```go
import "github.com/zhangpeihaoks/firefly/internal/errors"

// 定义业务错误
var (
    ErrUserNotFound = errors.New(errors.CodeNotFound, "USER_NOT_FOUND", "用户不存在")
    ErrInvalidInput = errors.New(errors.CodeBadRequest, "INVALID_INPUT", "输入参数无效")
)

// 在业务逻辑中使用
func GetUser(id int64) (*User, error) {
    user, err := findUser(id)
    if err != nil {
        return nil, ErrUserNotFound
    }
    return user, nil
}

// 在处理函数中返回
server.Route(http.MethodGet, "/users/:id", func(ctx context.Context, req any) (any, error) {
    user, err := GetUser(123)
    if err != nil {
        return nil, err // 自动转换为 HTTP 错误响应
    }
    return user, nil
})
```

## 健康检查

```go
import "github.com/zhangpeihaoks/firefly/internal/health"

checker := health.NewChecker()

// 添加自定义健康检查
checker.AddCheck("database", func(ctx context.Context) error {
    return db.Ping(ctx)
})

checker.AddReadinessCheck("redis", func(ctx context.Context) error {
    return redisClient.Ping(ctx).Err()
})

// 注册健康检查端点
server.Route(http.MethodGet, "/health", checker.HealthHandler())
server.Route(http.MethodGet, "/ready", checker.ReadinessHandler())
```

## 统一响应格式

```go
import "github.com/zhangpeihaoks/firefly/pkg/response"

// 成功响应
server.Route(http.MethodGet, "/users/:id", func(ctx context.Context, req any) (any, error) {
    user := getUser()
    return response.Success(user), nil
})

// 分页响应
server.Route(http.MethodGet, "/users", func(ctx context.Context, req any) (any, error) {
    users, total := listUsers(page, pageSize)
    return response.SuccessWithPage(users, page, pageSize, total), nil
})

// 响应格式：
// {
//   "code": 200,
//   "message": "success",
//   "data": { ... }
// }
```

## 项目结构

```
.
├── app/                    # 应用生命周期管理
│   ├── app.go             # 核心应用实现
│   └── app_test.go        # 单元测试和属性测试
├── conf/                   # 配置加载
│   └── conf.go            # Bootstrap 配置结构
├── config/                 # 配置文件
│   ├── config.yaml        # 默认配置
│   └── config.yaml.template
├── docs/                   # 文档
│   ├── config_port_example.md
│   └── request_response_usage.md
├── examples/               # 使用示例
│   ├── app_startup/       # 应用启动示例
│   ├── dynamic_routing/   # 动态路由示例
│   ├── metrics/           # Prometheus 指标示例
│   └── service_layer/     # 服务层模式示例
├── integration/            # 集成测试
│   ├── request_flow_test.go
│   ├── service_discovery_test.go
│   └── database_test.go
├── internal/               # 私有包
│   ├── config/            # 配置管理 (Viper)
│   ├── database/          # 数据库连接器 (MySQL, PostgreSQL, MongoDB, Redis)
│   ├── di/                # 依赖注入 (自定义 DI 容器)
│   ├── errors/            # 统一错误处理
│   ├── health/            # 健康检查端点
│   ├── log/               # 结构化日志 (slog + lumberjack)
│   ├── metrics/           # Prometheus 指标
│   ├── middleware/        # HTTP 中间件
│   ├── plugin/            # 插件系统
│   ├── registry/          # 服务注册发现 (文件, Consul)
│   ├── serializer/        # 序列化 (JSON, Protobuf)
│   ├── tracing/           # 分布式追踪 (OpenTelemetry)
│   └── transport/         # HTTP/gRPC 服务器
│       ├── http/          # HTTP 服务器 (基于 Gin)
│       └── grpc/          # gRPC 服务器
├── pkg/                    # 公共包
│   ├── config/            # 配置管理
│   ├── log/               # 结构化日志
│   └── response/          # 响应辅助函数
├── go.mod                  # Go 模块定义
├── main.go                  # 应用入口
├── Makefile               # 构建自动化
├── Dockerfile             # 容器构建
└── docker-compose.yml     # 开发环境
```

## 测试

运行单元测试：

```bash
go test ./... -v
```

运行测试并生成覆盖率：

```bash
go test ./... -coverprofile coverage.out
go tool cover -html=coverage.out
```

运行集成测试：

```bash
go test ./integration/... -v
```

运行属性测试：

```bash
go test ./internal/... -run "TestProperty" -v
```

### 测试覆盖

Firefly 使用属性测试（Property-Based Testing）与传统单元测试相结合。属性测试验证某些属性对所有输入都成立，提供更强的正确性保证。

主要测试的属性包括：
- **属性 1**：应用配置正确性
- **属性 2**：服务器并发管理
- **属性 4**：服务器配置正确性
- **属性 5**：中间件链执行顺序
- **属性 6**：Recovery 中间件 panic 捕获
- **属性 15**：配置加载正确性
- **属性 17**：路由注册正确性
- **属性 18**：路由分组正确性
- **属性 19**：动态路由参数解析
- **属性 20-24**：请求/响应处理
- **属性 25-47**：服务发现、健康检查、数据库、追踪、指标等

## 依赖

- **Gin** - HTTP Web 框架
- **gRPC** - RPC 框架
- **Viper** - 配置管理
- **slog** - 结构化日志（Go 1.21+ 标准库）
- **lumberjack** - 日志轮转
- **Prometheus** - 指标收集和暴露
- **OpenTelemetry** - 分布式追踪
- **GORM** - MySQL 和 PostgreSQL ORM
- **MongoDB Driver** - MongoDB 连接器
- **go-redis** - Redis 客户端
- **testing/quick** - 属性测试

## 文档

更多详细文档请参阅：

- [配置指南](docs/config_port_example.md)
- [请求/响应使用](docs/request_response_usage.md)

## 许可证

MIT 许可证 - 详见 [LICENSE](LICENSE)

## 贡献

欢迎贡献代码！请先阅读贡献指南。

---

## 需求映射

框架设计与需求文档完全对应，确保可追溯性：

| 功能模块 | 需求编号 | 说明 |
|---------|---------|------|
| 应用生命周期 | 1.x | 服务启动、关闭、信号处理 |
| 传输层 | 2.x, 7.x, 8.x | HTTP/gRPC 服务器 |
| 中间件 | 3.x | 洋葱模型中间件系统 |
| 错误处理 | 4.x | 统一错误结构和状态码转换 |
| 日志系统 | 5.x | 结构化日志和日志轮转 |
| 配置管理 | 6.x | YAML 配置和环境变量 |
| 路由管理 | 9.x | 动态路由和路由分组 |
| 请求响应 | 10.x | 统一请求响应处理 |
| 服务发现 | 11.x | 文件、Consul |
| 健康检查 | 12.x | /health 和 /ready 端点 |
| 依赖注入 | 13.x | 自定义编译时注入容器 |
| 插件系统 | 14.x | 可扩展插件架构 |
| 数据库 | 15.x | MySQL、PostgreSQL、MongoDB、Redis |
| 性能扩展 | 16.x | 限流、超时控制 |
| 分布式追踪 | 17.x | OpenTelemetry 集成 |
| 指标监控 | 18.x | Prometheus 指标 |
| 安全性 | 19.x | TLS、请求限制、日志脱敏 |
