package config

import "github.com/spf13/viper"

func Init(path string) {
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
}

// GetSessionServiceName 获取 Session 服务名称
func GetSessionServiceName() string {
	return viper.GetString("session.service_name")
}

// GetSessionServiceIP 获取 Session 服务 IP 地址
func GetSessionServiceIP() string {
	return viper.GetString("session.ip")
}

// GetSessionServicePort 获取 Session 服务端口
func GetSessionServicePort() int {
	return viper.GetInt("session.port")
}

// GetSessionServiceRedisEndpoint 获取 Session 服务 Redis 端点
func GetSessionServiceRedisEndpoint() string {
	return viper.GetString("session.redis.endpoint")
}

// GetSessionServiceRedisPassword 获取 Session 服务 Redis 密码
func GetSessionServiceRedisPassword() string {
	return viper.GetString("session.redis.password")
}

// GetSessionServiceRedisDB 获取 Session 服务 Redis 数据库编号
func GetSessionServiceRedisDB() int {
	return viper.GetInt("session.redis.db")
}

// GetSessionServiceRedisPoolSize 获取 Session 服务 Redis 连接池大小
func GetSessionServiceRedisPoolSize() int {
	poolSize := viper.GetInt("session.redis.pool_size")
	if poolSize <= 0 {
		return 10 // 默认值
	}
	return poolSize
}

// GetSessionServiceRedisMinIdleConns 获取 Session 服务 Redis 最小空闲连接数
func GetSessionServiceRedisMinIdleConns() int {
	minIdleConns := viper.GetInt("session.redis.min_idle_conns")
	if minIdleConns <= 0 {
		return 5 // 默认值
	}
	return minIdleConns
}

// GetLogDebug 获取日志 Debug 模式配置
func GetLogDebug() bool {
	return viper.GetBool("log.debug")
}

// GetLogDir 获取日志目录
func GetLogDir() string {
	dir := viper.GetString("log.dir")
	if dir == "" {
		return "/home/www/logs/applogs" // 默认值
	}
	return dir
}

// GetLogFilename 获取日志文件名
func GetLogFilename() string {
	filename := viper.GetString("log.filename")
	if filename == "" {
		return "default.log" // 默认值
	}
	return filename
}
