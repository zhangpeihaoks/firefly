# Firefly (萤火虫)

Firefly is a modular, extensible Go backend server framework with a layered architecture. It provides application lifecycle management, transport layer abstraction, middleware system, unified error handling, structured logging, serialization switching, and comprehensive property-based testing.

[![Go Version](https://img.shields.io/github/go-mod/go-version/quajiu/firefly)](https://github.com/zhangpeihaoks/firefly)
[![License](https://img.shields.io/github/license/quajiu/firefly)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/zhangpeihaoks/firefly)](https://goreportcard.com/report/github.com/zhangpeihaoks/firefly)

## Features

- **Modular Architecture**: Clean separation between transport, middleware, service, and data layers
- **Multiple Protocol Support**: HTTP and gRPC server implementations
- **Middleware System**: Recovery, Logging, Tracing, Metrics, Auth, RateLimit, CORS
- **Unified Error Handling**: Consistent error structure with HTTP/gRPC status code conversion
- **Structured Logging**: JSON-formatted logs with rotation support via lumberjack
- **Service Registry**: File-based and Consul service discovery
- **Database Support**: MySQL, PostgreSQL, MongoDB, and Redis connectors
- **Observability**: OpenTelemetry integration for distributed tracing
- **Metrics**: Prometheus metrics with automatic collection
- **Configuration Management**: YAML-based configuration with environment variable support
- **Serialization**: Pluggable serialization (JSON and Protobuf)
- **Dependency Injection**: Custom compile-time dependency injection container
- **Plugin System**: Extensible plugin architecture with lifecycle management
- **Comprehensive Testing**: Property-based testing for all core components
- **Security**: TLS/HTTPS support, request limiting, log masking

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Application Layer                          │
│                           (App)                                 │
│         Lifecycle Management | Signal Handling | Graceful Stop  │
└─────────────────────────────────────────────────────────────────┘
            │                              │
            ▼                              ▼
┌─────────────────────────┐    ┌─────────────────────────┐
│    HTTP Server          │    │    gRPC Server          │
│  (Gin-based)            │    │  (grpc-go)              │
│  - Dynamic Routing      │    │  - Interceptors         │
│  - Route Groups         │    │  - Health Check         │
│  - Request/Response     │    │  - Max Message Size     │
└─────────────────────────┘    └─────────────────────────┘
            │                              │
            ▼                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Serialization Layer                          │
│              JSON Serializer | Protobuf Serializer              │
└─────────────────────────────────────────────────────────────────┘
            │
            ▼
┌─────────────────────────────────────────────────────────────────┐
│                       Middleware Layer                          │
│  Recovery | Logging | Tracing | Metrics | Auth | RateLimit        │
│                      CORS                                          │
└─────────────────────────────────────────────────────────────────┘
            │
            ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Service Layer                              │
│              Business Logic & Handlers                          │
└─────────────────────────────────────────────────────────────────┘
            │
            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Infrastructure Layer                         │
│  Config | Log | Registry | Discovery | Database | Cache        │
│         Health | Plugin | DI | Tracing                   │
└─────────────────────────────────────────────────────────────────┘
```

## Quick Start

### Installation

```bash
go get github.com/zhangpeihaoks/firefly
```

### Basic HTTP Server

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
	// Initialize logger
	cleanup := log.New(&log.Config{
		FileName:   "app.log",
		MaxSize:    100,
		MaxBackups: 5,
		Level:      "info",
		JSONFormat: true,
	})
	defer cleanup()

	// Create HTTP server
	server := httpserver.NewServer(
		httpserver.Address(":8080"),
		httpserver.Timeout(30*time.Second),
		httpserver.Middleware(
			middleware.Recovery(),
			middleware.Logging(),
		),
	)

	// Register routes
	server.Route(http.MethodGet, "/health", func(ctx context.Context, req any) (any, error) {
		return map[string]string{"status": "ok"}, nil
	})

	server.Route(http.MethodGet, "/users/:id", func(ctx context.Context, req any) (any, error) {
		userID, _ := httpserver.GetPathParamInt(ctx, "id")
		return map[string]interface{}{
			"id":   userID,
			"name": "John Doe",
		}, nil
	})

	// Create and run application
	application := app.New(
		app.Name("my-service"),
		app.Server(server),
	)

	if code, err := application.Run(); err != nil {
		log.Error("application failed", "error", err, "code", code)
	}
}
```

### Running the Service

```bash
go run main.go
```

The server will start on `http://localhost:8080`.

## Configuration

### YAML Configuration

Create a `config.yaml` file:

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
  max_size: 100
  max_backups: 5
  max_age: 7
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

redis:
  address: localhost:6379
  password: ""
  db: 0
  pool_size: 10
  min_idle_conns: 5
  max_retries: 3

tracing:
  enabled: true
  endpoint: http://localhost:14268/api/traces
  sampler_ratio: 0.1

metrics:
  enabled: true
  path: /metrics
```

### Loading Configuration

```go
cfg := config.New()
var appConfig Bootstrap
cfg.Load("config.yaml", &appConfig)
```

## Middleware

Firefly provides built-in middleware:

```go
server := httpserver.NewServer(
    httpserver.Address(":8080"),
    httpserver.Middleware(
        // Panic recovery - prevents crashes
        middleware.Recovery(),
        
        // Request/response logging
        middleware.Logging(),
        
        // Distributed tracing
        middleware.Tracing(),
        
        // Prometheus metrics
        middleware.Metrics(),
        
        // Rate limiting (token bucket: 100 req/s, burst 200)
        middleware.RateLimit(middleware.WithRateLimiter(
            middleware.NewTokenBucketLimiter(100, 200),
        )),
        
        // CORS
        middleware.CORS(),
    ),
)
```

## Service Registration

```go
// Using Consul
import "github.com/zhangpeihaoks/firefly/internal/registry/consul"

registrar := consul.NewRegistrar(&consul.RegistrarConfig{
    Address: "127.0.0.1:8500",
    Timeout: 10 * time.Second,
})

// Register on startup
registrar.Register(ctx, &registry.ServiceInstance{
    ID:      "my-service-1",
    Name:    "my-service",
    Version: "v1.0.0",
    Endpoints: []string{"http://localhost:8080"},
})

// Deregister on shutdown
defer registrar.Deregister(ctx, "my-service-1")
```

## Database

```go
import (
    "github.com/zhangpeihaoks/firefly/internal/database"
    "github.com/zhangpeihaoks/firefly/internal/database/mysql"
)

// Create connector
factory := mysql.NewFactory()
conn, err := factory.Create(&database.Config{
    Driver: "mysql",
    DSN:    "user:password@tcp(localhost:3306)/dbname",
})

// Connect and ping
if err := conn.Ping(ctx); err != nil {
    log.Fatal(err)
}
```

## Error Handling

```go
import "github.com/zhangpeihaoks/firefly/internal/errors"

// Return errors with codes
func GetUser(id int64) (*User, error) {
    user, err := findUser(id)
    if err != nil {
        return nil, errors.New(errors.CodeNotFound, "USER_NOT_FOUND", "User not found")
    }
    return user, nil
}

// Use in handlers
server.Route(http.MethodGet, "/users/:id", func(ctx context.Context, req any) (any, error) {
    user, err := GetUser(123)
    if err != nil {
        return nil, err // Automatic HTTP error response
    }
    return user, nil
})
```

## Project Structure

```
.
├── app/                    # Application lifecycle management
│   ├── app.go             # Core application implementation
│   └── app_test.go        # Unit and property-based tests
├── conf/                   # Configuration loading
│   └── conf.go            # Bootstrap config struct
├── config/                 # Configuration files
│   ├── config.yaml        # Default configuration
│   └── config.yaml.template
├── docs/                   # Documentation
│   ├── config_port_example.md
│   └── request_response_usage.md
├── examples/               # Usage examples
│   ├── app_startup/       # Application startup example
│   ├── dynamic_routing/   # Dynamic routing example
│   ├── metrics/           # Prometheus metrics example
│   └── service_layer/     # Service layer pattern example
├── integration/            # Integration tests
│   ├── request_flow_test.go
│   ├── service_discovery_test.go
│   └── database_test.go
├── internal/               # Private packages
│   ├── config/            # Configuration management (Viper)
│   ├── database/          # Database connectors (MySQL, PostgreSQL, MongoDB, Redis)
│   ├── di/                # Dependency injection (Custom DI container)
│   ├── errors/            # Unified error handling
│   ├── health/            # Health check endpoints
│   ├── log/               # Structured logging (slog + lumberjack)
│   ├── metrics/           # Prometheus metrics
│   ├── middleware/        # HTTP middleware
│   ├── plugin/            # Plugin system
│   ├── registry/          # Service registry (File, Consul)
│   ├── serializer/        # Serialization (JSON, Protobuf)
│   ├── tracing/           # Distributed tracing (OpenTelemetry)
│   └── transport/         # HTTP/gRPC servers
│       ├── http/          # HTTP server with Gin
│       └── grpc/          # gRPC server
├── pkg/                    # Public packages
│   ├── config/            # Configuration management
│   ├── log/               # Structured logging
│   └── response/          # Response helpers
├── go.mod                  # Go module definition
├── main.go                  # Application entry point
├── Makefile               # Build automation
├── Dockerfile             # Container build
└── docker-compose.yml     # Development environment
```

## Testing

Run unit tests:

```bash
go test ./... -v
```

Run tests with coverage:

```bash
go test ./... -coverprofile coverage.out
go tool cover -html=coverage.out
```

Run integration tests:

```bash
go test ./integration/... -v
```

Run property-based tests specifically:

```bash
go test ./internal/... -run "TestProperty" -v
```

### Test Coverage

Firefly uses property-based testing (PBT) alongside traditional unit tests. Property-based tests verify that certain properties hold for all inputs, providing stronger correctness guarantees.

Key properties tested:
- **Property 1**: Application configuration correctness
- **Property 2**: Server concurrent management
- **Property 4**: Server configuration correctness
- **Property 5**: Middleware chain execution order
- **Property 6**: Recovery middleware panic capture
- **Property 15**: Config loading correctness
- **Property 17**: Route registration correctness
- **Property 18**: Route grouping correctness
- **Property 19**: Dynamic route parameter parsing
- **Property 20-24**: Request/response handling
- **Property 25-47**: Service discovery, health checks, database, tracing, metrics, etc.

## Dependencies

- **Gin** - HTTP web framework
- **gRPC** - RPC framework
- **Viper** - Configuration management
- **slog** - Structured logging (Go 1.21+ standard library)
- **lumberjack** - Log rotation
- **Prometheus** - Metrics collection and exposition
- **OpenTelemetry** - Distributed tracing
- **GORM** - SQL ORM for MySQL and PostgreSQL
- **MongoDB Driver** - MongoDB connector
- **go-redis** - Redis client
- **testing/quick** - Property-based testing

## Documentation

For more detailed documentation, see:

- [Configuration Guide](docs/config_port_example.md)
- [Request/Response Usage](docs/request_response_usage.md)

## License

MIT License - see [LICENSE](LICENSE) for details.

## Contributing

Contributions are welcome! Please read the contributing guidelines first.

---

## 中文文档 (Chinese Documentation)

[点击此处查看完整中文文档](README_CN.md)

### 快速概览

Firefly 是一个模块化、可扩展的 Go 后端服务器框架，采用分层架构设计。

**核心特性：**
- 应用生命周期管理，支持优雅关闭
- HTTP 和 gRPC 双协议支持
- 完善的中间件系统（Recovery、Logging、Tracing、Metrics 等）
- 统一错误处理，支持 HTTP/gRPC 状态码转换
- 结构化日志，支持日志轮转
- 多数据库支持（MySQL、PostgreSQL、MongoDB、Redis）
- 服务注册与发现（文件、Consul）
- 分布式追踪（OpenTelemetry）
- Prometheus 指标监控
- 完善的属性测试覆盖

**快速开始：**

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

更多详情请参阅 [中文文档](README_CN.md)。