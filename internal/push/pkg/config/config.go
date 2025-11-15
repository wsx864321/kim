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

// GetPushServiceName 获取 Push 服务名称
func GetPushServiceName() string {
	name := viper.GetString("push.service_name")
	if name == "" {
		return "kim-push"
	}
	return name
}

// GetPushServicePort 获取 Push 服务端口
func GetPushServicePort() int {
	port := viper.GetInt("push.port")
	if port <= 0 {
		return 9003
	}
	return port
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
		return "push.log"
	}
	return filename
}

// GetRegistryEndpoints 获取注册中心端点列表
func GetRegistryEndpoints() []string {
	return viper.GetStringSlice("registry.endpoints")
}

