## PRPC

plato rpc简称PRCP，其底层依赖GRPC，支持服务发现、服务注册、负载均衡、服务治理（metric、trace、限流、熔断）等等功能。各位可以根据自己的
想法在此基础上进行优化和添加新的特性。


整体加架构图:

<img src="doc/imgs/img.png" width="500px" height="300px">

### RPC介绍
RPC(Remote Procedure Call)：远程过程调用，它是一种通过网络从远程计算机程序上请求服务，而不需要了解底层网络技术的思想。

通常在同一服务内部我们采用函数调用的方式进行通信，那么如果在服务与服务之间就需要用到RPC了。
<img src="doc/imgs/img_3.png" width="500px" height="300px">

### 协议传输
因为prpc是基于GRPC的，GRPC默认情况下采用的是http2协议，因此prpc的协议采用的自然就是http2。

http2相关文档介绍：

- [HTTP2协议解析](https://www.jianshu.com/p/42ca44202ca4)

GRPC底层协议介绍：

- [gRPC系列(三) 如何借助HTTP2实现传输](https://zhuanlan.zhihu.com/p/161577635)


### 数据序列化

还是因为PRPC是基于GRPC的，因为GRPC采用的protocol buffer，那么自然prpc也是采用protocol buffer进行的数据序列化。

protocol buffer介绍：

-[Protocol Buffer 简介] (https://blog.csdn.net/mzpmzk/article/details/80824839)

protocol buffer原理介绍：

- [Protocol Buffer原理解密](https://juejin.cn/post/6844903997292150791)

### 服务治理
#### 数据可观测性
#### 熔断
熔断基于的是sony/gobreaker

#### 限流
限流采用的是基于juju/ratelimit做的，常用的限流算法有漏斗算法和令牌桶算法。

算法介绍：

-[流量控制算法——漏桶算法和令牌桶算法](https://zhuanlan.zhihu.com/p/165006444)

这里还给大家介绍一种自适应过载保护的算法，bbr limit，QUIC在弱网环境比较优秀的比较重要的一个原因就是拥塞控制使用了bbr算法代替了之前Reno（New Reno）

bbr limit:

- [https://segmentfault.com/a/1190000041950209?sort=votes]https://segmentfault.com/a/1190000041950209?sort=votes

