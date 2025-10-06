package config

import "github.com/spf13/viper"

// GetRegistryEndpoints 获取discov的 endpoints
func GetRegistryEndpoints() []string {
	return viper.GetStringSlice("registry.endpoints")
}

// GetTraceEnable 是否开启trace
func GetTraceEnable() bool {
	return viper.GetBool("krpc.trace.enable")
}

// GetTraceCollectionUrl 获取trace collection url
func GetTraceCollectionUrl() string {
	return viper.GetString("krpc.trace.url")
}

// GetTraceServiceName 获取服务名
func GetTraceServiceName() string {
	return viper.GetString("krpc.trace.service_name")
}

// GetTraceSampler 获取trace采样率
func GetTraceSampler() float64 {
	return viper.GetFloat64("krpc.trace.sampler")
}
