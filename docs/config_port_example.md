# 配置端口示例

## 概述

从现在开始，HTTP 和 gRPC 服务器的端口配置支持两种方式：

## 方式 1: 使用 `port` 字段（推荐）

直接写端口号，更简洁：

```yaml
# HTTP server configuration
http:
  network: tcp
  port: 8080  # 直接写端口号
  timeout: 30s
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s

# gRPC server configuration
grpc:
  network: tcp
  port: 9090  # 直接写端口号
  timeout: 30s
  max_recv_msg_size: 4194304  # 4MB
  max_send_msg_size: 4194304  # 4MB
```

## 方式 2: 使用 `address` 字段

写完整的地址字符串：

```yaml
# HTTP server configuration
http:
  network: tcp
  address: :8080  # 或 "127.0.0.1:8080"
  timeout: 30s
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s

# gRPC server configuration
grpc:
  network: tcp
  address: :9090  # 或 "127.0.0.1:9090"
  timeout: 30s
  max_recv_msg_size: 4194304  # 4MB
  max_send_msg_size: 4194304  # 4MB
```

## 优先级

如果同时配置了 `port` 和 `address`，**`port` 优先**：

```yaml
http:
  port: 8080      # 这个会被使用
  address: :9090  # 这个会被忽略
```

## 使用方法

在代码中使用 `GetAddress()` 方法获取完整地址：

```go
// 加载配置
var bootstrap config.Bootstrap
if err := cfg.Load("config.yaml", &bootstrap); err != nil {
    log.Fatal(err)
}

// 创建 HTTP 服务器
httpSrv := http.NewServer(
    http.Address(bootstrap.HTTP.GetAddress()), // 使用 GetAddress() 方法
)

// 创建 gRPC 服务器
grpcSrv := grpc.NewServer(
    grpc.Address(bootstrap.GRPC.GetAddress()), // 使用 GetAddress() 方法
)
```

## 默认值

如果不配置端口，会使用默认值：

- HTTP 默认端口: `8080`
- gRPC 默认端口: `9090`

```go
// 使用默认配置
httpConfig := config.DefaultHTTPConfig()
fmt.Println(httpConfig.GetAddress()) // 输出: :8080

grpcConfig := config.DefaultGRPCConfig()
fmt.Println(grpcConfig.GetAddress()) // 输出: :9090
```

## 验证

配置验证会检查 `port` 或 `address` 至少提供一个：

```go
// 错误：两个都没配置
config := &config.HTTPConfig{}
errs := config.Validate()
// errs 会包含错误：address or port is required

// 正确：只配置 port
config := &config.HTTPConfig{Port: 8080}
errs := config.Validate()
// errs 为空，验证通过

// 正确：只配置 address
config := &config.HTTPConfig{Address: ":8080"}
errs := config.Validate()
// errs 为空，验证通过
```
