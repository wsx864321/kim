# KRPC

KRPC (Kim RPC) 是一个基于 gRPC 的 RPC 框架，提供了服务发现、服务注册、负载均衡、服务治理（监控、追踪、限流、熔断、超时）等功能。

## 特性

- ✅ **服务注册与发现**：支持基于 etcd 的服务注册与发现
- ✅ **负载均衡**：支持轮询（Round Robin）、P2C（Power of Two Choices）等负载均衡算法
- ✅ **服务治理**：
  - 📊 **监控指标**：集成 Prometheus，提供请求计数、延迟直方图等指标
  - 🔍 **分布式追踪**：支持 OpenTelemetry/Jaeger 追踪
  - 🚦 **限流**：基于令牌桶算法的限流器
  - ⚡ **熔断**：基于 sony/gobreaker 的熔断器
  - ⏱️ **超时控制**：客户端请求超时和慢请求检测
  - 🛡️ **异常恢复**：自动恢复 panic，防止服务崩溃
- ✅ **协议传输**：基于 HTTP/2 协议
- ✅ **数据序列化**：使用 Protocol Buffers 进行数据序列化

## 快速开始

### 服务端示例

```go
package main

import (
    "context"
    "github.com/wsx864321/kim/pkg/krpc"
    "github.com/wsx864321/kim/pkg/krpc/registry/etcd"
    "google.golang.org/grpc"
)

func main() {
    // 创建 etcd 注册中心
    registry, _ := etcd.NewETCDRegister(
        etcd.WithEndpoints([]string{"127.0.0.1:2379"}),
        etcd.WithDialTimeout(5*time.Second),
    )

    // 创建 gRPC 服务器
    server := krpc.NewPServer(
        krpc.WithServiceName("my-service"),
        krpc.WithPort(9001),
        krpc.WithWeight(100),
        krpc.WithRegistry(registry),
    )

    // 注册服务
    server.RegisterService(func(s *grpc.Server) {
        // 注册你的 gRPC 服务
        // pb.RegisterYourServiceServer(s, &YourServiceImpl{})
    })

    // 可选：注册自定义拦截器（如限流）
    // server.RegisterUnaryServerInterceptor(customInterceptor)

    // 启动服务
    server.Start(context.Background())
}
```

### 客户端示例

```go
package main

import (
    "context"
    "github.com/wsx864321/kim/pkg/krpc"
    "github.com/wsx864321/kim/pkg/krpc/registry/etcd"
    "github.com/wsx864321/kim/pkg/krpc/interceptor/client"
    "time"
)

func main() {
    // 创建 etcd 注册中心
    registry, _ := etcd.NewETCDRegister(
        etcd.WithEndpoints([]string{"127.0.0.1:2379"}),
    )

    // 创建 gRPC 客户端
    cli, err := krpc.NewKClient(
        krpc.WithClientServiceName("my-service"),
        krpc.WithClientRegistry(registry),
        // 可选：添加熔断器
        krpc.WithClientInterceptors(
            client.BreakerUnaryClientInterceptor(
                "my-service",
                100,                    // maxRequest
                10*time.Second,        // interval
                5*time.Second,         // timeout
                nil,                   // readyToTrip
            ),
            // 可选：添加超时控制
            client.TimeoutUnaryClientInterceptor(
                5*time.Second,         // timeout
                1*time.Second,         // slowThreshold
            ),
        ),
    )
    if err != nil {
        panic(err)
    }

    // 使用客户端连接
    conn := cli.Conn()
    // client := pb.NewYourServiceClient(conn)
    // resp, err := client.YourMethod(context.Background(), &pb.Request{})
}
```

## 核心概念

### RPC 介绍

RPC (Remote Procedure Call)：远程过程调用，它是一种通过网络从远程计算机程序上请求服务，而不需要了解底层网络技术的思想。

通常在同一服务内部我们采用函数调用的方式进行通信，那么如果在服务与服务之间就需要用到 RPC 了。

### 协议传输

KRPC 基于 gRPC，gRPC 默认情况下采用的是 HTTP/2 协议，因此 KRPC 的协议采用的自然就是 HTTP/2。

