package config

import "github.com/spf13/viper"

// Init 初始化配置
func Init(path string) {
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
}

// GetGatewayServiceName 获取 Gateway 服务名称
func GetGatewayServiceName() string {
	name := viper.GetString("gateway.service_name")
	if name == "" {
		return "kim-gateway"
	}
	return name
}

// GetGatewayServicePort 获取 Gateway 服务端口（gRPC端口）
func GetGatewayServicePort() int {
	port := viper.GetInt("gateway.port")
	if port <= 0 {
		return 9002
	}
	return port
}

// GetGatewayTCPPort 获取 Gateway TCP 端口
func GetGatewayTCPPort() int {
	port := viper.GetInt("gateway.tcp_port")
	if port <= 0 {
		return 8080
	}
	return port
}

// GetGatewayID 获取 Gateway 节点ID
func GetGatewayID() string {
	id := viper.GetString("gateway.gateway_id")
	if id == "" {
		return "gateway-1"
	}
	return id
}

// GetHeartbeatTimeout 获取心跳超时时间（秒）
func GetHeartbeatTimeout() int {
	timeout := viper.GetInt("gateway.heartbeat_timeout")
	if timeout <= 0 {
		return 180 // 默认3分钟
	}
	return timeout
}

// GetRefreshTTLInterval 获取刷新TTL间隔（秒）
func GetRefreshTTLInterval() int {
	interval := viper.GetInt("gateway.refresh_ttl_interval")
	if interval <= 0 {
		return 60 // 默认60秒
	}
	return interval
}

// GetNumWorkers 获取工作协程数量
func GetNumWorkers() int {
	workers := viper.GetInt("gateway.num_workers")
	if workers <= 0 {
		return 0 // 0表示使用默认值（2 * CPU核心数）
	}
	return workers
}

// GetLogDebug 获取日志 Debug 模式配置
func GetLogDebug() bool {
	return viper.GetBool("log.debug")
}

// GetLogDir 获取日志目录
func GetLogDir() string {
	dir := viper.GetString("log.dir")
	if dir == "" {
		return "/home/www/logs/applogs"
	}
	return dir
}

// GetLogFilename 获取日志文件名
func GetLogFilename() string {
	filename := viper.GetString("log.filename")
	if filename == "" {
		return "gateway.log"
	}
	return filename
}

// GetRegistryEndpoints 获取注册中心端点列表
func GetRegistryEndpoints() []string {
	return viper.GetStringSlice("registry.endpoints")
}