**相关文档：**
- [HTTP2协议解析](https://www.jianshu.com/p/42ca44202ca4)
- [gRPC系列(三) 如何借助HTTP2实现传输](https://zhuanlan.zhihu.com/p/161577635)

### 数据序列化

KRPC 基于 gRPC，gRPC 采用 Protocol Buffers，因此 KRPC 也是采用 Protocol Buffers 进行数据序列化。

**相关文档：**
- [Protocol Buffer 简介](https://blog.csdn.net/mzpmzk/article/details/80824839)
- [Protocol Buffer原理解密](https://juejin.cn/post/6844904099991494663)

## 服务治理

### 监控指标 (Metrics)

KRPC 集成了 Prometheus，自动收集以下指标：

**服务端指标：**
- `prpc_server_req_client_handle_total`：请求总数（按方法、服务名、状态码、IP 分组）
- `prpc_server_req_client_handle_seconds`：请求延迟直方图（按方法、服务名、IP 分组）

**客户端指标：**
- `prpc_client_req_client_handle_total`：请求总数（按方法、服务名、状态码、IP 分组）
- `prpc_client_req_client_handle_seconds`：请求延迟直方图（按方法、服务名、IP 分组）

### 分布式追踪 (Trace)

KRPC 支持 OpenTelemetry/Jaeger 分布式追踪，自动在请求中传播追踪上下文。

**使用示例：**
```go
import "github.com/wsx864321/kim/pkg/krpc/trace"

// 启动追踪 Agent
trace.StartAgent()
defer trace.StopAgent()
```

### 限流 (Rate Limiting)

限流采用基于 `juju/ratelimit` 的令牌桶算法实现。

**使用示例：**
```go
import (
    "github.com/wsx864321/kim/pkg/krpc/interceptor/server"
    "time"
)

configs := map[server.MethodName]server.RateLimitConfig{
    "/your.service/YourMethod": {
        Cap:             100,              // 桶容量
        Rate:            10.0,            // 每秒令牌生成速率
        WaitMaxDuration: 100 * time.Millisecond, // 最大等待时间
    },
}

interceptor := server.RateLimitUnaryServerInterceptor(configs)
server.RegisterUnaryServerInterceptor(interceptor)
```

**算法介绍：**
- [流量控制算法——漏桶算法和令牌桶算法](https://zhuanlan.zhihu.com/p/165006444)

**自适应过载保护算法：**
- [BBR Limit](https://segmentfault.com/a/1190000041950209?sort=votes) - QUIC 在弱网环境下使用 BBR 算法进行拥塞控制

### 熔断 (Circuit Breaker)

熔断基于 `sony/gobreaker` 实现，当服务出现大量错误时自动熔断，防止级联故障。

**使用示例：**
```go
import "github.com/wsx864321/kim/pkg/krpc/interceptor/client"

interceptor := client.BreakerUnaryClientInterceptor(
    "my-service",           // 名称
    100,                    // maxRequest: 半开状态下最大请求数
    10*time.Second,         // interval: 统计时间窗口
    5*time.Second,          // timeout: 超时时间
    nil,                     // readyToTrip: 自定义熔断条件
)

cli, _ := krpc.NewKClient(
    krpc.WithClientServiceName("my-service"),
    krpc.WithClientInterceptors(interceptor),
)
```

### 超时控制 (Timeout)

客户端支持请求超时控制和慢请求检测。

**使用示例：**
```go
import "github.com/wsx864321/kim/pkg/krpc/interceptor/client"

interceptor := client.TimeoutUnaryClientInterceptor(
    5*time.Second,  // timeout: 默认超时时间
    1*time.Second,  // slowThreshold: 慢请求阈值
)

cli, _ := krpc.NewKClient(
    krpc.WithClientServiceName("my-service"),
    krpc.WithClientInterceptors(interceptor),
)
```

### 异常恢复 (Recovery)

服务端自动捕获 panic 并记录堆栈信息，防止服务崩溃。

**默认启用**：服务端自动启用，无需额外配置。

## 服务注册与发现

### etcd 注册中心

KRPC 支持基于 etcd 的服务注册与发现。

**服务注册：**
- 服务启动时自动注册到 etcd
- 支持服务权重配置
- 自动心跳保活
- 优雅关闭时自动注销

**服务发现：**
- 自动监听服务变化
- 本地缓存服务列表
- 支持负载均衡

**使用示例：**
```go
import "github.com/wsx864321/kim/pkg/krpc/registry/etcd"

// 创建注册中心
registry, err := etcd.NewETCDRegister(
    etcd.WithEndpoints([]string{"127.0.0.1:2379"}),
    etcd.WithDialTimeout(5*time.Second),
    etcd.WithKeepAliveInterval(10*time.Second),
)
```

## 负载均衡

KRPC 支持多种负载均衡策略：

- **Round Robin**：轮询算法（默认）
- **P2C (Power of Two Choices)**：随机选择两个节点，选择负载较低的
- **WRR (Weighted Round Robin)**：加权轮询

**配置示例：**
```go
// 客户端默认使用 Round Robin
cli, _ := krpc.NewKClient(
    krpc.WithClientServiceName("my-service"),
)

// 服务端配置权重
server := krpc.NewPServer(
    krpc.WithWeight(100), // 权重越高，被选中的概率越大
)
```

## 配置选项

### 服务端选项

- `WithServiceName(name string)`：设置服务名称
- `WithPort(port int)`：设置服务端口
- `WithWeight(weight int)`：设置服务权重（默认 100）
- `WithRegistry(registry Registrar)`：设置服务注册中心

### 客户端选项

- `WithClientServiceName(name string)`：设置目标服务名称
- `WithClientRegistry(registry Registrar)`：设置服务注册中心
- `WithClientInterceptors(...)`：设置客户端拦截器
- `WithDirect(direct bool)`：是否直连服务地址
- `WithURL(url string)`：直接设置服务地址

## 最佳实践

1. **服务注册**：生产环境建议使用 etcd 等服务注册中心
2. **监控告警**：集成 Prometheus 监控，设置合理的告警规则
3. **限流配置**：根据服务容量合理配置限流参数
4. **熔断配置**：根据服务特性配置熔断阈值
5. **超时设置**：根据业务需求设置合理的超时时间
6. **优雅关闭**：确保服务关闭时正确注销服务注册

## 许可证

本项目采用开源许可证，各位可以根据自己的想法在此基础上进行优化和添加新的特性。
